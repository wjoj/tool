package trace

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"google.golang.org/grpc"
)

type TracerType string

const (
	TracerTypeZikpin = "zikpin"
	TracerTypeJaeger = "jaeger"
)

type TracerCfg struct {
	Type        TracerType
	EndpointURL string
	HostURL     string
	IsOpen      bool
	Kafkas      []string
}

func (c *TracerCfg) String() string {
	msg := "Trace Info:"
	msg += fmt.Sprintf("\n\tType: %v", c.Type)
	msg += fmt.Sprintf("\n\tEndpointURL: %v", c.EndpointURL)
	msg += fmt.Sprintf("\n\tHostURL: %v", c.HostURL)
	msg += fmt.Sprintf("\n\tIsOpen: %v", c.IsOpen)
	msg += fmt.Sprintf("\n\tKafkas: %v", c.Kafkas)
	return msg
}

func (c *TracerCfg) Show() {
	fmt.Println(c)
}

func NewTracer(cfg *TracerCfg, srvName string) (opentracing.Tracer, error) {
	switch cfg.Type {
	case TracerTypeJaeger:
		return NewJaeger(cfg, srvName, true)
	default:
		return NewZikpin(cfg, srvName, true)
	}
}

func TracerGrpcServerUnaryInterceptor(tracer opentracing.Tracer) grpc.UnaryServerInterceptor {
	return otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())
}

func TracerGrpcStreamServerUnaryInterceptor(tracer opentracing.Tracer) grpc.StreamServerInterceptor {
	return otgrpc.OpenTracingStreamServerInterceptor(tracer, otgrpc.LogPayloads())
}

func TracerGrpcClientUnaryInterceptor(tracer opentracing.Tracer) grpc.UnaryClientInterceptor {
	return otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())
}

func TracerGrpcStreamClientUnaryInterceptor(tracer opentracing.Tracer) grpc.StreamClientInterceptor {
	return otgrpc.OpenTracingStreamClientInterceptor(tracer, otgrpc.LogPayloads())
}

func GlobalTracer() opentracing.Tracer {
	return opentracing.GlobalTracer()
}

func GlobalTracerGrpcFunc(ctx context.Context, funcName string) opentracing.Span {
	if parent := opentracing.SpanFromContext(ctx); parent != nil {
		pctx := parent.Context()
		if tracer := opentracing.GlobalTracer(); tracer != nil {
			return tracer.StartSpan(funcName, opentracing.ChildOf(pctx))
		}
	}
	tr := new(tracerSpanEmpty)
	return tr
}

func TracerHttpFunc(req *http.Request) (opentracing.Span, context.Context) {
	if tracer := opentracing.GlobalTracer(); tracer != nil {
		var span opentracing.Span
		ctx, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
		if err != nil {
			span = tracer.StartSpan(req.URL.Path)
		} else {
			span = tracer.StartSpan(req.URL.Path, ext.RPCServerOption(ctx))
		}
		ctxc := opentracing.ContextWithSpan(req.Context(), span)
		return span, ctxc
	}
	tr := new(tracerSpanEmpty)
	return tr, req.Context()
}

func TracerHttpGinMiddleware() func(gin.Context) {
	return func(ctx gin.Context) {
		span, ctxc := TracerHttpFunc(ctx.Request)
		defer span.Finish()
		ctx.Request = ctx.Request.WithContext(ctxc)
		ctx.Next()
	}
}

func TracerHttpMiddleware() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span, ctxc := TracerHttpFunc(r)
			defer span.Finish()
			h.ServeHTTP(w, r.WithContext(ctxc))
		})
	}
}
