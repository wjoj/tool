package tool

import (
	"fmt"
	"io/ioutil"
	httph "net/http"

	"github.com/wjoj/tool/db"
	"github.com/wjoj/tool/http"
	"github.com/wjoj/tool/log"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

type HTTPConfig struct {
	Environment EnvironmentType `json:"environment" yaml:"environment"`
	Name        string          `json:"name" yaml:"name"`
	Http        *http.HTTP      `json:"http" yaml:"http"`
	DB          *db.Config      `json:"db" yaml:"db"`
	Log         *log.Config     `json:"log" yaml:"log"`
}

func (c *HTTPConfig) Show() {
	msg := ""
	msg += fmt.Sprintln("Server Name: " + c.Name)
	msg += fmt.Sprintln("" + fmt.Sprintf("The Environment: %s", c.Environment))
	if c.Http != nil {
		msg += fmt.Sprintln("" + fmt.Sprintf("HTTP Service Port: %v", c.Http.Port))
		if c.Http.Prom != nil {
			msg += fmt.Sprintln(c.Http.Prom)
		}
		if c.Http.Trace != nil {
			msg += fmt.Sprintln(c.Http.Trace)
		}
	}
	if c.DB != nil {
		msg += fmt.Sprintln(c.DB)
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

func ServiceHTTPStart(cfg *HTTPConfig, dbFunc func(dbm *gorm.DB), handler httph.Handler) {
	cfg.Show()
	if db, err := cfg.DB.StartDB(); err != nil {
		panic(fmt.Errorf("db error: %v", err))
	} else if db != nil {
		dbFunc(db)
	}
	cfg.Http.Start(func(err error) {
		panic(fmt.Errorf("http service error: %v", err))
	}, handler)
}
