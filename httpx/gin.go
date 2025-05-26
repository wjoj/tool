package httpx

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/wjoj/tool/v2/log"
	"github.com/wjoj/tool/v2/utils"
	"go.uber.org/zap"
)

type Config struct {
	Debug                bool          `yaml:"debug" json:"debug"`
	Log                  bool          `yaml:"log" json:"log"`
	LogName              string        `yaml:"logName" json:"logName"` //指定log的名称
	Port                 int           `yaml:"port" json:"port"`
	ShutdownCloseMaxWait time.Duration `yaml:"shutdownCloseMaxWait" json:"shutdownCloseMaxWait"` //
	Ping                 bool          `yaml:"ping" json:"ping"`
	Swagger              bool          `yaml:"swagger" json:"swagger"`         // 是否开启docs
	RoutePrefix          string        `yaml:"routePrefix" json:"routePrefix"` // 路由前缀
	Cors                 bool          `yaml:"cors" json:"cors"`               // 是否启用cors
	CorsCfg              CorsConfig    `yaml:"corsCfg" json:"corsCfg"`
}

type CorsConfig struct {
	AllowOrigins     []string       `yaml:"allowOrigins" json:"allowOrigins"`
	AllowMethods     []string       `yaml:"allowMethods" json:"allowMethods"`
	AllowHeaders     []string       `yaml:"allowHeaders" json:"allowHeaders"`
	ExposeHeaders    []string       `yaml:"exposeHeaders" json:"exposeHeaders"`
	AllowCredentials bool           `yaml:"allowCredentials" json:"allowCredentials"`
	MaxAge           utils.Duration `yaml:"maxAge" json:"maxAge"`
}

type Http struct {
	cfg *Config
	*gin.Engine
	srv  *http.Server
	done chan struct{}
}

func New(cfg *Config) (*Http, error) {
	var err error
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	if cfg.ShutdownCloseMaxWait == 0 {
		cfg.ShutdownCloseMaxWait = 5 * time.Minute
	}
	if len(cfg.LogName) == 0 {
		cfg.LogName = utils.DefaultKey.DefaultKey
	}
	if len(cfg.CorsCfg.MaxAge) == 0 {
		cfg.CorsCfg.MaxAge = "12h"
	}
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	var g *gin.Engine
	if cfg.Debug {
		g = gin.Default()
	} else {
		g = gin.New()
		if cfg.Log && cfg.LogName == "--" {
			g.Use(gin.Recovery())
			g.Use(gin.Logger())
		}
	}
	// g.Use(csrf.New())
	if cfg.Log && cfg.LogName != "--" {
		// g.Use(zapLogger(log.GetLogger(cfg.LogName).Desugar()))
		logc := log.GetLogger(cfg.LogName).Desugar()
		g.Use(ginzap.Ginzap(logc, time.RFC3339, true))
		g.Use(ginzap.RecoveryWithZap(logc, true))
	}
	if cfg.Cors {
		corsConfig := cors.DefaultConfig()
		if len(cfg.CorsCfg.AllowOrigins) > 0 {
			corsConfig.AllowAllOrigins = false
			corsConfig.AllowOrigins = cfg.CorsCfg.AllowOrigins
		} else {
			corsConfig.AllowAllOrigins = true // 允许所有来源
			corsConfig.AllowOriginWithContextFunc = func(c *gin.Context, origin string) bool {
				if len(cfg.CorsCfg.AllowOrigins) == 0 || len(origin) == 0 {
					return true
				}
				return slices.Contains(cfg.CorsCfg.AllowOrigins, origin)
			}
		}
		if len(cfg.CorsCfg.AllowMethods) > 0 {
			corsConfig.AllowMethods = cfg.CorsCfg.AllowMethods
		} else {
			corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
		}
		if len(cfg.CorsCfg.AllowHeaders) > 0 {
			corsConfig.AllowHeaders = cfg.CorsCfg.AllowHeaders
		} else {
			corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
		}
		corsConfig.AllowCredentials = cfg.CorsCfg.AllowCredentials
		corsConfig.ExposeHeaders = cfg.CorsCfg.ExposeHeaders
		corsConfig.MaxAge, err = cfg.CorsCfg.MaxAge.ParseDuration()
		if err != nil {
			return nil, fmt.Errorf("parse corsCfg.maxAge error: %v", err)
		}
		g.Use(cors.New(corsConfig))
	}

	if len(cfg.RoutePrefix) == 0 {
		g.RouterGroup = *g.Group(cfg.RoutePrefix)
	}
	if cfg.Ping {
		g.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})
	}
	if cfg.Swagger {
		g.RouterGroup.GET("docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
	return &Http{
		cfg:    cfg,
		Engine: g,
	}, nil
}

func (h *Http) Run(f func(eng *gin.Engine)) error {
	if f != nil {
		f(h.Engine)
	}
	h.srv = &http.Server{
		Addr:    ":" + strconv.Itoa(h.cfg.Port),
		Handler: h.Engine,
	}
	// 创建优雅关闭的channel
	h.done = make(chan struct{})
	go func() {
		// 启动服务
		if err := h.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			if h.cfg.Log && h.cfg.LogName != "--" {
				log.GetLogger(h.cfg.LogName).Errorf("listen: %s", err)
			}
		}
		close(h.done)
	}()

	// // 监听系统信号
	// quit := make(chan os.Signal, 1)
	// signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// <-quit
	// if h.cfg.Log && h.cfg.LogName != "--" {
	// 	log.GetLogger(h.cfg.LogName).Info("Shutting down server...")
	// }
	// // 创建超时上下文
	// ctx, cancel := context.WithTimeout(context.Background(), h.cfg.ShutdownCloseMaxWait)
	// defer cancel()
	// // 关闭服务器
	// if err := h.srv.Shutdown(ctx); err != nil {
	// 	if h.cfg.Log && h.cfg.LogName != "--" {
	// 		log.GetLogger(h.cfg.LogName).Errorf("Server forced to shutdown: %v", err)
	// 	}
	// 	return err
	// }
	// // 等待所有请求完成
	// select {
	// case <-h.done:
	// 	if h.cfg.Log && h.cfg.LogName != "--" {
	// 		log.GetLogger(h.cfg.LogName).Info("Server gracefully stopped")
	// 	}
	// case <-ctx.Done():
	// 	if h.cfg.Log && h.cfg.LogName != "--" {
	// 		log.GetLogger(h.cfg.LogName).Warn("Timeout while waiting for requests to complete")
	// 	}
	// }
	return nil
}

func (h *Http) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), h.cfg.ShutdownCloseMaxWait)
	defer cancel()
	if err := h.srv.Shutdown(ctx); err != nil {
		if h.cfg.Log && h.cfg.LogName != "--" {
			log.GetLogger(h.cfg.LogName).Errorf("server forced to shutdown: %v", err)
		}
		return err
	}
	// 等待所有请求完成
	select {
	case <-h.done:
		if h.cfg.Log && h.cfg.LogName != "--" {
			log.GetLogger(h.cfg.LogName).Info("server gracefully stopped")
		}
	case <-ctx.Done():
		if h.cfg.Log && h.cfg.LogName != "--" {
			log.GetLogger(h.cfg.LogName).Warn("timeout while waiting for requests to complete")
		}
	}
	return nil
}

var https = make(map[string]*Http)

func Init(cfgs map[string]Config, options ...Option) error {
	log.Info("init http server...")
	if len(cfgs) == 0 {
		log.Warn("init http server cfgs is empty")
		return fmt.Errorf("init http server cfgs is empty")
	}
	opt := applyGenGormOptions(options...)
	if len(opt.defKey.Keys) == 0 {
		for key := range cfgs {
			opt.defKey.Keys = append(opt.defKey.Keys, key)
		}
	} else {
		opt.defKey.Keys = append(opt.defKey.Keys, opt.defKey.DefaultKey)
	}

	for _, key := range opt.defKey.Keys {
		_, is := https[key]
		if is {
			continue
		}
		cfg, is := cfgs[key]
		if !is {
			log.Errorf("init http server %s not found", key)
			return fmt.Errorf("init http server %s not found", key)
		}
		cli, err := New(&cfg)
		if err != nil {
			log.Errorf("init http server %s error: %v", key, err)
			return err
		}
		funs, is := opt.engFuncs[key]
		if !is {
			log.Warnf("init http server %s not found engineFunc", key)
		}
		https[key] = cli
		go func() {
			log.Infof("http server port: %d", cli.cfg.Port)
			if err := cli.Run(func(eng *gin.Engine) {
				for i := range funs {
					funs[i](eng)
				}
			}); err != nil {
				log.Errorf("init http server %s run error: %v", key, err)
				return
			}
		}()
	}

	return nil
}

func ShutdownAll() error {
	for key, cli := range https {
		if err := cli.Shutdown(); err != nil {
			log.Warnf("http server %s shutdown error: %v", key, err)
			continue
		}
	}
	return nil
}

func zapLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		logger.Info(path,
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			// zap.Any("keys", c.Keys),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
			zap.Int("size", c.Writer.Size()),
			zap.String("error", c.Errors.ByType(gin.ErrorTypePrivate).String()),
		)
	}
}
