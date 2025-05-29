package config

import (
	"github.com/wjoj/tool/v2/db/dbx"
	"github.com/wjoj/tool/v2/db/mongox"
	"github.com/wjoj/tool/v2/db/redisx"
	"github.com/wjoj/tool/v2/httpx"
	"github.com/wjoj/tool/v2/log"
	"github.com/wjoj/tool/v2/resources/casbinx"
	"github.com/wjoj/tool/v2/resources/jwt"
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
	Env       EnvType                   `yaml:"env" json:"env"`             //开发环境
	Namespace string                    `yaml:"namespace" json:"namespace"` //命名空间
	Logs      map[string]log.Config     `yaml:"logs" json:"logs"`           //日志配置
	Rediss    map[string]redisx.Config  `yaml:"rediss" json:"rediss"`       //redis配置
	Dbs       map[string]dbx.Config     `yaml:"dbs" json:"dbs"`             //db配置
	Mongos    map[string]mongox.Config  `yaml:"mongos" json:"mongos"`       //mongo配置
	Http      map[string]httpx.Config   `yaml:"http" json:"http"`           //http配置
	Casbins   map[string]casbinx.Config `yaml:"casbins" json:"casbins"`     //casbin配置
	Jwts      map[string]jwt.Config     `yaml:"jwts" json:"jwts"`
}
