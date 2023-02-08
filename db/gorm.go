package db

import (
	"fmt"
	"log"
	"os"
	"time"

	// "github.com/8treenet/gcache"
	// "github.com/8treenet/gcache/option"
	"github.com/wjoj/tool/base"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
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

/*
https://gorm.io/zh_CN/docs/models.html
*/
type Config struct {
	Type           DBType         `json:"type" yaml:"type"`
	Debug          bool           `json:"-" yaml:"-"`
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

func (c *Config) String() string {
	msg := ""
	msg += fmt.Sprintf("Data storage type: %s", c.Type)
	msg += fmt.Sprintf("\n\tAccount: %s Password: %s", c.Account, c.Password)
	msg += fmt.Sprintf("\n\tHost: " + c.Host + " Port: " + fmt.Sprintf("%d", c.Port))
	msg += fmt.Sprintf("\n\tDBName: " + c.DBName)
	msg += fmt.Sprintf("\n\tPool: " + fmt.Sprintf("%d Free pool: %d", c.PoolNumber, c.PoolFreeNumber))
	if len(c.LogPath) != 0 {
		if c.LogPath == DBLogModelTypeConsole {
			msg += fmt.Sprintf("\n\tLog type %s", c.LogPath)
		} else {
			msg += fmt.Sprintf("\n\tLog type file path: %s", c.LogPath)
		}
	}
	return msg
}

func (c *Config) Show() {
	fmt.Println(c)
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
		// fmt.Printf("数据库存储方式:MySQL\n")
		if len(dbObj.Account) == 0 || len(dbObj.Password) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的账号或密码不能为空")
		}
		dbDSN = mysql.New(mysql.Config{
			DSN: fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8mb4&parseTime=True&loc=Local&allowNativePasswords=true",
				dbObj.Account,
				dbObj.Password,
				dbObj.Host,
				dbObj.Port,
				dbObj.DBName,
			),
		})

	case DBTypePostGres:
		// fmt.Printf("数据库存储方式:PostGres\n")
		if len(dbObj.Account) == 0 || len(dbObj.Password) == 0 {
			return nil, fmt.Errorf("数据库链接错误: 数据库链接的账号或密码不能为空")
		}
		dbDSN = postgres.Open(fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s port=%d",
			dbObj.Host, dbObj.Account, dbObj.DBName, dbObj.Password, dbObj.Port))

	case DBTypeMsSQL:
		// fmt.Printf("数据库存储方式:MSSQL\n")
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
		// fmt.Printf("数据库存储方式:SQLite\n")
		dbDSN = sqlite.Open(fmt.Sprintf("%v.db", dbObj.DBName))

	}
	dbConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	}
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
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  logger.Info, // Log level
				IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				Colorful:                  true,        // Disable color
			},
		)
	}

	db, err := gorm.Open(dbDSN, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("数据库链接错误: %v", err)
	}

	// opt := option.DefaultOption{}
	// opt.Expires = 300              //缓存时间, 默认120秒。范围30-43200
	// opt.Level = option.LevelSearch //缓存级别，默认LevelSearch。LevelDisable:关闭缓存，LevelModel:模型缓存， LevelSearch:查询缓存
	// opt.AsyncWrite = false         //异步缓存更新, 默认false。 insert update delete 成功后是否异步更新缓存。 ps: affected如果未0，不触发更新。
	// opt.PenetrationSafe = false    //开启防穿透, 默认false。 ps:防击穿强制全局开启。

	// gcache.AttachDB(db, &opt, &option.RedisOption{Addr: "localhost:6379"})
	if c.Debug {
		db = db.Debug()
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

	if err := dc.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func (c *Config) StartDB() (*gorm.DB, error) {
	if err := c.IsDB(); err != nil {
		return nil, nil
	}
	db, err := c.OpenDB()
	if err != nil {
		return nil, err
	}
	return db, nil
}
