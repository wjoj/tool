package httpx

import (
	"github.com/gin-gonic/gin"
	"github.com/wjoj/tool/v2/utils"
)

const ContextKeyI18n = "i18n"

type I18nInf interface {
	Translate(c *gin.Context, msgId string) string
}

type Options struct {
	defKey            *utils.DefaultKeys
	engFuncs          map[string][]func(eng *gin.Engine)
	beforeHandlerFunc map[string][]gin.HandlerFunc
	i18nKeys          map[string]string  //map[httpKey]=i18nKey
	i18nMap           map[string]I18nInf //map[i18nKey]=I18nInf
}

type Option func(c *Options)

// 设置默认key
func WithDefaultKeyOption(key string) Option {
	return func(c *Options) {
		c.defKey.DefaultKey = key
	}
}

// 设置要使用配置文件的key
func WithLogConfigKeysOption(keys ...string) Option {
	return func(c *Options) {
		c.defKey.Keys = keys
	}
}

func WithI18nEnableOption() Option {
	return func(c *Options) {
		c.i18nKeys[c.defKey.DefaultKey] = c.defKey.DefaultKey
	}
}

func WithI18nDisableOption() Option {
	return func(c *Options) {
		c.i18nKeys = make(map[string]string)
	}
}
func WithI18nKeyOption(key, i18nKey string) Option {
	return func(c *Options) {
		c.i18nKeys[key] = i18nKey
	}
}

func WithHandlerFuncOption(key string, fun func(ctx *gin.Context)) Option {
	return func(c *Options) {
		c.beforeHandlerFunc[key] = append(c.beforeHandlerFunc[key], fun)
	}
}

func WithSetI18nMapOption(key string, i18n I18nInf) Option {
	return func(c *Options) {
		c.i18nMap[key] = i18n
	}
}

// WithGinEngineFuncOption 创建一个Option函数，用于向Options中添加Gin引擎的处理函数
// 参数:
//
//	key: 用于标识这组处理函数的键
//	fs: 一个或多个处理Gin引擎的函数
//
// 返回值:
//
//	返回一个Option类型的函数，该函数会将处理函数添加到Options的engFuncs映射中
func WithGinEngineFuncOption(key string, fs ...func(eng *gin.Engine)) Option {
	return func(c *Options) {
		c.engFuncs[key] = fs
	}
}

func applyGenGormOptions(options ...Option) Options {
	opts := Options{
		defKey:            utils.DefaultKey,
		engFuncs:          make(map[string][]func(eng *gin.Engine)),
		beforeHandlerFunc: make(map[string][]gin.HandlerFunc),
		i18nKeys:          make(map[string]string),
		i18nMap:           make(map[string]I18nInf),
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		option(&opts)
	}
	return opts
}
