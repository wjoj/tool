package casbinx

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gin-contrib/authz"
	"github.com/gin-gonic/gin"
	"github.com/wjoj/tool/v2/db/dbx"
	"github.com/wjoj/tool/v2/db/redisx"
	"github.com/wjoj/tool/v2/log"
	"github.com/wjoj/tool/v2/utils"
)

type DbType string

const (
	DbTypeGorm  DbType = "gorm"
	DbTypeRedis DbType = "redis"
)

type Config struct {
	DBType DbType `yaml:"dbType"`
	Key    string `yaml:"key"`
	Prefix string `yaml:"prefix"`
	Name   string `yaml:"name"`
}

const rbac_model = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")`

const rabc_model2 = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act`

type Casbin struct {
	*casbin.Enforcer
	cfg *Config
}

func New(cfg *Config) (*Casbin, error) {
	if len(cfg.Key) == 0 {
		cfg.Key = utils.DefaultKey.DefaultKey
	}
	if len(cfg.Name) == 0 {
		cfg.Name = "casbin_rule"
	}
	var adapter persist.Adapter
	var err error
	switch cfg.DBType {
	case DbTypeGorm:
		adapter, err = gormadapter.NewAdapterByDBUseTableName(dbx.GetClient(cfg.Key), cfg.Prefix, cfg.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize casbin adapter: %v", err)
		}
	case DbTypeRedis:
		PolicyKey = cfg.Prefix + cfg.Name
		adapter = NewFromRedixClient(redisx.GetClient())
	}

	m, err := model.NewModelFromString(rbac_model)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin model: %v", err)
	}
	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %v", err)
	}
	if err = enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load casbin policy: %v", err)
	}
	return &Casbin{
		Enforcer: enforcer,
		cfg:      cfg,
	}, nil
}

var cb *Casbin
var cbMap map[string]*Casbin
var defaultKey = utils.DefaultKey.DefaultKey

func Init(cfgs map[string]Config, options ...Option) error {
	log.Info("init casbin")
	opt := applyGenGormOptions(options...)
	defaultKey = opt.defKey.DefaultKey
	var err error
	cbMap, err = utils.Init("casbin", defaultKey, opt.defKey.Keys, cfgs, func(cfg Config) (*Casbin, error) {
		return New(&cfg)
	}, func(c *Casbin) {
		cb = c
	})
	if err != nil {
		log.Errorf("%v", err)
		return err
	}
	log.Info("init casbin success")
	return nil
}
func InitGlobal(cfg *Config) error {
	var err error
	cb, err = New(cfg)
	if err != nil {
		return err
	}
	return nil
}

func Get(key ...string) *Casbin {
	c, err := utils.Get("jwt", defaultKey, func(s string) (*Casbin, bool) {
		cli, is := cbMap[s]
		return cli, is
	}, key...)
	if err != nil {
		panic(err)
	}
	return c
}

func BasicAuthorizer(key ...string) func(*gin.Context) {
	return authz.NewAuthorizer(Get(key...).Enforcer)
}
