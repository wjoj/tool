package tool

import (
	"fmt"
	"io/ioutil"

	"github.com/wjoj/tool/db"
	"github.com/wjoj/tool/http"
	"gopkg.in/yaml.v2"
)

type HTTPConfig struct {
	Environment EnvironmentType `json:"environment" yaml:"environment"`
	Name        string          `json:"name" yaml:"name"`
	Http        *http.HTTP      `json:"http" yaml:"http"`
	DB          *db.Config      `json:"db" yaml:"db"`
}

func NewHTTPConfig(fpath string) (*HTTPConfig, error) {
	yamlFile, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	var conf HTTPConfig
	err = yaml.Unmarshal(yamlFile, &conf)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration file, %v", err)
	}
	if conf.Http == nil {
		return nil, fmt.Errorf("the HTTP service configuration is empty")
	}
	return &conf, nil
}

func DefaultHTTPConfig() (*HTTPConfig, error) {
	return NewHTTPConfig("./etc/config.yaml")
}
