package mongox

import "github.com/wjoj/tool/utils"

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

// 设置要使用配置文件的key
func WithLogConfigKeysOption(keys ...string) Option {
	return func(c *Options) {
		c.defKey.Keys = keys
	}
}

func applyGenGormOptions(options ...Option) Options {
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
