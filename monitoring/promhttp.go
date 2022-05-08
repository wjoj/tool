package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type ConfigPrometheus struct {
	Port      int    `json:"port" yaml:"port"`
	Path      string `json:"path" yaml:"path"`
	Namespace string `json:"namespace" yaml:"namespace"`
}

func (c *ConfigPrometheus) Show() {
	msg := ""
	msg += fmt.Sprintf("Prometheus port: %v path: %v namespace: %v", c.Port, c.Path, c.Namespace)
	fmt.Println(msg)
}

var prometheusOpen int32

func GetPrometheusOpen() bool {
	if atomic.LoadInt32(&prometheusOpen) == 0 {
		return !true
	}
	return true
}

func SetprometheusOpen(is bool) {
	if is {
		prometheusOpen = 1
	} else {
		prometheusOpen = 0
	}
}

func RPCPrometheusStart(cfg *ConfigPrometheus) {
	if cfg == nil {
		return
	}
	if cfg.Port == 0 {
		cfg.Port = 1000
	}
	if len(cfg.Path) == 0 {
		cfg.Path = "/metrics"
	}
	SetPrometheusNamespace(cfg.Namespace)
	metricRPCReqDur = NewRequestsHistogramVec()
	metricRPCReqCodeTotal = NewRequestsCounterVec()
	SetprometheusOpen(true)
	http.Handle(cfg.Path, promhttp.Handler()) //默认
	fmt.Println("" + fmt.Sprintf("Prometheus start port:%d path:%s", cfg.Port, cfg.Path))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil); err != nil {
		panic(fmt.Sprintf("Prometheus start error	%v", err))
	}
}

func PrometheusStartCustom() {
	SetprometheusOpen(true)
	registry := prometheus.NewRegistry()
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry}))
	http.ListenAndServe(":8081", nil)
}

var (
	metricRPCReqDur       *prometheus.HistogramVec
	metricRPCReqCodeTotal *prometheus.CounterVec
)

var prometheusNamespace = ""

func SetPrometheusNamespace(n string) {
	prometheusNamespace = n
}

func NewRequestsHistogramVec() *prometheus.HistogramVec {
	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: prometheusNamespace,
		Subsystem: "requests",
		Name:      "duration_ms",
		Help:      "rpc server requests duration(ms).",
		Buckets:   []float64{5, 10, 25, 50, 100, 250, 500, 1000},
	}, []string{"method"})
	prometheus.MustRegister(vec)
	return vec
}

func NewRequestsCounterVec() *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: prometheusNamespace,
		Subsystem: "requests",
		Name:      "code_total",
		Help:      "rpc server requests code count.",
	}, []string{"method", "code"})
	prometheus.MustRegister(vec)
	return vec
}

func UnaryRPCServerPrometheusInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if !GetPrometheusOpen() {
		return handler(ctx, req)
	}

	startTime := time.Now()
	resp, err := handler(ctx, req)
	metricRPCReqDur.WithLabelValues(info.FullMethod).Observe(float64(time.Since(startTime) / time.Millisecond))
	metricRPCReqCodeTotal.WithLabelValues(info.FullMethod, strconv.Itoa(int(status.Code(err)))).Inc()
	return resp, err
}

func UnaryRPCClientPrometheusInterceptor(ctx context.Context, method string, req, reply interface{},
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if !GetPrometheusOpen() {
		return invoker(ctx, method, req, reply, cc, opts...)
	}
	startTime := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	metricRPCReqDur.WithLabelValues(method).Observe(float64(time.Since(startTime) / time.Millisecond))
	metricRPCReqCodeTotal.WithLabelValues(method, strconv.Itoa(int(status.Code(err)))).Inc()
	return err
}

//http
var (
	metricHttpServerReqDur       *prometheus.HistogramVec
	metricHttpServerReqCodeTotal *prometheus.CounterVec
)

func HTTPPrometheusStart(cfg *ConfigPrometheus) {
	if cfg == nil {
		return
	}
	if cfg.Port == 0 {
		cfg.Port = 1000
	}
	if len(cfg.Path) == 0 {
		cfg.Path = "/metrics"
	}
	metricHttpServerReqDur = NewMetricHttpServerReqDur(cfg.Namespace)
	metricHttpServerReqCodeTotal = NewMetricHttpServerReqCodeTotal(cfg.Namespace)
	http.Handle(cfg.Path, promhttp.Handler()) //默认
	fmt.Println("" + fmt.Sprintf("Prometheus start port:%d path:%s", cfg.Port, cfg.Path))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil); err != nil {
		panic(fmt.Sprintf("Prometheus start error	%v", err))
	}
}

func NewMetricHttpServerReqDur(namespace string) *prometheus.HistogramVec {
	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "http_requests",
		Name:      "duration_ms",
		Help:      "http server requests duration(ms).",
		Buckets:   []float64{5, 10, 25, 50, 100, 250, 500, 1000},
	}, []string{"path"})
	prometheus.MustRegister(vec)
	return vec
}

func NewMetricHttpServerReqCodeTotal(namespace string) *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: prometheusNamespace,
		Subsystem: "http_requests",
		Name:      "code_total",
		Help:      "http server requests code count.",
	}, []string{"path", "code"})
	prometheus.MustRegister(vec)
	return vec
}

func HttpGinPrometheusMiddleware() func(gin.Context) {
	return func(ctx gin.Context) {
		startTime := time.Now()
		ctx.Next()
		metricHttpServerReqDur.WithLabelValues(ctx.Request.RequestURI).Observe(float64(time.Since(startTime) / time.Millisecond))
		metricHttpServerReqCodeTotal.WithLabelValues(ctx.Request.RequestURI, strconv.Itoa(ctx.Writer.Status())).Inc()
	}
}

func HttpPrometheusMiddleware() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()
			wcp := &HttpResponseWriter{
				Writer: w,
			}
			h.ServeHTTP(wcp, r)
			metricHttpServerReqDur.WithLabelValues(r.RequestURI).Observe(float64(time.Since(startTime) / time.Millisecond))
			metricHttpServerReqCodeTotal.WithLabelValues(r.RequestURI, strconv.Itoa(wcp.Code)).Inc()
		})
	}
}
