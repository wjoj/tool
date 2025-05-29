package config

import (
	"github.com/wjoj/tool/v2/db/dbx"
	"github.com/wjoj/tool/v2/db/mongox"
	"github.com/wjoj/tool/v2/db/redisx"
	"github.com/wjoj/tool/v2/httpx"
	"github.com/wjoj/tool/v2/log"
	"github.com/wjoj/tool/v2/resources/casbinx"
	"github.com/wjoj/tool/v2/resources/jwt"
	"github.com/wjoj/tool/v2/utils"
)

var (
	env        EnvType
	namespace  string
	configFile string
	configRoot string
	logs       map[string]log.Config
	rediss     map[string]redisx.Config
	dbs        map[string]dbx.Config
	mongos     map[string]mongox.Config
	http       map[string]httpx.Config
	casbins    map[string]casbinx.Config
	jwts       map[string]jwt.Config
)

func SetDefaultKey(key string) {
	defaultKey = key
}

func GetDefaultKey() string {
	return defaultKey
}

// GetEnv 获取环境
func GetEnv() EnvType {
	return env
}
func SetEnv(e EnvType) {
	env = e
}
func GetNamespace() string {
	return namespace
}

func SetNamespace(n string) {
	namespace = n
}
func GetConfigFile() string {
	return configFile
}
func SetConfigFile(f string) {
	configFile = f
}

func GetConfigRoot() string {
	return configRoot
}
func SetConfigRoot(r string) {
	configRoot = r
}

// SetLog 设置日志
func SetLog(lgs map[string]log.Config) {
	logs = lgs
}

func GetLogs() map[string]log.Config {
	return logs
}

// GetLog 获取日志
func GetLog(key ...string) (logc log.Config) {
	logc, err := utils.Get("config log", GetDefaultKey(), func(k string) (log.Config, bool) {
		m, is := logs[k]
		return m, is
	}, key...)
	if err != nil {
		panic(err)
	}
	return
}

func SetRediss(r map[string]redisx.Config) {
	rediss = r
}

// SetRedis 设置redis
func GetRediss() map[string]redisx.Config {
	return rediss
}

func GetRedis(key ...string) (redis redisx.Config) {
	redis, err := utils.Get("config redis", GetDefaultKey(), func(k string) (redisx.Config, bool) {
		m, is := rediss[k]
		return m, is
	}, key...)
	if err != nil {
		panic(err)
	}
	return
}

func SetDbs(d map[string]dbx.Config) {
	dbs = d
}

// SetDb 设置db
func GetDbs() map[string]dbx.Config {
	return dbs
}

func GetDb(key ...string) (db dbx.Config) {
	db, err := utils.Get("config db", GetDefaultKey(), func(k string) (dbx.Config, bool) {
		m, is := dbs[k]
		return m, is
	}, key...)
	if err != nil {
		panic(err)
	}
	return
}

func SetMongos(m map[string]mongox.Config) {
	mongos = m
}

func GetMongos() map[string]mongox.Config {
	return mongos
}

func GetMongo(key ...string) (mgo mongox.Config) {
	mgo, err := utils.Get("config mongo", GetDefaultKey(), func(k string) (mongox.Config, bool) {
		m, is := mongos[k]
		return m, is
	}, key...)
	if err != nil {
		panic(err)
	}
	return
}

func SetHttp(h map[string]httpx.Config) {
	http = h
}

func GetHttp() map[string]httpx.Config {
	return http
}

func GetHttpServer(key ...string) (cfg httpx.Config) {
	cfg, err := utils.Get("config http", GetDefaultKey(), func(k string) (httpx.Config, bool) {
		m, is := http[k]
		return m, is
	}, key...)
	if err != nil {
		panic(err)
	}
	return
}

func SetCasbins(c map[string]casbinx.Config) {
	casbins = c
}

func GetCasbins() map[string]casbinx.Config {
	return casbins
}

func GetCasbin(key ...string) (casbin casbinx.Config) {
	casbin, err := utils.Get("config casbin", GetDefaultKey(), func(k string) (casbinx.Config, bool) {
		m, is := casbins[k]
		return m, is
	}, key...)
	if err != nil {
		panic(err)
	}
	return
}

func SetJwts(j map[string]jwt.Config) {
	jwts = j
}

func GetJwts() map[string]jwt.Config {
	return jwts
}

func GetJwt(key ...string) (jt jwt.Config) {
	jt, err := utils.Get("config jwt", GetDefaultKey(), func(k string) (jwt.Config, bool) {
		m, is := jwts[k]
		return m, is
	}, key...)
	if err != nil {
		panic(err)
	}
	return
}
