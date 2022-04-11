package tool

import (
	"reflect"
	"regexp"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	"github.com/go-playground/locales/zh_Hant_TW"
	ut "github.com/go-playground/universal-translator"
	validator "gopkg.in/go-playground/validator.v9"
	en_translations "gopkg.in/go-playground/validator.v9/translations/en"
	zh_translations "gopkg.in/go-playground/validator.v9/translations/zh"
	zh_tw_translations "gopkg.in/go-playground/validator.v9/translations/zh_tw"
)

const (
	ValidatorLocaleZh   = "zh"
	ValidatorLocaleEn   = "en"
	ValidatorLocaleZhTw = "zh_tw"
)

func LoadValidator() (*validator.Validate, *ut.UniversalTranslator, ut.Translator) {
	en := en.New()
	zh := zh.New()
	zh_tw := zh_Hant_TW.New()
	valir := validator.New()
	Uni := ut.New(en, zh, zh_tw)
	valir.SetTagName("validate")
	locale := ValidatorLocaleZh
	trans, _ := Uni.GetTranslator(locale)
	switch locale {
	case ValidatorLocaleZh:
		zh_translations.RegisterDefaultTranslations(valir, trans)

	case ValidatorLocaleEn:
		en_translations.RegisterDefaultTranslations(valir, trans)

	case ValidatorLocaleZhTw:
		zh_tw_translations.RegisterDefaultTranslations(valir, trans)

	default:
		zh_translations.RegisterDefaultTranslations(valir, trans)

	}

	valir.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		if ut.Locale() == ValidatorLocaleZh {
			return ut.Add("required", "{0} is must value!", true)
		}
		return ut.Add("required", "{0}为必须值!", true) // see universal-translator for details
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Field())
		return t
	})
	valir.RegisterTranslation("username", trans, func(ut ut.Translator) error {
		if ut.Locale() == ValidatorLocaleZh {
			return ut.Add("username", "{0} is a combination of English and numeric characters and contains 3 to 16 characters!", true)
		}
		return ut.Add("username", "{0}由英文和数字字符组合，长度为3到16个字符!", true) // see universal-translator for details
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("username", fe.Field())
		return t
	})
	valir.RegisterTranslation("pwd", trans, func(ut ut.Translator) error {
		if ut.Locale() == ValidatorLocaleZh {
			return ut.Add("pwd", "the {0} contains 4 to 16 characters, including digits, underscores (_), and hyphens (-)!!", true)
		}
		return ut.Add("pwd", "{0}由英文和数字、下划线、横线字符组合，长度为4到16个字符!", true) // see universal-translator for details
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("pwd", fe.Field())
		return t
	})
	valir.RegisterTranslation("datetime", trans, func(ut ut.Translator) error {
		if ut.Locale() == ValidatorLocaleZh {
			return ut.Add("pwd", "the {0} Is the date format, for example:2012-02-05", true)
		}
		return ut.Add("datetime", "{0}为日期格式，如：2012-02-05", true) // see universal-translator for details
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("datetime", fe.Field())
		return t
	})
	valir.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		reg := regexp.MustCompile(`^[a-z0-9_-]{3,16}$`)
		return reg.MatchString(fl.Field().String())
	})
	valir.RegisterValidation("pwd", func(fl validator.FieldLevel) bool {
		reg := regexp.MustCompile(`^[a-zA-Z0-9]{4,16}$`)
		return reg.MatchString(fl.Field().String())
	})
	valir.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
		reg := regexp.MustCompile(`^1([38][0-9]|14[579]|5[^4]|16[6]|7[1-35-8]|9[189])\d{8}$`)
		return reg.MatchString(fl.Field().String())
	})
	valir.RegisterValidation("datetime", func(fl validator.FieldLevel) bool {
		reg := regexp.MustCompile(`^\d{4}\-(0?[1-9]|[1][012])\-(0?[1-9]|[12][0-9]|3[01])$`)
		return reg.MatchString(fl.Field().String())
	})
	valir.RegisterTagNameFunc(func(field reflect.StructField) string {
		return field.Tag.Get("name")
	})
	return valir, Uni, trans
}
