package httpx

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/wjoj/tool/v2/log"
)

type ResponseData struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	UUID string `json:"uuid"`
	Data any    `json:"data"`
}

type Bind interface {
	Name() string
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
			Msg:  er[0].Error(),
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

func HandleWithParams[P any](f func(g *gin.Context, parameter P) (data any, err error), binds ...Bind) func(g *gin.Context) {
	return func(g *gin.Context) {
		var req P
		if err := bindParemeter(g, &req, binds...); err != nil {
			return
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

type HandleType[A any] func(g *gin.Context, auth A) (data any, err error)
type HandleResType[A any] func(g *gin.Context, auth A) func(res ...any)

type HandleT[A any] interface {
	HandleType[A] | HandleResType[A]
}

func HandleAuthT[A any, F HandleT[A], Ag AuthInfG[A]](f F, binds ...Bind) func(g *gin.Context) {
	return func(g *gin.Context) {
		auth := new(A)
		var a Ag = auth
		if err := a.Resolver(g); err != nil {
			Fail(g, err)
			return
		}
		var req any
		if err := bindParemeter(g, &req, binds...); err != nil {
			return
		}
		var fn any = f
		switch fnc := fn.(type) {
		case HandleType[A]:
			data, err := fnc(g, *auth)
			if err != nil {
				Fail(g, err)
				return
			}
			Success(g, data)
		case HandleResType[A]:
			fnc(g, *auth)(resf(g))
		}
	}
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

func HandleWithAuthAndResponse[A any, Ag AuthInfG[A], R any, Rinf ResponseInf[R]](f func(g *gin.Context, auth A, res *R)) func(g *gin.Context) {
	return func(g *gin.Context) {
		auth := new(A)
		var a Ag = auth
		if err := a.Resolver(g); err != nil {
			Fail(g, err)
			return
		}
		res := new(R)
		var r Rinf = res
		f(g, *auth, r)
		handleJSONResponse(g, r)
	}
}

func HandleWithParamsAndResponse[P any, R any, Rinf ResponseInf[R]](f func(g *gin.Context, parameter P, res *R), binds ...Bind) func(g *gin.Context) {
	return func(g *gin.Context) {
		var req P
		if err := bindParemeter(g, &req, binds...); err != nil {
			return
		}
		res := new(R)
		var r Rinf = res
		f(g, req, r)
		handleJSONResponse(g, r)
	}
}

func bindParemeter(g *gin.Context, req any, binds ...Bind) error {
	for _, bind := range binds {
		switch bid := bind.(type) {
		case binding.BindingUri:
			if err := g.ShouldBindUri(req); err != nil {
				Fail(g, err)
				return err
			}
		case binding.BindingBody:
			if err := g.ShouldBindBodyWith(req, bid); err != nil {
				Fail(g, err)
				return err
			}
		case binding.Binding:
			if err := g.ShouldBindWith(req, bid); err != nil {
				Fail(g, err)
				return err
			}
		default:
			err := errors.New("not found bind")
			Fail(g, err)
			return err
		}
	}
	return nil
}

type Res struct {
	Data       any
	HttpStatus HttpStatus
	Code       ErrCodeType
	Msg        string
}

type HttpStatus int
type MsgType string
type FailType string
type ResponseInfc interface {
	Data() any
	Msg() string
	IsLog() bool
	Code() ErrCodeType
	Uid() string
	Success(data any, msg ...string)
	Fail(msg string)
	FailDataCode(data any, msg string, code ErrCodeType)
	Error(err error)
	ErrorDataCode(data any, err error, code ErrCodeType)
	FailLog(msg string)
	FailDataCodeLog(data any, msg string, code ErrCodeType)
	Abort()
	IsAbort() bool
}
type ResponseInf[R any] interface {
	*R
	ResponseInfc
}
type ResData struct {
	code    ErrCodeType
	msg     string
	isLog   bool
	data    any
	isAbort bool
}

func (res ResData) Data() any {
	return res.data
}
func (res ResData) Msg() string {
	return res.msg
}
func (res ResData) IsLog() bool {
	return res.isLog
}
func (res ResData) Code() ErrCodeType {
	return res.code
}
func (res ResData) Uid() string {
	return uuid()
}

func (res *ResData) Success(data any, msg ...string) {
	res.data = data
	if len(msg) > 0 {
		res.msg = msg[0]
	} else {
		res.msg = ErrCodeTypeSuccess.Error()
	}
	res.code = ErrCodeTypeSuccess
}

func (res *ResData) Fail(msg string) {
	res.msg = msg
	res.code = ErrCodeTypeFail
}
func (res *ResData) FailDataCode(data any, msg string, code ErrCodeType) {
	res.data = data
	res.msg = msg
	res.code = code
}

func (res *ResData) FailLog(msg string) {
	res.msg = msg
	res.code = ErrCodeTypeFail
	res.isLog = true
}

func (res *ResData) FailDataCodeLog(data any, msg string, code ErrCodeType) {
	res.data = data
	res.msg = msg
	res.code = code
	res.isLog = true
}

func (res *ResData) Error(err error) {
	if err != nil {
		res.msg = err.Error()
	} else {
		res.msg = ErrCodeTypeFail.Error()
	}
	res.code = ErrCodeTypeFail
	res.isLog = true
}
func (res *ResData) ErrorDataCode(data any, err error, code ErrCodeType) {
	res.data = data
	if err != nil {
		res.msg = err.Error()
	} else {
		res.msg = code.Error()
	}
	res.code = code
	res.isLog = true
}

func (res *ResData) Abort() {
	res.isAbort = true
}

func (res *ResData) IsAbort() bool {
	return res.isAbort
}

func resf(ctx *gin.Context) func(res ...any) {
	return func(res ...any) {
		re := Res{
			HttpStatus: http.StatusOK,
			Code:       ErrCodeTypeSuccess,
			Msg:        ErrCodeTypeSuccess.Error(),
		}
		for i := range res {
			resdata := res[i]
			switch data := resdata.(type) {
			case HttpStatus:
				re.HttpStatus = data
			case ErrCodeType:
				re.Code = data
				re.Msg = data.Error()
			case int:
				re.Code = ErrCodeType(data)
			case MsgType:
				re.Msg = string(data)
			case validator.ValidationErrors:
				re.Code = ErrCodeTypeFail
				// re.Msg = data[0].Translate(TransZh)
			case *ErrCodeType:
				re.Code = *data
				re.Msg = data.Error()
			case ErrMsgData:
				re.Code = ErrCodeType(data.GetCode())
				re.Msg = data.Error()
			case *ErrMsgData:
				re.Code = ErrCodeType(data.GetCode())
				re.Msg = data.Error()
			case FailType:
				re.Code = ErrCodeTypeFail
				re.Msg = string(data)
			case error:
				re.Code = ErrCodeTypeFail
				re.Msg = data.Error()
			case Res:
				re = data
			case *Res:
				re = *data
			default:
				re.Data = data
			}
		}
		Json(ctx, int(re.HttpStatus), ResponseData{
			Code: int(re.Code),
			Msg:  re.Msg,
			Data: re.Data,
			UUID: uuid(),
		})
	}
}

func HandleAuthParameter[A any, AInf AuthInfG[A], P any](f func(g *gin.Context, auth A, parameter P) (data any, err error), binds ...Bind) func(g *gin.Context) {
	return func(g *gin.Context) {
		auth := new(A)
		var a AInf = auth
		if err := a.Resolver(g); err != nil {
			Fail(g, err)
			return
		}
		var req P
		if err := bindParemeter(g, &req, binds...); err != nil {
			return
		}
		data, err := f(g, *auth, req)
		if err != nil {
			Fail(g, err)
			return
		}
		Success(g, data)
	}
}
func HandleWithAuthParamsAndResponse[A any, AInf AuthInfG[A], P any, R any, Rinf ResponseInf[R]](f func(g *gin.Context, auth A, parameter P, res *R), binds ...Bind) func(g *gin.Context) {
	return func(g *gin.Context) {
		auth := new(A)
		var a AInf = auth
		if err := a.Resolver(g); err != nil {
			Fail(g, err)
			return
		}
		var req P
		if err := bindParemeter(g, &req, binds...); err != nil {
			return
		}
		res := new(R)
		var r Rinf = res
		f(g, *auth, req, r)
		handleJSONResponse(g, r)
	}
}

func handleJSONResponse(g *gin.Context, r ResponseInfc) {
	if r.IsAbort() {
		return
	}
	msg := r.Msg()
	if i18nc, is := g.Get(ContextKeyI18n); is && !r.IsLog() {
		if len(r.Msg()) == 0 {
			log.Warnf("i18n msgId is empty")
		} else if i18n, ok := i18nc.(I18nInf); ok {
			msg = i18n.Translate(g, r.Msg())
		}
	}
	Json(g, http.StatusOK, ResponseData{
		Code: int(r.Code()),
		Msg:  msg,
		Data: r.Data(),
		UUID: uuid(),
	})
}

type AuthParameter[A any, P any] struct {
	Context   *gin.Context
	Auth      A
	Parameter P
}

func HandleWithAuthAndParams[A any, AInf AuthInfG[A], P any](f func(p AuthParameter[A, P]) (data any, err error), binds ...binding.Binding) func(g *gin.Context) {
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
		data, err := f(AuthParameter[A, P]{
			Context:   g,
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
