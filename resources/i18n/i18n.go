package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/gin-gonic/gin"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/wjoj/tool/v2/log"
	"github.com/wjoj/tool/v2/utils"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Language string   `json:"language"`
	Encoders []string `json:"encoders"`
	Locales  string   `json:"locales"`
}
type Bundle struct {
	*i18n.Bundle
}

func New(cfg *Config) (*Bundle, error) {
	if cfg.Language == "" {
		cfg.Language = "zh"
	}
	if len(cfg.Encoders) == 0 {
		cfg.Encoders = append(cfg.Encoders, []string{"json", "yaml"}...)
	}
	if cfg.Locales == "" {
		cfg.Locales = "locales"
	}
	bundle := i18n.NewBundle(language.Chinese)
	if cfg.Language == "en" {
		bundle = i18n.NewBundle(language.English)
	}

	for _, encoder := range cfg.Encoders {
		if encoder == "json" {
			bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
		} else if encoder == "yaml" {
			bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
		} else if encoder == "toml" {
			bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
		} else {
			return nil, fmt.Errorf("unsupported encoder: %s", encoder)
		}
	}
	files, err := os.ReadDir(cfg.Locales)
	if err != nil {
		return nil, fmt.Errorf("读取locales目录失败: %v", err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := filepath.Ext(file.Name())
		if ext == ".json" || ext == ".yaml" {
			bundle.MustLoadMessageFile(filepath.Join(cfg.Locales, file.Name()))
		}
	}
	return &Bundle{bundle}, nil
}

func (b *Bundle) Translate(c *gin.Context, msgId string) string {
	accept := c.GetHeader("Accept-Language")
	localizer := i18n.NewLocalizer(b.Bundle, accept) //翻译
	return localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: msgId,
	})
}

var bundle *Bundle
var cbMap = make(map[string]*Bundle)
var defaultKey = utils.DefaultKey.DefaultKey

func Init(cfgs map[string]Config, options ...Option) error {
	log.Info("init i18n")
	if len(cfgs) == 0 {
		log.Errorf("init i18n fail config is empty")
		return fmt.Errorf("i18n config is empty")
	}
	opt := applyGenGormOptions(options...)
	defaultKey = opt.defKey.DefaultKey
	var err error
	cbMap, err = utils.Init("i18n", defaultKey, opt.defKey.Keys, cfgs, func(cfg Config) (*Bundle, error) {
		return New(&cfg)
	}, func(c *Bundle) {
		bundle = c
	})
	if err != nil {
		log.Errorf("init i18n fail %v", err)
		return err
	}
	log.Info("init i18n success")
	return nil
}

func InitGlobal(cfg *Config) error {
	var err error
	bundle, err = New(cfg)
	if err != nil {
		return err
	}
	if bundle == nil {
		return fmt.Errorf("i18n bundle is nil")
	}
	return nil
}
func GetMap() map[string]*Bundle {
	return cbMap
}
func Get(key ...string) *Bundle {
	i18nc, err := utils.Get("i18n", defaultKey, func(s string) (*Bundle, bool) {
		cli, is := cbMap[s]
		return cli, is
	}, key...)
	if err != nil {
		panic(err)
	}
	return i18nc
}

func GetGinLocalizer(c *gin.Context, keys ...string) *i18n.Localizer {
	accept := c.GetHeader("Accept-Language")
	return i18n.NewLocalizer(Get(keys...).Bundle, accept)
}

func GinTranslate(c *gin.Context, messageID string, keys ...string) string {
	localizer := GetGinLocalizer(c, keys...)
	return localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID: messageID,
	})
}
