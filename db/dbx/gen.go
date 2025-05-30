package dbx

import (
	"github.com/wjoj/tool/v2/utils"
	"gorm.io/gen"
	"gorm.io/gorm"
)

type TableModelOpt struct {
	Table          string
	ImportPkgPaths []string
	ModelOpts      []gen.ModelOpt
}
type GenDBInfo struct {
	OutPath           string
	ModelPkgPath      string
	Module            string
	PkgName           string
	FieldNullable     bool
	ModelTypePkgPaths []string
	TableModelOpts    []*TableModelOpt
}

type GenOptions struct {
	defKey *utils.DefaultKeys
	module string
	infos  map[string]*GenDBInfo
}

type GenOption func(c *GenOptions)

// 设置默认key
func WithDefaultKeyGenOption(key string) GenOption {
	return func(c *GenOptions) {
		c.defKey.DefaultKey = key
	}
}

// 设置要使用配置文件的key
func WithLogConfigKeysGenOption(keys ...string) GenOption {
	return func(c *GenOptions) {
		c.defKey.Keys = keys
	}
}

func WithGenModuleGenOption(module string) GenOption {
	return func(c *GenOptions) {
		c.module = module
	}
}

// 设置要使用配置文件的key
func WithGenDBInfoGenOption(key string, info *GenDBInfo) GenOption {
	return func(c *GenOptions) {
		c.infos[key] = info
	}
}

func applyGenOptions(options ...GenOption) GenOptions {
	opts := GenOptions{
		defKey: utils.DefaultKey,
		infos:  map[string]*GenDBInfo{},
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		option(&opts)
	}
	return opts
}

func GenByGorm(options ...GenOption) {
	opt := applyGenOptions(options...)

	keys := opt.defKey.Keys
	if len(keys) == 0 {
		for key := range dbs {
			keys = append(keys, key)
		}
	}
	for _, key := range keys {
		info, is := opt.infos[key]
		if !is {
			info = &GenDBInfo{
				FieldNullable: false,
			}
		}
		if len(info.PkgName) == 0 {
			info.PkgName = key
		}
		module := opt.module
		if len(module) == 0 {
			module = "models/" + info.PkgName
		} else {
			module = module + "/" + info.PkgName
		}
		if len(info.OutPath) == 0 {
			info.OutPath = "./models/" + info.PkgName + "/dao"
		}
		if len(info.ModelPkgPath) == 0 {
			info.ModelPkgPath = "./models/" + info.PkgName
		}
		genb := gen.NewGenerator(gen.Config{
			OutPath:           info.OutPath,
			ModelPkgPath:      info.ModelPkgPath,
			Mode:              gen.WithDefaultQuery | gen.WithQueryInterface,
			FieldNullable:     info.FieldNullable,
			FieldCoverable:    false,
			FieldSignable:     false,
			FieldWithIndexTag: false,
			FieldWithTypeTag:  true,
		})

		genb.WithDataTypeMap(map[string]func(detailType gorm.ColumnType) (dataType string){
			"tinyint":   func(detailType gorm.ColumnType) (dataType string) { return "int" },
			"smallint":  func(detailType gorm.ColumnType) (dataType string) { return "int" },
			"mediumint": func(detailType gorm.ColumnType) (dataType string) { return "int64" },
			"bigint":    func(detailType gorm.ColumnType) (dataType string) { return "int64" },
			"int":       func(detailType gorm.ColumnType) (dataType string) { return "int" },
			"float":     func(detailType gorm.ColumnType) (dataType string) { return "float64" },
			"json":      func(detailType gorm.ColumnType) (dataType string) { return "datatypes.JSON" }, // 自定义时间
			"timestamp": func(detailType gorm.ColumnType) (dataType string) { return "typesx.Time" },    // 自定义时间
			"datetime":  func(detailType gorm.ColumnType) (dataType string) { return "typesx.Time" },    // 自定义时间
			"date":      func(detailType gorm.ColumnType) (dataType string) { return "typesx.Date" },    // 自定义时间
			"decimal":   func(detailType gorm.ColumnType) (dataType string) { return "typesx.Decimal" }, // 金额类型全部转换为第三方库,github.com/shopspring/decimal
		})
		genb.WithImportPkgPath(append([]string{
			"github.com/wjoj/tool/v2/typesx",
			"github.com/wjoj/tool/v2/utils",
			"github.com/shopspring/decimal",
			"gorm.io/datatypes",
			module,
		}, info.ModelTypePkgPaths...)...)

		genb.UseDB(Get(key))
		genb.ApplyBasic(genb.GenerateAllTable(setModelOpts()...)...)
		for _, mopt := range info.TableModelOpts {
			gm := genb.GenerateModel(mopt.Table,
				append(setModelOpts(),
					mopt.ModelOpts...,
				)...,
			)
			if len(mopt.ImportPkgPaths) != 0 {
				gm.ImportPkgPaths = append(gm.ImportPkgPaths, mopt.ImportPkgPaths...)
			}
		}
		genb.Execute()
	}
}

func setModelOpts() []gen.ModelOpt {
	jsonField := gen.FieldJSONTagWithNS(func(columnName string) (tagContent string) {
		if columnName == "deleted_at" {
			return "-"
		} else if columnName == "password" {
			return "-"
		}
		return columnName
	})
	return []gen.ModelOpt{
		jsonField,
		gen.FieldType("deleted_at", "gorm.DeletedAt"),
		gen.FieldJSONTag("deleted_at", "-"),
	}
}
