package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

func Read[T any](cfgRoot, cfgFile string, onFunc func(e fsnotify.Event)) (fcfg *viper.Viper, cfg T, err error) {
	cfgpath := filepath.Join(cfgRoot, cfgFile)
	ext := strings.ToLower(strings.Replace(filepath.Ext(cfgFile), ".", "", 1))
	if ext == "yml" {
		ext = "yaml"
	}
	fcfg = viper.New()
	fcfg.SetConfigFile(cfgpath)
	fcfg.SetConfigType(ext)
	if er := fcfg.ReadInConfig(); er != nil {
		err = errors.New("read config file failed: " + er.Error())
		return
	}
	tagName := ext
	if ext == "yml" {
		tagName = "yaml"
	}
	if er := fcfg.Unmarshal(&cfg, decoderTagName(tagName)); er != nil {
		err = errors.New("unmarshal config failed: " + er.Error())
		return
	}
	if onFunc == nil {
		return
	}
	fcfg.OnConfigChange(onFunc)
	fcfg.WatchConfig()
	return
}

func ReadSys(cfgRoot, cfgFile string) error {
	cfgpath := filepath.Join(cfgRoot, cfgFile)
	ext := strings.ToLower(strings.Replace(filepath.Ext(cfgFile), ".", "", 1))
	if ext == "yml" {
		ext = "yaml"
	}
	viper.SetConfigFile(cfgpath)
	viper.SetConfigType(ext)
	if err := viper.ReadInConfig(); err != nil {
		return errors.New("read config file failed: " + err.Error())
	}
	tagName := ext
	if ext == "yml" {
		tagName = "yaml"
	}
	var cfg *App
	if err := viper.Unmarshal(&cfg, decoderTagName(tagName)); err != nil {
		return errors.New("unmarshal config failed: " + err.Error())
	}
	if len(cfg.Env) == 0 {
		cfg.Env = "dev"
	}
	envstr := os.Getenv("ENV")
	if len(envstr) != 0 {
		cfg.Env = EnvType(envstr)
	}

	SetEnv(cfg.Env)
	SetConfigRoot(cfgRoot)
	SetConfigFile(cfgFile)
	SetNamespace(cfg.Namespace)
	SetLog(cfg.Logs)
	SetRediss(cfg.Rediss)
	SetDbs(cfg.Dbs)
	SetMongos(cfg.Mongos)
	SetHttp(cfg.Http)
	SetCasbins(cfg.Casbins)
	SetJwts(cfg.Jwts)
	SetI18ns(cfg.I18ns)
	viper.OnConfigChange(func(e fsnotify.Event) { // 监听配置文件修改
		fmt.Printf("config file changed:%+v\n", e)
	})
	viper.WatchConfig()
	return nil
}

func decoderTagName(tag string) viper.DecoderConfigOption {
	return func(dc *mapstructure.DecoderConfig) {
		dc.TagName = tag
	}
}
