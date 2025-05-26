package mongox

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wjoj/tool/v2/log"
	"github.com/wjoj/tool/v2/utils"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Config struct {
	Url             string        `yaml:"url" json:"url"`
	Dbname          string        `yaml:"dbname" json:"dbname"`
	MaxConnIdleTime time.Duration `yaml:"maxConnIdleTime" json:"maxConnIdleTime"`
	MaxPoolSize     uint64        `yaml:"maxPoolSize" json:"maxPoolSize"`
	MinPoolSize     uint64        `yaml:"minPoolSize" json:"minPoolSize"`
	MaxConnecting   uint64        `yaml:"maxConnecting" json:"maxConnecting"`
}

type Mongo struct {
	cfg *Config
	cli *mongo.Client
	db  *mongo.Database
}

func New(cfg *Config) (*Mongo, error) {
	if len(cfg.Url) == 0 {
		return nil, fmt.Errorf("mongo url is empty")
	}
	url := ""
	if strings.HasPrefix(cfg.Url, ":") {
		url = "mongodb" + cfg.Url
	} else if strings.HasPrefix(cfg.Url, "//") {
		url = "mongodb:" + cfg.Url
	} else if strings.HasPrefix(cfg.Url, "mongodb://") {
		url = cfg.Url
	} else {
		url = "mongodb://" + cfg.Url
	}
	mgo := options.Client().ApplyURI(url)
	mgo.SetMaxConnIdleTime(cfg.MaxConnIdleTime).
		SetMaxConnecting(cfg.MaxConnecting).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize)
	client, err := mongo.Connect(mgo)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(context.Background(), nil); err != nil {
		return nil, err
	}
	if len(cfg.Dbname) != 0 {
		return &Mongo{
			cfg: cfg,
			cli: client,
			db:  client.Database(cfg.Dbname),
		}, nil
	}
	return &Mongo{
		cfg: cfg,
		cli: client,
	}, nil
}

func (m *Mongo) Client() *mongo.Client {
	return m.cli
}
func (m *Mongo) Database() *mongo.Database {
	return m.db
}
func (m *Mongo) Ping() error {
	return m.cli.Ping(context.Background(), nil)
}
func (m *Mongo) AllDBs() ([]mongo.DatabaseSpecification, int64, error) {
	data, err := m.cli.ListDatabases(context.Background(), bson.D{})
	if err != nil {
		return nil, 0, err
	}
	return data.Databases, data.TotalSize, nil
}
func (m *Mongo) DbAllDCollections(dbname string) ([]mongo.CollectionSpecification, error) {
	colls, err := m.DB(dbname).ListCollectionSpecifications(context.Background(), bson.D{})
	if err != nil {
		return nil, err
	}
	return colls, nil
}

func (m *Mongo) DB(name string) *mongo.Database {
	return m.cli.Database(name)
}

func (m *Mongo) DbCollection(dbname, colname string) *Collection {
	return &Collection{
		dbName:  dbname,
		colName: colname,
		col:     m.cli.Database(dbname).Collection(colname),
	}
}
func (m *Mongo) Collection(colname string) *Collection {
	if len(m.cfg.Dbname) != 0 {
		log.Warnf("use default dbname is empty")
		return nil
	}
	return &Collection{
		dbName:  m.cfg.Dbname,
		colName: colname,
		col:     m.db.Collection(colname),
	}
}

// Close函数用于关闭Mongo连接
func (m *Mongo) Close() error {
	return m.cli.Disconnect(context.Background())
}

var dbs = make(map[string]*Mongo)
var db *Mongo
var defaultKey = utils.DefaultKey.DefaultKey

func Init(cfgs map[string]Config, options ...Option) error {
	opt := applyGenGormOptions(options...)
	defaultKey = opt.defKey.DefaultKey
	dbs = make(map[string]*Mongo)
	if len(opt.defKey.Keys) != 0 {
		opt.defKey.Keys = append(opt.defKey.Keys, opt.defKey.DefaultKey)
		for _, key := range opt.defKey.Keys {
			_, is := dbs[key]
			if is {
				continue
			}
			cfg, is := cfgs[key]
			if !is {
				return fmt.Errorf("mongo client %s not found", key)
			}
			cli, err := New(&cfg)
			if err != nil {
				return err
			}
			dbs[key] = cli
			if key == defaultKey {
				db = cli
			}
		}
		return nil
	}
	for name, cfg := range cfgs {
		cli, err := New(&cfg)
		if err != nil {
			return err
		}
		dbs[name] = cli
		if name == defaultKey {
			db = cli
		}
	}
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

func GetClient(name ...string) *Mongo {
	if len(name) == 0 {
		cli, is := dbs[defaultKey]
		if !is {
			panic(fmt.Errorf("mongo client %s not found", utils.DefaultKey.DefaultKey))
		}
		return cli
	}
	cli, is := dbs[name[0]]
	if !is {
		panic(fmt.Errorf("mongo client %s not found", name[0]))
	}
	return cli
}

// Client 重新一遍所有方法
func Client() *Mongo {
	return db
}
func Close() error {
	return db.Close()
}

func CloseAll() error {
	for _, cli := range dbs {
		err := cli.Close()
		if err != nil {
			continue
		}
	}
	return nil
}

func Insert(d ...any) {

}

func Update(filter bson.M, d ...any) (int64, error) {
	return 0, nil
}

func Delete(colname string, filter bson.M, isMany bool) error {
	return db.Collection(colname).Delete(filter, isMany)
}
