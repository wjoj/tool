package httpx

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
	return "fail"
}

func (e ErrCodeType) String() string {
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

func (e ErrCodeType) SetMsg(msg string) ErrMsgData {
	return ErrMsgData{
		Code: e,
		Msg:  msg,
		Data: nil,
	}
}
