package db

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wjoj/tool/locks"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoCollection interface {
	DBName() string
	CollectionName() string
}

type MongoConfig struct {
	Host          string `json:"host" yaml:"host"`
	Port          uint   `json:"port" yaml:"port"`
	User          string `json:"user" yaml:"user"`
	Pwd           string `json:"pwd" yaml:"pwd"`
	DBName        string `json:"dbname" yaml:"dbname"`
	Timeout       int    `json:"timeout" yaml:"timeout"`             //ms
	ReconnectTime int    `json:"reconnectTime" yaml:"reconnectTime"` //ms
	MaxPoolSize   uint64 `json:"maxPoolSize" yaml:"maxPoolSize"`
	MinPoolSize   uint64 `json:"minPoolSize" yaml:"minPoolSize"`
}

func (c *MongoConfig) URL() (string, error) {
	if len(c.Pwd) != 0 && len(c.User) == 0 {
		return "", fmt.Errorf("the mongodb account cannot be empty")
	}
	if len(c.Host) == 0 {
		return "", fmt.Errorf("the mongodb host cannot be empty")
	}
	if c.Port == 0 {
		c.Port = 27017
	}
	if len(c.User) == 0 {
		return fmt.Sprintf("mongodb://%s:%d/%s", c.Host, c.Port, c.DBName), nil
	}
	return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s", c.User, c.Pwd, c.Host, c.Port, c.DBName), nil
}

type Mongo struct {
	c         *MongoConfig
	cli       *mongo.Client
	watch     int32
	lock      sync.Mutex
	shareLock *locks.Share
	dbnames   map[string]*mongo.Database
	close     chan struct{}
}

func NewMongo(cfg *MongoConfig) (*Mongo, error) {
	if len(cfg.Host) == 0 {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = 27017
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 500
	}
	if cfg.ReconnectTime == 0 {
		cfg.ReconnectTime = 1000
	}
	url, err := cfg.URL()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(cfg.Timeout))
	mgo := options.Client().ApplyURI(url)
	mgo.MinPoolSize = &cfg.MaxPoolSize
	mgo.MinPoolSize = &cfg.MinPoolSize
	client, err := mongo.Connect(ctx, mgo)
	cancel()
	if err != nil {
		return nil, err
	}
	cl := &Mongo{
		cli:       client,
		c:         cfg,
		dbnames:   make(map[string]*mongo.Database),
		close:     make(chan struct{}),
		shareLock: locks.NewShare(),
	}
	if err := cl.Ping(); err != nil {
		return nil, err
	}
	return cl, err
}

func (m *Mongo) Ping() error {
	return m.cli.Ping(context.Background(), nil)
}

func (m *Mongo) Close() error {
	if atomic.LoadInt32(&m.watch) == 1 {
		m.close <- struct{}{}
	}
	return m.cli.Disconnect(context.Background())
}

func (m *Mongo) WatchConnect() {
	if !atomic.CompareAndSwapInt32(&m.watch, 0, 1) {
		return
	}
	for {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(m.c.ReconnectTime))
		if err := m.cli.Ping(ctx, nil); err != nil {
			fmt.Printf("\nmongo ping err:%v", err)
		re:
			ctx2, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(m.c.Timeout))
			if err := m.cli.Connect(ctx2); err != nil {
				fmt.Printf("\n mongo connect error:%v", err)
				goto re
			}
			cancel()
			continue
		}
		select {
		case <-ctx.Done():
		case <-m.close:
			atomic.StoreInt32(&m.watch, 0)
			return
		}
	}
}

func (m *Mongo) AllDBs() ([]mongo.DatabaseSpecification, int64, error) {
	data, err := m.cli.ListDatabases(context.Background(), bson.D{})
	if err != nil {
		return nil, 0, err
	}
	return data.Databases, data.TotalSize, nil
}

func (m *Mongo) IsDBExit(dbname string) error {
	if dbMap, _, err := m.shareLock.LockWait("db_exit", func() (any, error) {
		dbs, _, err := m.AllDBs()
		if err != nil {
			return nil, err
		}
		dbMap := make(map[string]struct{})
		for _, dbm := range dbs {
			dbMap[dbm.Name] = struct{}{}
		}
		return dbMap, nil
	}); err != nil {
		return err
	} else if _, is := dbMap.(map[string]struct{})[dbname]; is {
		return nil
	}
	return fmt.Errorf("the %s database does not exist", dbname)
}

func (m *Mongo) Dbname(name string) *mongo.Database {
	m.lock.Lock()
	defer m.lock.Unlock()
	db, is := m.dbnames[name]
	if is {
		return db
	}
	db = m.cli.Database(name)
	m.dbnames[name] = db
	return db
}

func (m *Mongo) DbnameStructure(v MongoCollection) *mongo.Database {
	return m.Dbname(v.DBName())
}

func (m *Mongo) AllDBNameCollections(dbname string, col string) ([]*mongo.CollectionSpecification, error) {
	colls, err := m.Dbname(dbname).ListCollectionSpecifications(context.Background(), bson.D{})
	if err != nil {
		return nil, err
	}
	return colls, nil
}

func (m *Mongo) IsDBNameCollectionExit(dbname string, col string) error {
	if collMap, _, err := m.shareLock.LockWait("key string", func() (any, error) {
		colls, err := m.AllDBNameCollections(dbname, col)
		if err != nil {
			return nil, err
		}
		collMap := make(map[string]struct{})
		for _, coll := range colls {
			collMap[coll.Name] = struct{}{}
		}
		return collMap, nil
	}); err != nil {
		return err
	} else if _, is := collMap.(map[string]struct{})[col]; is {
		return nil
	}
	return fmt.Errorf("the %s collection does not exist", col)
}

func (m *Mongo) DbnameCollection(dbname string, col string) *Collection {
	return &Collection{
		name: col,
		col:  m.Dbname(dbname).Collection(col),
	}
}

func (m *Mongo) DbnameCollectionStructure(v MongoCollection) *Collection {
	return m.DbnameCollection(v.DBName(), v.CollectionName())
}

type Collection struct {
	col  *mongo.Collection
	name string
}

func (c *Collection) Switch(name string) *Collection {
	return &Collection{
		name: name,
		col:  c.col.Database().Collection(name),
	}
}

func (c *Collection) Collection() *mongo.Collection {
	return c.col
}

func (c *Collection) Insert(d ...interface{}) ([]interface{}, error) {
	if len(d) == 0 {
		res, err := c.col.InsertOne(context.Background(), d)
		if err != nil {
			return nil, err
		}
		return []interface{}{res.InsertedID}, nil
	} else {
		res, err := c.col.InsertMany(context.Background(), d)
		if err != nil {
			return nil, err
		}
		return res.InsertedIDs, nil
	}
}

func (c *Collection) Update(filter bson.M, d ...interface{}) (int64, error) {
	if len(d) == 0 {
		res, err := c.col.UpdateOne(context.Background(), filter, d)
		if err != nil {
			return 0, err
		}
		if res.ModifiedCount == res.UpsertedCount {
			return res.UpsertedCount, nil
		}
		return res.ModifiedCount - res.UpsertedCount, nil
	} else {
		res, err := c.col.UpdateMany(context.Background(), filter, d)
		if err != nil {
			return 0, err
		}
		if res.ModifiedCount == res.UpsertedCount {
			return res.UpsertedCount, nil
		}
		return res.ModifiedCount - res.UpsertedCount, nil
	}
}

func (c *Collection) Delete(filter bson.M, isMany bool) error {
	if isMany {
		_, err := c.col.DeleteMany(context.Background(), filter)
		if err != nil {
			return err
		}
	}
	_, err := c.col.DeleteOne(context.Background(), filter)
	return err
}

func (c *Collection) Find(sel string, filter bson.M, offset, limit int64, out interface{}) error {
	sels := strings.Split(sel, ",")
	sls := make(bson.D, len(sels))
	for _, s := range sels {
		sls = append(sls, bson.E{Key: s, Value: 1})
	}
	switch reflect.TypeOf(out).Kind() {
	case reflect.Array, reflect.Slice:
		opts := []*options.FindOptions{}
		if len(sls) != 0 {
			opts = append(opts, options.Find().SetProjection(sls))
		}
		if limit != 0 {
			opts = append(opts, options.Find().SetLimit(limit))
		}
		if offset != 0 {
			opts = append(opts, options.Find().SetSkip(offset))
		}
		cur, err := c.col.Find(context.Background(), filter, opts...)
		if err != nil {
			return err
		}
		err = cur.All(context.Background(), out)
		if err != nil {
			return err
		}
		return cur.Close(context.Background())
	}
	opts := []*options.FindOneOptions{}
	if len(sls) != 0 {
		opts = append(opts, options.FindOne().SetProjection(sls))
	}
	return c.col.FindOne(context.Background(), filter, opts...).Decode(out)
}

func (c *Collection) Aggregate(pipeline bson.D, v interface{}) error {
	cur, err := c.col.Aggregate(context.Background(), pipeline)
	if err != nil {
		return err
	}
	err = cur.All(context.Background(), v)
	if err != nil {
		return err
	}
	return cur.Close(context.Background())
}

func (c *Collection) Count(filter bson.D) (int64, error) {
	return c.col.CountDocuments(context.Background(), filter)
}

var Mgo *Mongo

func LoadGlobalMongo(cfg *MongoConfig) error {
	m, err := NewMongo(cfg)
	if err != nil {
		return err
	}
	Mgo = m
	go m.WatchConnect()
	return nil
}

func SetGlobalMongo(mgo *Mongo) {
	Mgo = mgo
}

func GlobalMongo() *Mongo {
	return Mgo
}
