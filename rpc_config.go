package tool

import (
	"fmt"
	"io/ioutil"

	"github.com/gin-gonic/gin"
	"github.com/wjoj/tool/db"
	"github.com/wjoj/tool/rpc"
	"gopkg.in/yaml.v2"
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

type SetviceConfig struct {
	Environment EnvironmentType    `json:"environment" yaml:"environment"`
	Name        string             `json:"name" yaml:"name"`
	RPC         *rpc.ConfigService `json:"rpc" yaml:"rpc"`
	DB          *db.Config         `json:"db" yaml:"db"`
}

func (c *SetviceConfig) Show() {
	msg := ""
	msg += "Server Name: " + c.Name
	msg += fmt.Sprintln("The Environment: " + c.Environment)
	msg += fmt.Sprintln("" + fmt.Sprintf("RPC Service Port: %v", c.RPC.Port))
	if c.DB != nil {
		c.DB.Show()
	}
	fmt.Println(msg)
}

func NewServiceConfig(fpath string) (*SetviceConfig, error) {
	yamlFile, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	var conf SetviceConfig
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

func DefaultServiceConfig() (*SetviceConfig, error) {
	return NewServiceConfig("./etc/config.yaml")
}
