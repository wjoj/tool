package trace

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

type tracerSpanEmpty struct {
}

func (m *tracerSpanEmpty) Finish() {

}

func (m *tracerSpanEmpty) FinishWithOptions(opts opentracing.FinishOptions) {

}

func (m *tracerSpanEmpty) Context() opentracing.SpanContext {
	return nil
}

func (m *tracerSpanEmpty) SetOperationName(operationName string) opentracing.Span {
	return nil
}

func (m *tracerSpanEmpty) SetTag(key string, value interface{}) opentracing.Span {
	return nil
}

func (m *tracerSpanEmpty) LogFields(fields ...log.Field) {

}
func (m *tracerSpanEmpty) LogKV(alternatingKeyValues ...interface{}) {

}

func (m *tracerSpanEmpty) SetBaggageItem(restrictedKey, value string) opentracing.Span {
	return nil
}

func (m *tracerSpanEmpty) BaggageItem(restrictedKey string) string {
	return ""
}

func (m *tracerSpanEmpty) Tracer() opentracing.Tracer {
	return nil
}
func (m *tracerSpanEmpty) LogEvent(event string) {

}
func (m *tracerSpanEmpty) LogEventWithPayload(event string, payload interface{}) {

}

func (m *tracerSpanEmpty) Log(data opentracing.LogData) {

}
