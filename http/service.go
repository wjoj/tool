package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wjoj/tool/monitoring"
	"github.com/wjoj/tool/trace"
	"golang.org/x/sync/errgroup"
)

type HTTP struct {
	ServiceName  string
	Port         int                          `json:"port" yaml:"port"`
	ReadTimeout  time.Duration                `json:"readTimeout" yaml:"readTimeout"`   //单位秒 0表示不超时
	WriteTimeout time.Duration                `json:"writeTimeout" yaml:"writeTimeout"` //单位秒 0表示不超时
	Trace        *trace.TracerCfg             `json:"trace" yaml:"trace"`
	Prom         *monitoring.ConfigPrometheus `json:"prom" yaml:"prom"`
}

type HttpHandler interface {
	Name() string
	Handler() any
}

type Base interface {
	Load(g *gin.RouterGroup, handles ...HttpHandler)
}

var (
	g errgroup.Group
)

func HTTPServer(cfg *HTTP, handler http.Handler) *http.Server {
	http.NewServeMux()
	if cfg.Trace != nil {
		if len(cfg.ServiceName) == 0 {
			cfg.ServiceName = "http-server"
		}
		_, err := trace.NewTracer(cfg.Trace, cfg.ServiceName)
		if err != nil {
			panic(err)
		}
	}
	if cfg.Prom != nil {
		if len(cfg.Prom.Namespace) == 0 {
			cfg.Prom.Namespace = "http-server"
		}
		monitoring.HTTPPrometheusStart(cfg.Prom)
	}
	return &http.Server{
		Addr:         fmt.Sprintf(":%v", cfg.Port),
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout * time.Second,
		WriteTimeout: cfg.WriteTimeout * time.Second,
	}
}

func HTTPDone(errfunc func(err error), srvs ...*http.Server) {
	for _, srv := range srvs {
		if srv == nil {
			continue
		}
		g.Go(func() error {
			return srv.ListenAndServe()
		})

		srv.Shutdown(context.Background())
	}
	go func() {
		if err := g.Wait(); err != nil {
			errfunc(err)
		}
	}()
}

func (c *HTTP) Start(errfunc func(err error), handler http.Handler) {
	if c == nil {
		errfunc(errors.New("the HTTP service configuration is empty"))
		return
	}
	if c.Port == 0 {
		c.Port = 8080
	}
	HTTPDone(errfunc, HTTPServer(c, handler))
}
