package casbinx

import "github.com/wjoj/tool/v2/utils"

type Options struct {
	defKey *utils.DefaultKeys
}

type Option func(c *Options)

// 设置默认key
func WithDefaultKeyOption(key string) Option {
	return func(c *Options) {
		c.defKey.DefaultKey = key
	}
}

// 设置日志要使用配置文件的key
func WithLogConfigKeysOption(keys ...string) Option {
	return func(c *Options) {
		c.defKey.Keys = keys
	}
}

func applyOptions(options ...Option) Options {
	opts := Options{
		defKey: utils.DefaultKey,
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		option(&opts)
	}
	return opts
}
