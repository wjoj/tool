package tool

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type WhereOrderType int

const (
	WhereOrderTypeNot  WhereOrderType = -1
	WhereOrderTypeASC                 = 0
	WhereOrderTypeDesc                = 1
)

func (m WhereOrderType) Key(name string) string {
	if m == WhereOrderTypeASC {
		return name + " asc"
	} else if m == WhereOrderTypeDesc {
		return name + " desc"
	}
	return ""
}

type whereIfs struct {
	Value    string
	Key      string
	Ifs      string
	NotValue string
}

type WhereStructure struct {
	whs  map[string]*whereIfs
	lock sync.RWMutex
}

func whereStructureFuncIfs(m interface{}, tag string, funcIfs func(tag reflect.StructTag, key string) string) *WhereStructure {
	if len(tag) == 0 {
		tag = "wh"
	}
	if funcIfs == nil {
		panic(fmt.Errorf("funcIfs can't be empty"))
	}
	immutable := reflect.ValueOf(m)
	if immutable.Kind() == reflect.Ptr {
		immutable = immutable.Elem()
	}
	typeIm := reflect.TypeOf(m)
	if typeIm.Kind() == reflect.Ptr {
		typeIm = typeIm.Elem()
	}
	lng := typeIm.NumField()
	fileds := make(map[string]*whereIfs, lng)
	for i := 0; i < lng; i++ {
		if len(tag) == 0 {
			continue
		}
		field := typeIm.Field(i)
		tags := strings.Split(tag, " ")
		if len(tags) == 0 {
			continue
		}
		key := field.Tag.Get(tags[0])
		if len(key) == 0 || key == "-" {
			continue
		}

		if len(tags) > 1 && len(tags[1]) != 0 {
			keys := strings.Split(key, ";")
			if len(keys) == 0 {
				continue
			}
			for _, k := range keys {
				kvars := strings.Split(k, ":")
				if len(kvars) == 0 {
					continue
				}
				if kvars[0] == tags[1] && (len(kvars) > 1 && len(kvars[1]) != 0) {
					key = kvars[1]
					break
				}
			}
		}

		info := &whereIfs{}
		info.Key = key
		val := immutable.FieldByName(field.Name)
		if val.Kind() == reflect.String {
			info.Value = fmt.Sprintf("'%v'", val)
		} else {
			info.Value = fmt.Sprintf("%v", val)
		}
		info.Ifs = funcIfs(field.Tag, key)
		fileds[key] = info
	}
	return &WhereStructure{
		whs: fileds,
	}
}

//NewWhereStructureFuncIfs
//m is the structure
//tag example：`gorm:"column:name"` at this time tag="gorm column" or `gorm:"name"` at this time tag="gorm"
//funcIfs returns the tag value
func NewWhereStructureFuncIfs(m interface{}, tag string, funcIfs func(key string) string) *WhereStructure {
	return whereStructureFuncIfs(m, tag, func(_ reflect.StructTag, key string) string {
		return funcIfs(key)
	})
}

//NewWhereStructure
//m is the structure
//tag example：`gorm:"column:name"` at this time tag="gorm column" or `gorm:"name"` at this time tag="gorm"
//tagIfs  `ifs:"="` at this time  where is name=value
func NewWhereStructure(m interface{}, tag, tagIfs string) *WhereStructure {
	if len(tagIfs) == 0 {
		tagIfs = "ifs"
	}
	return whereStructureFuncIfs(m, tag, func(tag reflect.StructTag, _ string) string {
		return tag.Get(tagIfs)
	})
}

func (m *WhereStructure) SetNotValue(key, notVal string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	v, is := m.whs[key]
	if !is {
		return
	}
	v.NotValue = notVal
}

func (m *WhereStructure) AddStructure(m2 *WhereStructure) {
	if m2 == nil {
		return
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	for k, val := range m2.whs {
		m.whs[k] = val
	}
}

func (m *WhereStructure) AddFiled(key, ifs, val string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.whs[key] = &whereIfs{
		Key:   key,
		Ifs:   ifs,
		Value: val,
	}
}

func (m *WhereStructure) GetWhereByKey(key string) *Where {
	m.lock.RLock()
	defer m.lock.RUnlock()
	mw, is := m.whs[key]
	if !is {
		return nil
	}
	wh := new(Where)
	if mw.Value == mw.NotValue {
		return wh
	}
	mw.Value = whereQuotes(mw.Value).(string)
	wh.Add(fmt.Sprintf("%v %v %v", mw.Key, mw.Ifs, mw.Value), "")
	return wh
}

func (m *WhereStructure) Where(ifs string) *Where {
	wh := new(Where)
	for _, val := range m.whs {
		if val.Value == val.NotValue {
			continue
		}
		val.Value = whereQuotes(val.Value).(string)
		wh.Add(fmt.Sprintf("%v %v %v", val.Key, val.Ifs, val.Value), ifs)
	}
	return wh
}

//Where Where
type Where struct {
	str strings.Builder
}

func (w *Where) String() string {
	return w.str.String()
}

func (w *Where) Add(str string, tag string) {
	if w.str.Len() == 0 {
		w.str.WriteString(str)
	} else {
		w.str.WriteString(tag)
		w.str.WriteString(str)
	}
}

//And
func (w *Where) And(str string) {
	w.Add(str, " and ")
}

func (w *Where) OR(str string) {
	w.Add(str, " or ")
}

func (w *Where) AndIf(key string, ifs string, val interface{}) {
	val = whereQuotes(val)
	w.And(fmt.Sprintf("%v%v%v", key, ifs, val))
}

func (w *Where) OrIf(key string, ifs string, val interface{}) {
	val = whereQuotes(val)
	w.OR(fmt.Sprintf("%v%v%v", key, ifs, val))
}

func (w *Where) AndIfNotVal(key string, ifs string, val interface{}, notVal interface{}) {
	if val == notVal {
		return
	}
	val = whereQuotes(val)
	w.And(fmt.Sprintf("%v %v %v", key, ifs, val))
}

func (w *Where) OrIfNotVal(key string, ifs string, val interface{}, notVal interface{}) {
	if val == notVal {
		return
	}
	val = whereQuotes(val)
	w.OR(fmt.Sprintf("%v %v %v", key, ifs, val))
}

func (w *Where) AndWhere(wh *Where) {
	if wh == nil {
		return
	}
	w.And(wh.String())
}

func (w *Where) ORWhere(wh *Where) {
	if wh == nil {
		return
	}
	w.OR(wh.String())
}

func (w *Where) Keys(keys []string, ifs string, val interface{}, tag string) {
	val = whereQuotes(val)
	var str strings.Builder
	for i, key := range keys {
		sql := fmt.Sprintf("%v%v%v", key, ifs, val)
		if i == 0 {
			str.WriteString(sql)
		} else {
			str.WriteString(tag)
			str.WriteString(sql)
		}
	}
	w.str.WriteString(str.String())
}

//AndKeys  andOr Value `or`` or `and``
func (w *Where) AndKeys(keys []string, ifs string, val interface{}, andOr string) {
	w.And("")
	w.Keys(keys, ifs, val, andOr)
}

//AndKeysNotVal  andOr Value `or`` or `and``
func (w *Where) AndKeysNotVal(keys []string, ifs string, val interface{}, notVal interface{}, andOr string) {
	if val == notVal {
		return
	}
	w.AndKeys(keys, ifs, val, andOr)
}

//ORKeys  andOr Value `or`` or `and``
func (w *Where) ORKeys(keys []string, ifs string, val interface{}, andOr string) {
	w.OR("")
	w.Keys(keys, ifs, val, andOr)
}

//ORKeysNotVal  andOr Value `or`` or `and``
func (w *Where) ORKeysNotVal(keys []string, ifs string, val interface{}, notVal interface{}, andOr string) {
	if val == notVal {
		return
	}
	w.ORKeys(keys, ifs, val, andOr)
}

//SonKeys  andOr Value `or`` or `and``
func (w *Where) SonKeys(keys []string, ifs string, val interface{}, andOr string) {
	w.str.WriteString("(")
	w.Keys(keys, ifs, val, andOr)
	w.str.WriteString(")")
}

//AndSonKeys  andOr Value `or`` or `and``
func (w *Where) AndSonKeys(keys []string, ifs string, val interface{}, andOr string) {
	w.And("")
	w.SonKeys(keys, ifs, val, andOr)
}

//ORSonKeys  andOr Value `or`` or `and``
func (w *Where) ORSonKeys(keys []string, ifs string, val interface{}, andOr string) {
	w.OR("")
	w.SonKeys(keys, ifs, val, andOr)
}

func (w *Where) AddWhereStructure(m *WhereStructure, ifs, ifs2 string) {
	if m == nil {
		return
	}
	w.Add(m.Where(ifs).String(), ifs2)
}

func (w *Where) AndWhereStructure(m *WhereStructure, ifs string) {
	if m == nil {
		return
	}
	w.AndWhere(m.Where(ifs))
}

func (w *Where) ORWhereStructure(m *WhereStructure, ifs string) {
	if m == nil {
		return
	}
	w.ORWhere(m.Where(ifs))
}

func whereQuotes(v interface{}) interface{} {
	switch ty := reflect.TypeOf(v); ty.Kind() {
	case reflect.String:
		if !IsDigit(v.(string)) {
			return fmt.Sprintf("'%v'", v)
		}
	}
	return v
}
