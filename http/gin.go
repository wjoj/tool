package http

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/wjoj/tool/trace"
)

type ginOptions struct {
	trace        bool
	monitoring   bool
	crossHeaders *Headers
}

type ginOption func(o *ginOptions)

func GinWithTrace(is bool) ginOption {
	return func(o *ginOptions) {
		o.trace = is
	}
}

func GinWithMonitoring(is bool) ginOption {
	return func(o *ginOptions) {
		o.monitoring = is
	}
}

func GinWithCrossHeaders(h *Headers) ginOption {
	return func(o *ginOptions) {
		o.crossHeaders = h
	}
}

func getGinOptions(opts ...ginOption) *ginOptions {
	opt := new(ginOptions)
	for _, oy := range opts {
		oy(opt)
	}
	return opt
}

var gGloabl *gin.Engine
var gOnce sync.Once

func GinGlobalEngine(opts ...ginOption) *gin.Engine {
	gOnce.Do(func() {
		gGloabl = gin.Default()
	})
	opt := getGinOptions(opts...)
	if opt.crossHeaders != nil {
		gGloabl.Use(MiddlewareCross(opt.crossHeaders))
	}
	if opt.trace {
		if !trace.IsGlobal() {
			panic("trace is not configured")
		}
		gGloabl.Use(MiddlewareGinTrace())
	}
	if opt.monitoring {
		if !isPrometheusOpen() {
			panic("prometheus is not configured")
		}
		gGloabl.Use(MiddlewareGinPrometheus())
	}
	return gGloabl
}

func GinRouterGroup(g *gin.RouterGroup, opts ...ginOption) *gin.RouterGroup {
	opt := getGinOptions(opts...)
	if opt.crossHeaders != nil {
		g.Use(MiddlewareCross(opt.crossHeaders))
	}
	if opt.trace {
		if !trace.IsGlobal() {
			panic("trace is not configured")
		}
		g.Use(MiddlewareGinTrace())
	}
	if opt.monitoring {
		if !isPrometheusOpen() {
			panic("prometheus is not configured")
		}
		g.Use(MiddlewareGinPrometheus())
	}
	return g
}
