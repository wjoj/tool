package trace

import (
	"fmt"
	"strings"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	tracerLog "github.com/opentracing/opentracing-go/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

//_ = db.Use(&trace.OpenTracingPlugin{})

const (
	callBackBeforeName = "opentracing:before"
	callBackAfterName  = "opentracing:after"
)

type OpenTracingPlugin struct{}

var _ gorm.Plugin = &OpenTracingPlugin{}

func (op *OpenTracingPlugin) Name() string {
	return "openTracingPlugin"
}

func (op *OpenTracingPlugin) Initialize(db *gorm.DB) (err error) {
	// 开始前 - 并不是都用相同的方法，可以自己自定义
	db.Callback().Create().Before("gorm:before_create").Register(callBackBeforeName, before)
	db.Callback().Query().Before("gorm:query").Register(callBackBeforeName, before)
	db.Callback().Delete().Before("gorm:before_delete").Register(callBackBeforeName, before)
	db.Callback().Update().Before("gorm:setup_reflect_value").Register(callBackBeforeName, before)
	db.Callback().Row().Before("gorm:row").Register(callBackBeforeName, before)
	db.Callback().Raw().Before("gorm:raw").Register(callBackBeforeName, before)

	// 结束后 - 并不是都用相同的方法，可以自己自定义
	db.Callback().Create().After("gorm:after_create").Register(callBackAfterName, after)
	db.Callback().Query().After("gorm:after_query").Register(callBackAfterName, after)
	db.Callback().Delete().After("gorm:after_delete").Register(callBackAfterName, after)
	db.Callback().Update().After("gorm:after_update").Register(callBackAfterName, after)
	db.Callback().Row().After("gorm:row").Register(callBackAfterName, after)
	db.Callback().Raw().After("gorm:raw").Register(callBackAfterName, after)
	return
}

const _GormSpan = "_GormSpan"

func before(db *gorm.DB) {

	if !opentracing.IsGlobalTracerRegistered() {
		return
	}

	operationName := fmt.Sprintf("Mysql - %s", db.Statement.Schema.Table)

	span, _ := opentracing.StartSpanFromContext(db.Statement.Context, operationName)

	span.SetTag(string(ext.DBType), "sql")
	span.SetTag("db.table", db.Statement.Schema.Table)

	a, ok := db.Statement.Config.Dialector.(*mysql.Dialector)
	if ok {
		index := strings.Index(a.DSN, "tcp(")
		span.SetTag(string(ext.DBInstance), a.DSN[index:])
	}

	// 记录当前span
	db.InstanceSet(_GormSpan, span)

}
func after(db *gorm.DB) {

	_span, isExist := db.InstanceGet(_GormSpan)
	if !isExist {
		return
	}

	span, ok := _span.(opentracing.Span)
	if !ok {
		return
	}

	defer span.Finish()

	// Error
	if db.Error != nil {
		ext.Error.Set(span, true)
		span.LogFields(tracerLog.Error(db.Error))
	}

	// 记录sql
	span.SetTag(string(ext.DBStatement), db.Dialector.Explain(db.Statement.SQL.String(), db.Statement.Vars...))
	span.LogFields(tracerLog.String("sql", db.Dialector.Explain(db.Statement.SQL.String(), db.Statement.Vars...)))

	// 记录影响行数
	span.SetTag("db.count", db.RowsAffected)

	// 截取 sql 表示记录方法
	span.SetTag("db.method", strings.ToUpper(strings.Split(db.Statement.SQL.String(), " ")[0]))

}
