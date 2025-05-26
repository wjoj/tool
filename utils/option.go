package utils

var (
	DefaultKey = &DefaultKeys{
		DefaultKey: "def",
		Keys:       []string{},
	}
)

type DefaultKeys struct {
	DefaultKey string
	Keys       []string
}

type DefaultOption func(c *DefaultKeys)

// 设置默认key
func WithDefaultKeyOption(key string) DefaultOption {
	return func(c *DefaultKeys) {
		c.DefaultKey = key
	}
}

// 设置日志要使用配置文件的key
func WithLogConfigKeysOption(keys ...string) DefaultOption {
	return func(c *DefaultKeys) {
		c.Keys = keys
	}
}
