package httpx

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type ResponseData struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	UUID string `json:"uuid"`
	Data any    `json:"data"`
}

type Controller struct {
}

func (b Controller) Parameter(g *gin.Context, req any, binds ...binding.Binding) error {
	for i := range binds {
		if err := g.ShouldBindWith(req, binds[i]); err != nil {
			Fail(g, err)
			return err
		}
	}
	return nil
}

func (b Controller) Handle(c *gin.Context, f func() (data any, err error)) {
	data, err := f()
	if err != nil {
		Fail(c, err)
		return
	}
	Success(c, data)
}

func (b Controller) HandleParameter(c *gin.Context, req any, binds []binding.Binding, f func() (data any, err error)) {
	if err := b.Parameter(c, req, binds...); err != nil {
		return
	}
	b.Handle(c, f)
}

type AuthInf interface {
	Resolver(g *gin.Context) error
}

func (b Controller) Auth(g *gin.Context, auth AuthInf) error {
	if err := auth.Resolver(g); err != nil {
		Fail(g, err)
		return err
	}
	return nil
}
func (b Controller) HandleAuthParameter(c *gin.Context, auth AuthInf, req any, binds []binding.Binding, f func() (data any, err error)) {
	err := b.Auth(c, auth)
	if err != nil {
		return
	}
	b.HandleParameter(c, req, binds, f)
}
func Json(g *gin.Context, status int, data any) {
	g.JSON(status, data)
}

func Success(g *gin.Context, data any) {
	Json(g, 200, ResponseData{
		Code: ErrCodeTypeSuccess.GetCode(),
		Msg:  ErrCodeTypeSuccess.Error(),
		Data: data,
		UUID: uuid(),
	})
}

func Fail(g *gin.Context, err error) {
	if err == nil {
		err = errors.New("fail")
	}

	switch er := err.(type) {
	case validator.ValidationErrors:
		Json(g, http.StatusOK, ResponseData{
			Code: ErrCodeTypeFail.GetCode(),
			UUID: uuid(),
			// Msg:  er[0].Translate(TransZh),
			Data: nil,
		})
	case ErrCodeType:
		Json(g, http.StatusOK, ResponseData{
			Code: er.GetCode(),
			Msg:  er.Error(),
			Data: nil,
			UUID: uuid(),
		})
	case *ErrCodeType:
		Json(g, http.StatusOK, ResponseData{
			Code: er.GetCode(),
			Msg:  er.Error(),
			Data: nil,
			UUID: uuid(),
		})
	case ErrMsgData:
		Json(g, http.StatusOK, ResponseData{
			Code: er.GetCode(),
			Msg:  er.Error(),
			Data: er.GetData(),
			UUID: uuid(),
		})
	case *ErrMsgData:
		Json(g, http.StatusOK, ResponseData{
			Code: er.GetCode(),
			Msg:  er.Error(),
			Data: er.GetData(),
			UUID: uuid(),
		})
	default:
		Json(g, http.StatusOK, ResponseData{
			Code: ErrCodeTypeFail.GetCode(),
			Msg:  er.Error(),
			Data: nil,
			UUID: uuid(),
		})
	}
}

func uuid() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

func Handle(f func(*gin.Context) (data any, err error)) func(g *gin.Context) {
	return func(g *gin.Context) {
		data, err := f(g)
		if err != nil {
			Fail(g, err)
			return
		}
		Success(g, data)
	}
}

func HandleParameter[P any](f func(g *gin.Context, parameter P) (data any, err error), binds ...binding.Binding) func(g *gin.Context) {
	return func(g *gin.Context) {
		var req P
		for _, bind := range binds {
			if err := g.ShouldBindWith(&req, bind); err != nil {
				Fail(g, err)
				return
			}
		}
		data, err := f(g, req)
		if err != nil {
			Fail(g, err)
			return
		}
		Success(g, data)
	}
}

type AuthInfG[A any] interface {
	*A
	AuthInf
}

func HandleAuth[A any, Ag AuthInfG[A]](f func(g *gin.Context, auth A) (data any, err error)) func(g *gin.Context) {
	return func(g *gin.Context) {
		auth := new(A)
		var a Ag = auth
		if err := a.Resolver(g); err != nil {
			Fail(g, err)
			return
		}
		data, err := f(g, *auth)
		if err != nil {
			Fail(g, err)
			return
		}
		Success(g, data)
	}
}

func HandleAuthParameter[A any, AInf AuthInfG[A], P any](f func(g *gin.Context, auth A, parameter P) (data any, err error), binds ...binding.Binding) func(g *gin.Context) {
	return func(g *gin.Context) {
		auth := new(A)
		var a AInf = auth
		if err := a.Resolver(g); err != nil {
			Fail(g, err)
			return
		}
		var req P
		for _, bind := range binds {
			if err := g.ShouldBindWith(&req, bind); err != nil {
				Fail(g, err)
				return
			}
		}
		data, err := f(g, *auth, req)
		if err != nil {
			Fail(g, err)
			return
		}
		Success(g, data)
	}
}

type AuthParameter[A any, P any] struct {
	Auth      A
	Parameter P
}

func HandleAuthParameter2[A any, AInf AuthInfG[A], P any](f func(g *gin.Context, p AuthParameter[A, P]) (data any, err error), binds ...binding.Binding) func(g *gin.Context) {
	return func(g *gin.Context) {
		auth := new(A)
		var a AInf = auth
		if err := a.Resolver(g); err != nil {
			Fail(g, err)
			return
		}
		var req P
		for _, bind := range binds {
			if err := g.ShouldBindWith(&req, bind); err != nil {
				Fail(g, err)
				return
			}
		}
		data, err := f(g, AuthParameter[A, P]{
			Auth:      *auth,
			Parameter: req,
		})
		if err != nil {
			Fail(g, err)
			return
		}
		Success(g, data)
	}
}
