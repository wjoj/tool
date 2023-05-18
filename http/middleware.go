package http

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wjoj/tool/monitoring"
	"github.com/wjoj/tool/trace"
)

type Headers struct {
	ops map[string]string
}

func NewHeaders() *Headers {
	return &Headers{ops: map[string]string{}}
}

func (h *Headers) AllowOrigin(o string) {
	h.ops["Access-Control-Allow-Origin"] = o
}

func (h *Headers) AllowMethods(m string) {
	h.ops["Access-Control-Allow-Methods"] = m
}

func (h *Headers) AllowHeaders(m string) {
	h.ops["Access-Control-Allow-Headers"] = m
}

func (h *Headers) AllowExposeHeaders(hd string) {
	h.ops["Access-Control-Expose-Headers"] = hd
}

func (h *Headers) MaxAge(ma int64) {
	h.ops["Access-Control-Max-Age"] = fmt.Sprintf("%d", ma)
}

func (h *Headers) AllowCredentials(is bool) {
	if is {
		h.ops["Access-Control-Allow-Credentials"] = "true"
	} else {
		h.ops["Access-Control-Allow-Credentials"] = "false"
	}
}

// application/json,charset=UTF-8  text/plain multipart/form-data application/x-www-form-urlencoded
func (h *Headers) ContentType(c string) {
	h.ops["Content-Type"] = c
}

func (h *Headers) Add(key, value string) {
	h.ops[key] = value
}

func (h *Headers) IsOrigin() bool {
	_, is := h.ops["Access-Control-Allow-Origin"]
	return is
}

func (h *Headers) Iteration(f func(string, string)) {
	for h, v := range h.ops {
		f(h, v)
	}
}

func MiddlewareCross(h *Headers) func(*gin.Context) {
	return func(ctx *gin.Context) {
		origin := ctx.Request.Header.Get("Origin")
		if origin != "" && h != nil {
			if !h.IsOrigin() {
				h.AllowOrigin(origin)
			}
		}
		if h != nil {
			h.Iteration(func(s1, s2 string) {
				ctx.Header(s1, s2)
				ctx.Writer.Header().Set(s1, s2)
			})
		}
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}

func MiddlewareGinTrace() func(*gin.Context) {
	return trace.MiddlewareHttpGin()
}

type prometheusM struct {
	his *prometheus.HistogramVec
	ctr *prometheus.CounterVec
}

var promM *prometheusM

func setGlobalPrometheusM(his *prometheus.HistogramVec, ctr *prometheus.CounterVec) {
	promM = &prometheusM{
		his: his,
		ctr: ctr,
	}
}

func isPrometheusOpen() bool {
	return promM != nil
}

func MiddlewareGinPrometheus() func(*gin.Context) {
	if !isPrometheusOpen() {
		return nil
	}
	return monitoring.MiddlewareHttpGinPrometheus(promM.his, promM.ctr)
}
