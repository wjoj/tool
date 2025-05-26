package dbx

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/wjoj/tool/log"
	"gorm.io/gorm"
)

var regfmt, _ = regexp.Compile(`(%[vsfdx0-9.]{1,}){1,}`)

type funcName struct {
	funcNameMaps   map[string]struct{}
	fieldFuncNames map[string]funcName
}

func (f funcName) IsFuncName(fname string) bool {
	if len(f.funcNameMaps) == 0 {
		return true
	}
	if _, is := f.funcNameMaps[fname]; is {
		return true
	}
	return false
}
func (f funcName) IsNotFuncName(fname string) bool {
	if len(f.funcNameMaps) == 0 {
		return false
	}
	if _, is := f.funcNameMaps[fname]; is {
		return true
	}
	return false
}
func (f funcName) GetFuncName(field string) funcName {
	if len(f.fieldFuncNames) == 0 {
		return funcName{
			funcNameMaps:   make(map[string]struct{}),
			fieldFuncNames: map[string]funcName{},
		}
	}
	fName, is := f.fieldFuncNames[field]
	if !is {
		fName = funcName{
			funcNameMaps:   make(map[string]struct{}),
			fieldFuncNames: map[string]funcName{},
		}
	}
	return fName
}

type KeyClauseType string

const (
	KeyClauseTypeWhere   KeyClauseType = "where"
	KeyClauseTypeOrderBy KeyClauseType = "orderBy"
	KeyClauseTypeGroup   KeyClauseType = "group"
	KeyClauseTypeHaving  KeyClauseType = "having"
	KeyClauseTypeJoins   KeyClauseType = "joins"
)

// QueryConditionOptions Query condition configuration
type QueryConditionOptions struct {
	tag          string                     //重置tag
	funcNames    funcName                   //加入条件的函数(使用名称)
	funcs        []func(*gorm.DB) *gorm.DB  //加入条件的函数
	notFuncNames funcName                   //不加入条件的函数
	keyClause    map[KeyClauseType]struct{} //加入条件的关键子句
}

type QueryConditionOption func(c *QueryConditionOptions)

// WithQueryConditionOptionTag Reset tag
func WithQueryConditionOptionTag(tag string) QueryConditionOption {
	return func(c *QueryConditionOptions) {
		c.tag = tag
	}
}

// WithQueryConditionOptionKeyClause Reset keyClause
func WithQueryConditionOptionKeyClause(keyClause map[KeyClauseType]struct{}) QueryConditionOption {
	return func(c *QueryConditionOptions) {
		c.keyClause = keyClause
	}
}

// WithQueryConditionOptionKeyClauseOrderBy Only use OrderBy
func WithQueryConditionOptionKeyClauseOrderBy() QueryConditionOption {
	return func(c *QueryConditionOptions) {
		c.keyClause[KeyClauseTypeOrderBy] = struct{}{}
	}
}

// WithQueryConditionOptionKeyClauseNotOrderBy Do not use OrderBy
func WithQueryConditionOptionKeyClauseNotOrderBy() QueryConditionOption {
	return func(c *QueryConditionOptions) {
		c.keyClause[KeyClauseTypeWhere] = struct{}{}
		c.keyClause[KeyClauseTypeGroup] = struct{}{}
		c.keyClause[KeyClauseTypeHaving] = struct{}{}
		c.keyClause[KeyClauseTypeJoins] = struct{}{}
	}
}

// funcNames: Use the name, such as：func1,Feild.func2,Feild2.Feild.func(字段名:Feild* 方法名:func*)
// funcNames: The usage method structure is as followsfunc(db *gorm.DB) *gorm.DB, 如:  OrderByC(bool) func(db *gorm.DB) *gorm.DB 或 OrderByC(db *gorm.DB) *gorm.DB
// WithQueryConditionOptionFuncNames Reset the method name (the name of the method to be added for the query condition)
func WithQueryConditionOptionFuncNames(funcNames ...any) QueryConditionOption {
	funcNamesc := &funcName{
		funcNameMaps:   map[string]struct{}{},
		fieldFuncNames: map[string]funcName{},
	}
	funcs := []func(*gorm.DB) *gorm.DB{}
	for _, fnames := range funcNames {
		switch fname := fnames.(type) {
		case func(*gorm.DB) *gorm.DB:
			funcs = append(funcs, fname)
		case string:
			analysisQueryConditionFuncNames(strings.Split(strings.TrimSpace(fname), "."), funcNamesc)
		default:
			log.Warnf("unsupported function:%+v", reflect.TypeOf(fnames))
		}
	}
	return func(c *QueryConditionOptions) {
		c.funcNames = *funcNamesc
		c.funcs = funcs
	}
}

// notFuncNames: as：func1,Feild.func2,Feild2.Feild.func(字段名:Feild* 方法:func*)
// WithQueryConditionOptionNotFuncNames Reset the method names that are not included (the method names of the queries that are not included)
func WithQueryConditionOptionNotFuncNames(notFuncNames ...any) QueryConditionOption {
	funcNamesc := &funcName{
		funcNameMaps:   map[string]struct{}{},
		fieldFuncNames: map[string]funcName{},
	}
	for _, fname := range notFuncNames {
		kind := reflect.TypeOf(fname).Kind()
		switch kind {
		case reflect.String:
			analysisQueryConditionFuncNames(strings.Split(strings.TrimSpace(fname.(string)), "."), funcNamesc)
		case reflect.Func:
			name := runtime.FuncForPC(reflect.ValueOf(fname).Pointer()).Name()
			names := strings.Split(name, ".")
			var cpnames []string
			if strings.HasSuffix(names[len(names)-1], "-fm") {
				cpnames = []string{strings.ReplaceAll(names[len(names)-1], "-fm", "")}
			} else if strings.HasPrefix(names[len(names)-1], "func") {
				cpnames = []string{names[len(names)-2]}
			}
			analysisQueryConditionFuncNames(cpnames, funcNamesc)
		default:
			log.Warnf("(not func)unsupported type:%+v", kind)
		}
	}
	return func(c *QueryConditionOptions) {
		c.notFuncNames = *funcNamesc
	}
}

// WithQueryConditionOption opt Reset the method name for addition
func WithQueryConditionOptionFuncName(opt funcName) QueryConditionOption {
	return func(c *QueryConditionOptions) {
		c.funcNames = opt
	}
}

// WithQueryConditionOptionNotFuncName opt Reset the method name without adding it.
func WithQueryConditionOptionNotFuncName(opt funcName) QueryConditionOption {
	return func(c *QueryConditionOptions) {
		c.notFuncNames = opt
	}
}

// WithQueryConditionOption opt
func WithQueryConditionOption(opt QueryConditionOptions) QueryConditionOption {
	return func(c *QueryConditionOptions) {
		*c = opt
	}
}

func applyQueryConditionOptions(options ...QueryConditionOption) QueryConditionOptions {
	opts := QueryConditionOptions{
		tag: "query", //默认
		funcNames: funcName{
			funcNameMaps:   map[string]struct{}{},
			fieldFuncNames: map[string]funcName{},
		},
		notFuncNames: funcName{
			funcNameMaps:   map[string]struct{}{},
			fieldFuncNames: map[string]funcName{},
		},
		keyClause: map[KeyClauseType]struct{}{},
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		option(&opts)
	}
	if len(opts.keyClause) == 0 {
		opts.keyClause = map[KeyClauseType]struct{}{
			KeyClauseTypeWhere:   {},
			KeyClauseTypeOrderBy: {},
			KeyClauseTypeGroup:   {},
			KeyClauseTypeHaving:  {},
			KeyClauseTypeJoins:   {},
		}
	}
	return opts
}
func analysisQueryConditionFuncNames(fs []string, fc *funcName) {
	if len(fs) == 0 {
		return
	}
	if len(fs) == 1 {
		fc.funcNameMaps[fs[0]] = struct{}{}
		return
	}
	fieldFs, is := fc.fieldFuncNames[fs[0]]
	if !is {
		fieldFs = funcName{
			funcNameMaps:   make(map[string]struct{}),
			fieldFuncNames: make(map[string]funcName),
		}
		fc.fieldFuncNames[fs[0]] = fieldFs
	}
	if len(fs) == 2 {
		fieldFs.funcNameMaps[fs[1]] = struct{}{}
		return
	}
	fieldFs2, is := fieldFs.fieldFuncNames[fs[1]]
	if !is {
		fieldFs2 = funcName{
			funcNameMaps:   make(map[string]struct{}),
			fieldFuncNames: make(map[string]funcName),
		}
		fieldFs.fieldFuncNames[fs[1]] = fieldFs2
	}
	analysisQueryConditionFuncNames(fs[2:], &fieldFs2)
}

// item: tag例子
//
// 例子1  field type `query` 默认sql语句 field = ?
// 例子1  field type `query:"or"` 默认sql语句 or field = ?
// 例子2  field type `query:"field like '%%%v%%'"`
// 例子2  field type `query:"field like '%%%v%%';or"`
// 例子3  field type `query:"field like '%%%v%%';or;order"`
// 例子4  field type `query:"field like '%%%v%%';or;order desc;group"`
// 例子5  field type `query:"field=?;or;order field desc;group field"`
// 例子6  field type `query:"field;or;order field desc;group field"`
// 例子7  field type `query:"order field desc;joins:left join table1 on table1.xx=?"`
// item: 对象 字段的tag=query(默认)加入查询条件 值为0或空字符串不加入查询(要使用0或空字符串使用指针)
// item: 对象定义的方法 结构为func(*gorm.DB) *gorm.DB,方法名任意 加入查询条件(注意方法定义推荐不使用指针)
// item: 对象的字段对应的类型,存在Value方法(必须有一个返回值), 这个字段的值为Value方法返回的值
// item: 对象转换方法UseMap ,返回值 map[string]any(字段名对应的值)
// opts: 重设tag标签名 或 指定加入搜索的方法名(结构为func(*gorm.DB) *gorm.DB),未指定方法名会组合所有结构为func(*gorm.DB) *gorm.DB的方法
// QueryConditions 拼接查询条件
func QueryConditions(item any, opts ...QueryConditionOption) func(db *gorm.DB) *gorm.DB {
	opt := applyQueryConditionOptions(opts...)
	ty := reflect.TypeOf(item)
	val := reflect.ValueOf(item)
	ty2 := ty
	val2 := val
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if !val.IsValid() {
		return func(db *gorm.DB) *gorm.DB {
			return db
		}
	}
	var useMap map[string]any
	var is bool
	valuef := val2.MethodByName("UseMap")
	if valuef.IsValid() {
		results := valuef.Call([]reflect.Value{})
		if len(results) != 1 {
			log.Warn("the UseMap method must have a map[string]any")
		} else {
			useMap, is = results[0].Interface().(map[string]any)
			if !is {
				log.Warn("the UseMap method must have a map[string]any")
				useMap = make(map[string]any)
			}
		}
	}
	return func(db *gorm.DB) *gorm.DB {
		for i := range opt.funcs {
			db = opt.funcs[i](db)
		}
		for i := range ty2.NumMethod() {
			fcname := ty2.Method(i).Name
			if !opt.funcNames.IsFuncName(fcname) {
				continue
			}
			if opt.notFuncNames.IsNotFuncName(fcname) {
				continue
			}
			fnc := val2.MethodByName(fcname)
			switch fnc.Type().String() {
			case "func(*gorm.DB) *gorm.DB":
				results := fnc.Call([]reflect.Value{reflect.ValueOf(db)})
				if len(results) == 0 {
					log.Warnf("the return value of the %s function must be either *gorm.DB or func(*gorm.DB) *gorm.DB", fcname)
					continue
				}
				switch results[0].Type().String() {
				case "*gorm.DB":
					db = results[0].Interface().(*gorm.DB)
				default:
					log.Warnf("the return value of the %s function must be either *gorm.DB or func(*gorm.DB) *gorm.DB", fcname)
					continue
				}
			}
		}
		for i := range ty.NumField() {
			field := ty.Field(i)
			fieldty := field.Type
			if fieldty.Kind() == reflect.Ptr {
				fieldty = fieldty.Elem()
			}
			tagVal, is := field.Tag.Lookup(opt.tag)
			if !is {
				continue
			}
			if tagVal == "-" {
				continue
			}
			switch fieldty.Kind() {
			case reflect.Struct:
				val := val.FieldByName(field.Name)
				if !val.IsValid() || val.IsZero() {
					continue
				}
				db = QueryConditions(
					val.Interface(),
					WithQueryConditionOptionTag(opt.tag),
					WithQueryConditionOptionFuncName(opt.funcNames.GetFuncName(field.Name)),
					WithQueryConditionOptionNotFuncName(opt.notFuncNames.GetFuncName(field.Name)),
					WithQueryConditionOptionKeyClause(opt.keyClause),
				)(db)
				continue
			}
			tagCon := analyzeQueryConditionsTagVal(tagVal, field.Name, opt.keyClause)
			db = dbSpliceSqlOrderGroup(db, tagCon)
			val := val.FieldByName(field.Name)
			if !val.IsValid() || val.IsZero() {
				continue
			}
			valuef := val.MethodByName("Value")
			if valuef.IsValid() {
				results := valuef.Call([]reflect.Value{})
				if len(results) != 1 {
					log.Warnf("the value method of %s does not return a value", field.Name)
				} else {
					val = results[0]
				}
			} else if vl, is := useMap[field.Name]; is {
				val = reflect.ValueOf(vl)
			}
			if !val.IsValid() || val.IsZero() {
				continue
			}
			db = dbSpliceSqlVal(db, tagCon, val.Interface())
		}
		return db
	}
}

type tagQueryConditions struct {
	Where     string
	Ifc       string
	Order     string
	Group     string
	Having    string
	Joins     string
	FieldName string
}

func analyzeQueryConditionsTagVal(tagVal string, fieldName string, keyClause map[KeyClauseType]struct{}) (tag tagQueryConditions) {
	tag = tagQueryConditions{}
	tagVals := strings.Split(tagVal, ";")
	tag.FieldName = strcase.ToSnake(fieldName)
	for i := range tagVals {
		val := tagVals[i]
		vall := strings.TrimSpace(strings.ToLower(val))
		if _, is := keyClause[KeyClauseTypeWhere]; is && slices.Contains([]string{"or", "not", "and"}, vall) {
			tag.Ifc = vall
		} else if _, is := keyClause[KeyClauseTypeOrderBy]; is && strings.HasPrefix(vall, "order") {
			tag.Order = strings.Replace(strings.Replace(vall, "order", "", 1), " ", "", -1)
			if tag.Order == "desc" {
				tag.Order = tag.FieldName + " desc"
			} else if tag.Order == "asc" {
				tag.Order = tag.FieldName + " asc"
			}
			if len(tag.Order) == 0 {
				tag.Order = tag.FieldName
			}
		} else if _, is := keyClause[KeyClauseTypeGroup]; is && strings.HasPrefix(vall, "group") {
			tag.Group = strings.Replace(vall, "group", "", 1)
			if len(tag.Group) == 0 {
				tag.Group = tag.FieldName
			}
		} else if _, is := keyClause[KeyClauseTypeHaving]; is && strings.HasPrefix(vall, "having") {
			tag.Having = strings.Replace(vall, "having", "", 1)
			if len(tag.Having) == 0 {
				tag.Having = tag.FieldName
			}
		} else if _, is := keyClause[KeyClauseTypeJoins]; is && strings.HasPrefix(vall, "joins") {
			tag.Joins = strings.Replace(vall, "joins", "", 1)
			if len(tag.Joins) == 0 {
				log.Warn("joins statements cannot be empty")
			}
		} else if _, is := keyClause[KeyClauseTypeWhere]; is {
			tag.Where = val
		}
	}
	if len(tag.Where) == 0 && ((len(tag.Having) == 0 && len(tag.Group) == 0 &&
		len(tag.Joins) == 0 && len(tag.Order) == 0) || len(tag.Ifc) > 0) {
		tag.Where = tag.FieldName
	}
	return
}

func dbSpliceSqlOrderGroup(db *gorm.DB, tag tagQueryConditions) *gorm.DB {
	if len(tag.Order) > 0 {
		db = db.Order(tag.Order)
	}
	if len(tag.Group) > 0 {
		db = db.Group(tag.Group)
	}
	return db
}

// dbSpliceSql 拼接sql
func dbSpliceSqlVal(db *gorm.DB, tag tagQueryConditions, val any) *gorm.DB {
	if len(tag.Joins) > 0 {
		if num := strings.Count(tag.Joins, "?"); num > 0 {
			db = db.Having(tag.Joins, slices.Repeat([]any{val}, num)...)
		} else if nums := regfmt.FindAllStringIndex(tag.Joins, -1); len(nums) > 0 {
			db = db.Having(fmt.Sprintf(tag.Joins, slices.Repeat([]any{val}, len(nums))...))
		} else {
			db = db.Having(tag.Joins+" = ?", val)
		}
	}
	if len(tag.Where) > 0 {
		var wheref func(query any, args ...any) (tx *DB)
		if tag.Ifc == "or" {
			wheref = db.Or
		} else if tag.Ifc == "not" {
			wheref = db.Not
		} else {
			wheref = db.Where
		}
		if num := strings.Count(tag.Where, "?"); num > 0 {
			db = wheref(tag.Where, slices.Repeat([]any{val}, num)...)
		} else if nums := regfmt.FindAllStringIndex(tag.Where, -1); len(nums) > 0 {
			db = wheref(fmt.Sprintf(tag.Where, slices.Repeat([]any{val}, len(nums))...))
		} else {
			db = wheref(tag.Where+" = ?", val)
		}
	}

	if len(tag.Having) > 0 {
		if num := strings.Count(tag.Having, "?"); num > 0 {
			db = db.Having(tag.Having, slices.Repeat([]any{val}, num)...)
		} else if nums := regfmt.FindAllStringIndex(tag.Having, -1); len(nums) > 0 {
			db = db.Having(fmt.Sprintf(tag.Having, slices.Repeat([]any{val}, len(nums))...))
		} else {
			db = db.Having(tag.Having+" = ?", val)
		}
	}
	return db
}
