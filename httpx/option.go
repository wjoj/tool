package httpx

import (
	"github.com/gin-gonic/gin"
	"github.com/wjoj/tool/v2/utils"
)

type Options struct {
	defKey   *utils.DefaultKeys
	engFuncs map[string][]func(eng *gin.Engine)
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
		defKey:   utils.DefaultKey,
		engFuncs: make(map[string][]func(eng *gin.Engine)),
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		option(&opts)
	}
	return opts
}
