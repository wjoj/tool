package dbx

import (
	"context"
	"database/sql"
	"fmt"
	logs "log"
	"os"
	"time"

	"github.com/wjoj/tool/v2/log"
	"github.com/wjoj/tool/v2/utils"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type DB = gorm.DB
type LogLevelType string

const (
	LogLevelError  LogLevelType = "error"
	LogLevelWarn   LogLevelType = "warn"
	LogLevelInfo   LogLevelType = "info"
	LogLevelSilent LogLevelType = "silent"
)

func (l LogLevelType) GormLoggerLevel() logger.LogLevel {
	if l == LogLevelError {
		return logger.Error
	} else if l == LogLevelWarn {
		return logger.Warn
	} else if l == LogLevelInfo {
		return logger.Info
	} else if l == LogLevelSilent {
		return logger.Silent
	}
	return logger.Info
}

type DriverType string

const (
	DriverMySQL      DriverType = "mysql"
	DriverSQLite     DriverType = "sqlite3"
	DriverPostGres   DriverType = "postgres"
	DriverMsSQL      DriverType = "mssql"
	DriverSQLServer  DriverType = "sqlserver"
	DriverClickHouse DriverType = "clickhouse"
)

type Config struct {
	Driver          DriverType    `yaml:"driver" json:"driver"`
	Host            string        `yaml:"host" json:"host"`
	Port            int           `yaml:"port" json:"port"`
	User            string        `yaml:"user" json:"user"`
	Pass            string        `yaml:"password" json:"password"`
	DbName          string        `yaml:"dbname" json:"dbname"`
	Debug           bool          `yaml:"debug" json:"debug"`
	Prefix          string        `yaml:"prefix" json:"prefix"`
	Charset         string        `yaml:"charset" json:"charset"`
	MaxIdleConns    int           `yaml:"maxIdleConns" json:"maxIdleConns"`
	MaxOpenConns    int           `yaml:"maxOpenConns" json:"maxOpenConns"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime" json:"connMaxLifetime"`
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime" json:"connMaxIdleTime"`
	TimeOut         int           `json:"timeout" yaml:"timeout"`
	LogLevel        LogLevelType  `yaml:"logLevel" json:"logLevel"`
	LogName         string        `yaml:"logName" json:"logName"`
}

func New(cfg *Config) (*gorm.DB, error) {
	if len(cfg.LogLevel) == 0 {
		cfg.LogLevel = LogLevelInfo
	}
	if len(cfg.LogName) == 0 {
		cfg.LogName = utils.DefaultKey.DefaultKey
	}
	var dbDSN gorm.Dialector
	switch cfg.Driver {
	case DriverMySQL:
		if len(cfg.User) == 0 || len(cfg.Pass) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的账号或密码不能为空")
		}
		if len(cfg.DbName) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的数据库名不能为空")
		}
		if len(cfg.Charset) == 0 {
			cfg.Charset = "utf8mb4"
		}
		if len(cfg.Host) == 0 {
			cfg.Host = "127.0.0.1"
		}
		if cfg.Port == 0 {
			cfg.Port = 3306
		}
		dbDSN = mysql.New(mysql.Config{
			DSN: fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=%s&parseTime=True&loc=Local&allowNativePasswords=true",
				cfg.User,
				cfg.Pass,
				cfg.Host,
				cfg.Port,
				cfg.DbName,
				cfg.Charset,
			),
		})
	case DriverSQLite:
		dbDSN = sqlite.Open(fmt.Sprintf("%v.db", cfg.DbName))
	case DriverPostGres:
		if len(cfg.User) == 0 || len(cfg.Pass) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的账号或密码不能为空")
		}
		if len(cfg.DbName) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的数据库名不能为空")
		}
		if len(cfg.Host) == 0 {
			cfg.Host = "127.0.0.1"
		}
		if cfg.Port == 0 {
			cfg.Port = 5432
		}
		dbDSN = postgres.Open(fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s port=%d",
			cfg.Host, cfg.User, cfg.DbName, cfg.Pass, cfg.Port))
	case DriverMsSQL:
	case DriverSQLServer:
		if len(cfg.User) == 0 || len(cfg.Pass) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的账号或密码不能为空")
		}
		if len(cfg.DbName) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的数据库名不能为空")
		}
		if len(cfg.Host) == 0 {
			cfg.Host = "127.0.0.1"
		}
		if cfg.Port == 0 {
			cfg.Port = 1433
		}
		dbDSN = sqlserver.Open(fmt.Sprintf("sqlserver://%v:%v@%v:%v?database=%v",
			cfg.User, cfg.Pass, cfg.Host, cfg.Port, cfg.DbName))
	case DriverClickHouse:
		if len(cfg.User) == 0 || len(cfg.Pass) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的账号或密码不能为空")
		}
		if len(cfg.DbName) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的数据库名不能为空")
		}
		if len(cfg.Host) == 0 {
			cfg.Host = "127.0.0.1"
		}
		if cfg.Port == 0 {
			cfg.Port = 9000
		}
		dbDSN = clickhouse.Open(fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s?read_timeout=%d",
			cfg.User, cfg.Pass, cfg.Host, cfg.Port, cfg.DbName, cfg.TimeOut))
	default:
		if len(cfg.DbName) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的数据库名不能为空")
		}
		dbDSN = sqlite.Open(fmt.Sprintf("%v.db", cfg.DbName))
	}
	sch := schema.NamingStrategy{
		SingularTable: true,
	}
	if len(cfg.Prefix) != 0 {
		sch.TablePrefix = cfg.Prefix
	}
	dbConfig := &gorm.Config{
		NamingStrategy: sch,
	}
	if len(cfg.LogName) != 0 {
		var out logger.Writer
		if cfg.LogName == "--" {
			out = logs.New(os.Stdout, "\r\n", logs.LstdFlags)
			dbConfig.Logger = logger.New(
				out, // io writer
				logger.Config{
					SlowThreshold:             time.Second,                    // Slow SQL threshold
					LogLevel:                  cfg.LogLevel.GormLoggerLevel(), // Log level
					IgnoreRecordNotFoundError: true,                           // Ignore ErrRecordNotFound error for logger
					Colorful:                  true,                           // Disable color
				},
			)
		} else {
			dbConfig.Logger = &zapLogger{log.GetLogger(cfg.LogName)}
		}

	}
	db, err := gorm.Open(dbDSN, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("数据库链接错误: %v", err)
	}
	if cfg.Debug {
		db = db.Debug()
	}
	dc, err := db.DB()
	if err != nil {
		return nil, err
	}
	if cfg.MaxIdleConns != 0 {
		dc.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns != 0 {
		dc.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.ConnMaxLifetime != 0 {
		dc.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime != 0 {
		dc.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}
	if err := dc.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

var dbs = map[string]*DB{} //全局
var db *DB
var defaultKey = utils.DefaultKey.DefaultKey

func Init(cfgs map[string]Config, options ...Option) error {
	log.Info("init db")
	opt := applyGenGormOptions(options...)
	defaultKey = opt.defKey.DefaultKey
	dbs = make(map[string]*DB)
	if len(opt.defKey.Keys) != 0 {
		opt.defKey.Keys = append(opt.defKey.Keys, opt.defKey.DefaultKey)
		for _, key := range opt.defKey.Keys {
			_, is := dbs[key]
			if is {
				continue
			}
			cfg, is := cfgs[key]
			if !is {
				log.Errorf("init db client %s not found", key)
				return fmt.Errorf("init db client %s not found", key)
			}
			cli, err := New(&cfg)
			if err != nil {
				log.Errorf("init db client %s error: %v", key, err)
				return err
			}
			dbs[key] = cli
			if key == defaultKey {
				db = cli
			}
		}
		log.Info("init db success")
		return nil
	}
	for name, cfg := range cfgs {
		cli, err := New(&cfg)
		if err != nil {
			log.Errorf("init db client %s error: %v", name, err)
			return err
		}
		dbs[name] = cli
		if name == defaultKey {
			db = cli
		}
	}
	log.Info("init db success")
	return nil
}
func InitGlobal(cfg *Config) error {
	var err error
	db, err = New(cfg)
	if err != nil {
		return err
	}
	return nil
}
func GetClient(name ...string) *DB {
	if len(name) == 0 {
		cli, is := dbs[defaultKey]
		if !is {
			panic(fmt.Errorf("db client %s not found", utils.DefaultKey.DefaultKey))
		}
		return cli
	}
	cli, is := dbs[name[0]]
	if !is {
		panic(fmt.Errorf("db client %s not found", name[0]))
	}
	return cli
}

// Client
func Client() *DB {
	return db
}
func Close() error {
	dc, err := db.DB()
	if err != nil {
		return err
	}
	return dc.Close()
}

func CloseAll() error {
	for _, cli := range dbs {
		dc, err := cli.DB()
		if err != nil {
			continue
		}
		dc.Close()
	}
	return nil
}

func Model(value any) *gorm.DB {
	return db.Model(value)
}

func Table(name string, args ...any) (tx *gorm.DB) {
	return db.Table(name, args...)
}

func Distinct(args ...any) (tx *DB) {
	return db.Distinct(args...)
}

func MapColumns(m map[string]string) (tx *DB) {
	return db.MapColumns(m)
}

// Create 创建记录
func Create(value any) *gorm.DB {
	return db.Create(value)
}
func CreateInBatches(value any, batchSize int) (tx *DB) {
	return db.CreateInBatches(value, batchSize)
}

// Save 保存记录
func Save(value any) *gorm.DB {
	return db.Save(value)
}

// First 获取第一条记录
func First(dest any, conds ...any) *gorm.DB {
	return db.First(dest, conds...)
}

// Take 获取一条记录
func Take(dest any, conds ...any) *gorm.DB {
	return db.Take(dest, conds...)
}

// Last 获取最后一条记录
func Last(dest any, conds ...any) *gorm.DB {
	return db.Last(dest, conds...)
}

// Find 查询多条记录
func Find(dest any, conds ...any) *gorm.DB {
	return db.Find(dest, conds...)
}

func FindInBatches(dest any, batchSize int, fc func(tx *DB, batch int) error) *DB {
	return db.FindInBatches(dest, batchSize, fc)
}
func FirstOrInit(dest any, conds ...any) (tx *DB) {
	return db.FirstOrInit(dest, conds...)
}
func FirstOrCreate(dest any, conds ...any) (tx *DB) {
	return db.FirstOrCreate(dest, conds...)
}

// Where 条件查询
func Where(query any, args ...any) *gorm.DB {
	return db.Where(query, args...)
}

func Not(query any, args ...any) (tx *DB) {
	return db.Not(query, args...)
}

func Or(query any, args ...any) (tx *DB) {
	return db.Or(query, args...)
}

// Order 排序
func Order(value any) *gorm.DB {
	return db.Order(value)
}

// Limit 限制数量
func Limit(limit int) *gorm.DB {
	return db.Limit(limit)
}

// Offset 偏移量
func Offset(offset int) *gorm.DB {
	return db.Offset(offset)
}

// Select 选择字段
func Select(query any, args ...any) *gorm.DB {
	return db.Select(query, args...)
}

// Omit 忽略字段
func Omit(columns ...string) *gorm.DB {
	return db.Omit(columns...)
}

// Group 分组
func Group(name string) *gorm.DB {
	return db.Group(name)
}

// Having Having条件
func Having(query any, args ...any) *gorm.DB {
	return db.Having(query, args...)
}

// Joins 关联查询
func Joins(query string, args ...any) *gorm.DB {
	return db.Joins(query, args...)
}
func InnerJoins(query string, args ...any) (tx *DB) {
	return db.InnerJoins(query, args...)
}

// Scopes 作用域
func Scopes(funcs ...func(*gorm.DB) *gorm.DB) *gorm.DB {
	return db.Scopes(funcs...)
}

// Preload 预加载
func Preload(query string, args ...any) *gorm.DB {
	return db.Preload(query, args...)
}

// Raw 原生SQL
func Raw(sql string, values ...any) *gorm.DB {
	return db.Raw(sql, values...)
}

// Exec 执行SQL
func Exec(sql string, values ...any) *gorm.DB {
	return db.Exec(sql, values...)
}

// Delete 删除记录
func Delete(value any, conds ...any) *gorm.DB {
	return db.Delete(value, conds...)
}

// Update 更新记录
func Update(column string, value any) *gorm.DB {
	return db.Update(column, value)
}

// Updates 更新多个字段
func Updates(values any) *gorm.DB {
	return db.Updates(values)
}

// Count 计数
func Count(count *int64) *gorm.DB {
	return db.Count(count)
}

// Pluck 查询单个列
func Pluck(column string, dest any) *gorm.DB {
	return db.Pluck(column, dest)
}

// Transaction 事务
func Transaction(fc func(tx *gorm.DB) error, opts ...*sql.TxOptions) error {
	return db.Transaction(fc, opts...)
}

// Begin 开始事务
func Begin(opts ...*sql.TxOptions) *gorm.DB {
	return db.Begin(opts...)
}

// Commit 提交事务
func Commit() *gorm.DB {
	return db.Commit()
}

// Rollback 回滚事务
func Rollback() *gorm.DB {
	return db.Rollback()
}

func WithContext(ctx context.Context) *DB {
	return db.WithContext(ctx)
}

func Scan(dest any) (tx *DB) {
	return db.Scan(dest)
}
func Clauses(conds ...clause.Expression) (tx *DB) {
	return db.Clauses(conds...)
}
func Attrs(attrs ...any) (tx *DB) {
	return db.Attrs(attrs...)
}

func Assign(attrs ...any) (tx *DB) {
	return db.Assign(attrs...)
}
func Unscoped() (tx *DB) {
	return db.Unscoped()
}

func AutoMigrate(models ...any) (err error) {
	return db.AutoMigrate(models...)
}
