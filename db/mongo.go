package db

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConfig struct {
	Host          string `json:"host" yaml:"host"`
	Port          uint   `json:"port" yaml:"port"`
	User          string `json:"user" yaml:"user"`
	Pwd           string `json:"pwd" yaml:"pwd"`
	DBName        string `json:"dbname" yaml:"dbname"`
	Timeout       int    `json:"timeout" yaml:"timeout"`             //ms
	ReconnectTime int    `json:"reconnectTime" yaml:"reconnectTime"` //ms
}

func (c *MongoConfig) URL() string {
	return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s", c.User, c.Pwd, c.Host, c.Port, c.DBName)
}

type Mongo struct {
	c       *MongoConfig
	cli     *mongo.Client
	watch   int32
	lock    sync.Mutex
	dbnames map[string]*mongo.Database
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
	url := cfg.URL()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(cfg.Timeout))
	mgo := options.Client().ApplyURI(url)
	client, err := mongo.Connect(ctx, mgo)
	cancel()
	if err != nil {
		return nil, err
	}
	cl := &Mongo{
		cli:     client,
		c:       cfg,
		dbnames: make(map[string]*mongo.Database),
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
	return m.cli.Disconnect(context.Background())
}

func (m *Mongo) WatchConnect() {
	if !atomic.CompareAndSwapInt32(&m.watch, 0, 1) {
		return
	}
	for {
		ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(m.c.ReconnectTime))
		if err := m.cli.Ping(ctx, nil); err != nil {
			fmt.Printf("\nping err:%v", err)
		re:
			ctx2, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(m.c.Timeout))
			if err := m.cli.Connect(ctx2); err != nil {
				fmt.Printf("\nConnect")
				goto re
			}
			cancel()
			continue
		}
		select {
		case <-ctx.Done():
		}
	}
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

func (m *Mongo) DbnameCollection(name string, col string) *mongo.Collection {
	return m.Dbname(name).Collection(col)
}
