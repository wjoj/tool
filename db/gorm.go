package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/wjoj/tool/base"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBType string

const (
	DBTypeMySQL      DBType = "mysql"
	DBTypeSQLite            = "sqlite3"
	DBTypePostGres          = "postgres"
	DBTypeMsSQL             = "mssql"
	DBTypeSQLServer         = "sqlserver"
	DBTypeClickHouse        = "clickhouse"
)

type DBLogModelType string

const (
	DBLogModelTypeConsole = "console"
)

type Config struct {
	Type           DBType         `json:"type" yaml:"type"`
	Account        string         `json:"account" yaml:"account"`
	Password       string         `json:"password" yaml:"password"`
	Host           string         `json:"host" yaml:"host"`
	Port           int            `json:"port" yaml:"port"`
	DBName         string         `json:"dbname" yaml:"dbname"`
	TimeOut        int            `json:"timeout" yaml:"timeout"`
	PoolFreeNumber int            `json:"poolFreeNumber" yaml:"poolFreeNumber"`
	PoolNumber     int            `json:"poolNumber" yaml:"poolNumber"`
	PoolLifeTime   time.Duration  `json:"poolLifeTime" yaml:"poolLifeTime"`
	LogPath        DBLogModelType `json:"logpath" yaml:"logpath"`
}

func (c *Config) IsDB() error {
	if c == nil || len(c.Type) == 0 {
		return fmt.Errorf("db configuration is empty")
	}
	return nil
}

func (c *Config) OpenDB() (*gorm.DB, error) {
	var dbDSN gorm.Dialector
	dbObj := c
	if len(dbObj.DBName) == 0 {
		return nil, fmt.Errorf("数据库名称不能为空")
	}

	switch dbObj.Type {
	case DBTypeMySQL:
		fmt.Printf("数据库存储方式:MySQL\n")
		if len(dbObj.Account) == 0 || len(dbObj.Password) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的账号或密码不能为空")
		}
		dbDSN = mysql.New(mysql.Config{
			DSN: fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8mb4&parseTime=False&loc=Local&allowNativePasswords=true",
				dbObj.Account,
				dbObj.Password,
				dbObj.Host,
				dbObj.Port,
				dbObj.DBName,
			),
		})

	case DBTypePostGres:
		fmt.Printf("数据库存储方式:PostGres\n")
		if len(dbObj.Account) == 0 || len(dbObj.Password) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的账号或密码不能为空")
		}
		dbDSN = postgres.Open(fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s port=%d",
			dbObj.Host, dbObj.Account, dbObj.DBName, dbObj.Password, dbObj.Port))

	case DBTypeMsSQL:
		fmt.Printf("数据库存储方式:MSSQL\n")
		if len(dbObj.Account) == 0 || len(dbObj.Password) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的账号或密码不能为空")
		}
		// dbDSN = fmt.Sprintf("sqlserver://%v:%v@%v:%v?database=%v",
		// 	dbObj.Account, dbObj.Password, dbObj.Host, dbObj.Port, dbObj.DBName)

	case DBTypeSQLServer:
		dbDSN = sqlserver.Open(fmt.Sprintf("sqlserver://%v:%v@%v:%v?database=%v",
			dbObj.Account, dbObj.Password, dbObj.Host, dbObj.Port, dbObj.DBName))

	case DBTypeClickHouse:
		dbDSN = clickhouse.Open(fmt.Sprintf("tcp://%v:%v?username=%v&password=%v&database=%v&read_timeout=%v",
			dbObj.Host, dbObj.Port, dbObj.Account, dbObj.Password, dbObj.DBName, dbObj.TimeOut))

	default:
		fmt.Printf("数据库存储方式:SQLite\n")
		dbDSN = sqlite.Open(fmt.Sprintf("%v.db", dbObj.DBName))

	}
	dbConfig := &gorm.Config{}
	if len(c.LogPath) != 0 {
		var out logger.Writer
		if c.LogPath == DBLogModelTypeConsole {
			out = log.New(os.Stdout, "\r\n", log.LstdFlags)
		} else {
			f, err := base.FileOpenAppend(string(c.LogPath))
			if err != nil {
				return nil, err
			}
			out = log.New(f, "\r\n", log.LstdFlags)
		}
		dbConfig.Logger = logger.New(
			out, // io writer
			logger.Config{
				SlowThreshold:             time.Second,   // Slow SQL threshold
				LogLevel:                  logger.Silent, // Log level
				IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
				Colorful:                  true,          // Disable color
			},
		)
	}

	db, err := gorm.Open(dbDSN, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("数据库链接错误: %v", err)
	}
	dc, err := db.DB()
	if err != nil {
		return nil, err
	}
	if dbObj.PoolFreeNumber != 0 {
		dc.SetMaxIdleConns(c.PoolFreeNumber)
	}
	if dbObj.PoolNumber != 0 {
		dc.SetMaxOpenConns(c.PoolNumber)
	}
	if dbObj.PoolLifeTime != 0 {
		dc.SetConnMaxLifetime(c.PoolLifeTime)
	}

	return db, nil
}
