package tool

import (
	"fmt"
	"io/ioutil"

	"github.com/wjoj/tool/rpc"
	"gopkg.in/yaml.v2"
)

type EnvironmentType string

//EnvironmentType 环境
const (
	EnvironmentTypeDebug  EnvironmentType = "debug"
	EnvironmentTypeFormal EnvironmentType = "formal"
)

type SetviceConfig struct {
	Environment EnvironmentType `json:"environment" yaml:"environment"`
	Name        string          `json:"name" yaml:"name"`
	RPC         *rpc.ServiceRPC
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
	if conf.RPC == nil {
		return nil, fmt.Errorf("the RPC service configuration is empty")
	}
	return &conf, nil
}

func DefaultServiceConfig() (*SetviceConfig, error) {
	return NewServiceConfig("./etc/config.yaml")
}
