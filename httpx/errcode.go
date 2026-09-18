package httpx

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type ErrMsgData struct {
	Code ErrCodeType `json:"code"`
	Msg  string      `json:"msg"`
	Data any         `json:"data"`
}

// 实现Error接口，返回ErrMsgData结构体的Msg字段
func (e ErrMsgData) Error() string {
	return e.Msg
}

func (e ErrMsgData) GetCode() int {
	return int(e.Code)
}

func (e ErrMsgData) GetData() any {
	return e.Data
}

type ErrCodeType int

const (
	ErrCodeTypeSuccess ErrCodeType = 0
	ErrCodeTypeFail    ErrCodeType = 10000 + iota
)

func (e ErrCodeType) Error() string {
	if e == ErrCodeTypeSuccess {
		return "success"
	}
	return "fail"
}

func (e ErrCodeType) String() string {
	if e == ErrCodeTypeSuccess {
		return "success"
	}
	return "fail"
}

func (e ErrCodeType) GetCode() int {
	return int(e)
}

func (e ErrCodeType) SetError(msg error) ErrMsgData {
	return ErrMsgData{
		Code: e,
		Msg:  msg.Error(),
		Data: nil,
	}
}
func (e ErrCodeType) SetData(data any) ErrMsgData {
	return ErrMsgData{
		Code: e,
		Msg:  "",
		Data: data,
	}
}

func (e ErrCodeType) SetMsg(msg string, msgs ...string) ErrMsgData {
	if len(msgs) > 0 {
		msg = fmt.Sprintf(msg, msgs)
	}
	return ErrMsgData{
		Code: e,
		Msg:  msg,
		Data: nil,
	}
}

func (e ErrCodeType) I18nMsg(g *gin.Context, msg string, msgs ...string) ErrMsgData {
	return e.I18nDataMsg(g, nil, msg, msgs...)
}

func (e ErrCodeType) I18nDataMsg(g *gin.Context, data any, msg string, msgs ...string) ErrMsgData {
	if len(msgs) > 0 {
		msg = fmt.Sprintf(msg, msgs)
	}
	if i18nc, is := g.Get(ContextKeyI18n); is {
		if i18n, ok := i18nc.(I18nInf); ok {
			msg = i18n.Translate(g, msg)
		}
	}
	return ErrMsgData{
		Code: e,
		Msg:  msg,
		Data: data,
	}
}
