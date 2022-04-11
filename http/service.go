package http

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

type HTTP struct {
	Port         int           `json:"port" yaml:"port"`
	ReadTimeout  time.Duration `json:"readTimeout" yaml:"readTimeout"`   //单位秒 0表示不超时
	WriteTimeout time.Duration `json:"writeTimeout" yaml:"writeTimeout"` //单位秒 0表示不超时
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

func DefaultHTTPServer(env string, errfunc func(err error), cfg *HTTP, handler http.Handler) {
	if cfg == nil {
		errfunc(errors.New("the HTTP service configuration is empty"))
		return
	}
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	HTTPDone(errfunc, HTTPServer(env, cfg, handler))
}
