package config

import (
	"github.com/wjoj/tool/db/dbx"
	"github.com/wjoj/tool/db/mongox"
	"github.com/wjoj/tool/db/redisx"
	"github.com/wjoj/tool/httpx"
	"github.com/wjoj/tool/log"
)

var defaultKey = "def"

type EnvType string

const (
	EnvDevelopment EnvType = "dev"     // 开发环境
	EnvTesting     EnvType = "test"    // 测试环境
	EnvStaging     EnvType = "staging" // 预发布环境
	EnvProduction  EnvType = "prod"    // 生产环境
)

type App struct {
	Env       EnvType                  `yaml:"env" json:"env"`             //开发环境
	Namespace string                   `yaml:"namespace" json:"namespace"` //命名空间
	Logs      map[string]log.Config    `yaml:"logs" json:"logs"`           //日志配置
	Rediss    map[string]redisx.Config `yaml:"rediss" json:"rediss"`       //redis配置
	Dbs       map[string]dbx.Config    `yaml:"dbs" json:"dbs"`             //db配置
	Mongos    map[string]mongox.Config `yaml:"mongos" json:"mongos"`       //mongo配置
	Http      map[string]httpx.Config  `yaml:"http" json:"http"`           //http配置
}
