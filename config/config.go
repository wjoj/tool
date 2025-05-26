package config

import (
	"github.com/wjoj/tool/v2/db/dbx"
	"github.com/wjoj/tool/v2/db/mongox"
	"github.com/wjoj/tool/v2/db/redisx"
	"github.com/wjoj/tool/v2/httpx"
	"github.com/wjoj/tool/v2/log"
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
func GetLog(name ...string) (log log.Config) {
	var is bool
	if len(name) == 0 {
		log, is = logs[GetDefaultKey()]
		if !is {
			panic(GetDefaultKey() + " log config not found")
		}
	} else {
		log, is = logs[name[0]]
		if !is {
			panic(name[0] + " log config not found")
		}
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

func GetRedis(name ...string) (redis redisx.Config) {
	var is bool
	if len(name) == 0 {
		redis, is = rediss[GetDefaultKey()]
		if !is {
			panic(GetDefaultKey() + " redis config not found")
		}
	} else {
		redis, is = rediss[name[0]]
		if !is {
			panic(name[0] + " redis config not found")
		}
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

func GetDb(name ...string) (db dbx.Config) {
	var is bool
	if len(name) == 0 {
		db, is = dbs[GetDefaultKey()]
		if !is {
			panic(GetDefaultKey() + " db config not found")
		}
	} else {
		db, is = dbs[name[0]]
		if !is {
			panic(name[0] + " db config not found")
		}
	}
	return
}

func SetMongos(m map[string]mongox.Config) {
	mongos = m
}

func GetMongos() map[string]mongox.Config {
	return mongos
}

func GetMongo(name ...string) (mgo mongox.Config) {
	var is bool
	if len(name) == 0 {
		mgo, is = mongos[GetDefaultKey()]
		if !is {
			panic(GetDefaultKey() + " mongo config not found")
		}
	} else {
		mgo, is = mongos[name[0]]
		if !is {
			panic(name[0] + " mongo config not found")
		}
	}
	return
}

func SetHttp(h map[string]httpx.Config) {
	http = h
}

func GetHttp() map[string]httpx.Config {
	return http
}

func GetHttpServer(name ...string) (cfg httpx.Config) {
	var is bool
	if len(name) == 0 {
		cfg, is = http[GetDefaultKey()]
		if !is {
			panic(GetDefaultKey() + " http config not found")
		}
	} else {
		cfg, is = http[name[0]]
		if !is {
			panic(name[0] + " http config not found")
		}
	}
	return
}
