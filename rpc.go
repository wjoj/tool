package tool

import (
	"fmt"
	"io/ioutil"

	"github.com/gin-gonic/gin"
	"github.com/wjoj/tool/db"
	"github.com/wjoj/tool/log"
	"github.com/wjoj/tool/rpc"
	"github.com/wjoj/tool/store"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

type EnvironmentType string

//EnvironmentType 环境
const (
	EnvironmentTypeDebug  EnvironmentType = "debug"
	EnvironmentTypeFormal EnvironmentType = "formal"
)

func (e EnvironmentType) GinMode() string {
	if e == EnvironmentTypeDebug {
		return gin.DebugMode
	}
	return gin.ReleaseMode
}

type RPCServiceConfig struct {
	Environment EnvironmentType    `json:"environment" yaml:"environment"`
	Name        string             `json:"name" yaml:"name"`
	RPC         *rpc.ConfigService `json:"rpc" yaml:"rpc"`
	DB          *db.Config         `json:"db" yaml:"db"`
	Redis       *store.ConfigRedis `json:"redis" yaml:"redis"`
	Log         *log.Config        `json:"log" yaml:"log"`
}

func (c *RPCServiceConfig) Show() {
	msg := ""
	msg += "Server Name: " + c.Name
	msg += fmt.Sprintf("\nThe Environment: %s", c.Environment)
	msg += fmt.Sprintf("\nRPC Service Port: %v", c.RPC.Port)
	if c.RPC.Prom != nil {
		c.RPC.Prom.Show()
	}
	if c.RPC.Trace != nil {
		c.RPC.Trace.Show()
	}
	if c.DB != nil {
		c.DB.Show()
	}
	if c.Redis != nil {
		msg += fmt.Sprintln(c.Redis)
	}
	fmt.Println(msg)
}

func NewServiceConfig(fpath string) (*RPCServiceConfig, error) {
	yamlFile, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	var conf RPCServiceConfig
	err = yaml.Unmarshal(yamlFile, &conf)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration file, %v", err)
	}

	if len(conf.Name) == 0 {
		return nil, fmt.Errorf("the name configuration is empty")
	}

	if conf.RPC == nil {
		return nil, fmt.Errorf("the RPC service configuration is empty")
	}
	if len(conf.RPC.ServiceName) == 0 {
		conf.RPC.ServiceName = conf.Name
	}
	return &conf, nil
}

func DefaultServiceConfig() (*RPCServiceConfig, error) {
	return NewServiceConfig("./etc/config.yaml")
}

func ServiceRPCStart(cfg *RPCServiceConfig, dbFunc func(dbm *gorm.DB), sFunc func(srv *grpc.Server)) {
	cfg.Show()
	if db, err := cfg.DB.StartDB(); err != nil {
		panic(fmt.Errorf("db error: %v", err))
	} else if db != nil {
		dbFunc(db)
	}
	if cfg.Log != nil {
		log.NewGlobal(cfg.Log)
	}
	if err := store.SetGlobalRedis(cfg.Redis); err != nil {
		panic(fmt.Errorf("redis error: %v", err))
	}
	cfg.RPC.Start(func(srv *grpc.Server) {
		sFunc(srv)
	}, func(err error) {
		panic(fmt.Errorf("rpc service error: %v", err))
	})
}
