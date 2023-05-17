package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	IsShutdown   bool                         `json:"isShutdown" yaml:"isShutdown"`
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
		his, ctr := monitoring.HTTPPrometheusStart(cfg.Prom)
		switch hand := handler.(type) {
		case *gin.Engine:
			hand.Use(monitoring.HttpGinPrometheusMiddleware(his, ctr))
		default:
			panic("this type is not supported by prometheus")
		}
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
		srccp := srv
		g.Go(func() error {
			return srccp.ListenAndServe()
		})
	}
	go func() {
		if err := g.Wait(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			} else {
				errfunc(err)
			}
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
	srv := HTTPServer(c, handler)
	HTTPDone(errfunc, srv)
	if c.IsShutdown {
		go func() {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
			<-ch
			close(ch)
			fmt.Println("准关闭http服务")
			if err := srv.Shutdown(context.Background()); err != nil {
				fmt.Println("" + fmt.Sprintf("关闭错误%v", err))
			} else {
				fmt.Println("已关闭http服务")
			}
			os.Exit(0)
		}()
	}
}
