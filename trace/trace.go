package trace

import (
	"context"
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

func NewTracer(cfg *TracerCfg, srvName string, isGlobal bool) (opentracing.Tracer, error) {
	switch cfg.Type {
	case TracerTypeJaeger:
		return NewJaeger(cfg, srvName, isGlobal)
	default:
		return NewZikpin(cfg, srvName, isGlobal)
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
		ctx, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
		if err != nil {
			tr := new(tracerSpanEmpty)
			return tr, req.Context()
		}

		span := tracer.StartSpan(req.RequestURI, ext.RPCServerOption(ctx))
		ctxc := opentracing.ContextWithSpan(req.Context(), span)
		return span, ctxc
	}
	tr := new(tracerSpanEmpty)
	return tr, req.Context()
}

func TracerHttpMiddleware() func(gin.Context) {
	return func(ctx gin.Context) {
		span, ctxc := TracerHttpFunc(ctx.Request)
		defer span.Finish()
		ctx.Request = ctx.Request.WithContext(ctxc)
		ctx.Next()
	}
}