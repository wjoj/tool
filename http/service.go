package http

import (
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
	Port         int           `json:"port" yaml:"port"`
	ReadTimeout  time.Duration `json:"readTimeout" yaml:"readTimeout"`   //单位秒 0表示不超时
	WriteTimeout time.Duration `json:"writeTimeout" yaml:"writeTimeout"` //单位秒 0表示不超时
	Trace        *trace.TracerCfg
	Prom         *monitoring.ConfigPrometheus `json:"prom" yaml:"prom"`
}

type Base interface {
	Load(g *gin.RouterGroup)
}

var (
	g errgroup.Group
)

func HTTPServer(env string, cfg *HTTP, handler http.Handler) *http.Server {
	gin.SetMode(env)
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
		srcCp := new(http.Server)
		*srcCp = *srv
		g.Go(func() error {
			return srcCp.ListenAndServe()
		})
	}
	go func() {
		if err := g.Wait(); err != nil {
			errfunc(err)
		}
	}()
}

func (c *HTTP) Start(env string, errfunc func(err error), handler http.Handler) {
	if c == nil {
		errfunc(errors.New("the HTTP service configuration is empty"))
		return
	}
	if c.Port == 0 {
		c.Port = 8080
	}
	HTTPDone(errfunc, HTTPServer(env, c, handler))
}
