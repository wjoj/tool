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

func (c *HTTPConfig) Show() {
	msg := ""
	msg += "Server Name: " + c.Name
	msg += fmt.Sprintln("The Environment: " + c.Environment)
	if c.Http != nil {
		msg += fmt.Sprintln("" + fmt.Sprintf("HTTP Service Port: %v", c.Http.Port))
	}
	if c.DB != nil {
		c.DB.Show()
	}
	fmt.Println(msg)
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
