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

func RPCPrometheusStart(cfg *ConfigPrometheus) (his *prometheus.HistogramVec, c *prometheus.CounterVec) {
	if cfg == nil {
		return
	}
	if cfg.Port == 0 {
		cfg.Port = 1000
	}
	if len(cfg.Path) == 0 {
		cfg.Path = "/metrics"
	}
	go func() {
		SetprometheusOpen(true)
		http.Handle(cfg.Path, promhttp.Handler()) //默认
		fmt.Println("" + fmt.Sprintf("Prometheus start port:%d path:%s", cfg.Port, cfg.Path))
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil); err != nil {
			panic(fmt.Sprintf("Prometheus start error	%v", err))
		}
	}()
	return NewRequestsHistogramVec(cfg.Namespace), NewRequestsCounterVec(cfg.Namespace)
}

func NewRequestsHistogramVec(namespace string) *prometheus.HistogramVec {
	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "requests",
		Name:      "duration_ms",
		Help:      "rpc server requests duration(ms).",
		Buckets:   []float64{5, 10, 25, 50, 100, 250, 500, 1000},
	}, []string{"method"})
	prometheus.MustRegister(vec)
	return vec
}

func NewRequestsCounterVec(namespace string) *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "requests",
		Name:      "code_total",
		Help:      "rpc server requests code count.",
	}, []string{"method", "code"})
	prometheus.MustRegister(vec)
	return vec
}

func UnaryRPCServerPrometheusInterceptor(his *prometheus.HistogramVec, ctr *prometheus.CounterVec) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if his == nil || ctr == nil {
			return handler(ctx, req)
		}
		startTime := time.Now()
		resp, err = handler(ctx, req)
		his.WithLabelValues(info.FullMethod).Observe(float64(time.Since(startTime) / time.Millisecond))
		ctr.WithLabelValues(info.FullMethod, strconv.Itoa(int(status.Code(err)))).Inc()
		return resp, err
	}
}

func UnaryRPCClientPrometheusInterceptor(his *prometheus.HistogramVec, ctr *prometheus.CounterVec) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if his == nil || ctr == nil {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		startTime := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		his.WithLabelValues(method).Observe(float64(time.Since(startTime) / time.Millisecond))
		ctr.WithLabelValues(method, strconv.Itoa(int(status.Code(err)))).Inc()
		return err
	}
}

//http
func HTTPPrometheusStart(cfg *ConfigPrometheus) (*prometheus.HistogramVec, *prometheus.CounterVec) {
	if cfg == nil {
		return nil, nil
	}
	if cfg.Port == 0 {
		cfg.Port = 1000
	}
	if len(cfg.Path) == 0 {
		cfg.Path = "/metrics"
	}
	go func() {
		http.Handle(cfg.Path, promhttp.Handler()) //默认
		fmt.Println("" + fmt.Sprintf("Prometheus start port:%d path:%s", cfg.Port, cfg.Path))
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil); err != nil {
			panic(fmt.Sprintf("Prometheus start error	%v", err))
		}
	}()
	return NewMetricHttpServerReqDur(cfg.Namespace), NewMetricHttpServerReqCodeTotal(cfg.Namespace)
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
		Namespace: namespace,
		Subsystem: "http_requests",
		Name:      "code_total",
		Help:      "http server requests code count.",
	}, []string{"path", "code"})
	prometheus.MustRegister(vec)
	return vec
}

func HttpGinPrometheusMiddleware(his *prometheus.HistogramVec, ctr *prometheus.CounterVec) func(gin.Context) {
	return func(ctx gin.Context) {
		startTime := time.Now()
		ctx.Next()
		his.WithLabelValues(ctx.Request.RequestURI).Observe(float64(time.Since(startTime) / time.Millisecond))
		ctr.WithLabelValues(ctx.Request.RequestURI, strconv.Itoa(ctx.Writer.Status())).Inc()
	}
}

func HttpPrometheusMiddleware(his *prometheus.HistogramVec, ctr *prometheus.CounterVec) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()
			wcp := &HttpResponseWriter{
				Writer: w,
			}
			h.ServeHTTP(wcp, r)
			his.WithLabelValues(r.RequestURI).Observe(float64(time.Since(startTime) / time.Millisecond))
			ctr.WithLabelValues(r.RequestURI, strconv.Itoa(wcp.Code)).Inc()
		})
	}
}
