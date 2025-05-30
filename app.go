package tool

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wjoj/tool/v2/config"
	"github.com/wjoj/tool/v2/db/dbx"
	"github.com/wjoj/tool/v2/db/mongox"
	"github.com/wjoj/tool/v2/db/redisx"
	"github.com/wjoj/tool/v2/httpx"
	"github.com/wjoj/tool/v2/log"
	"github.com/wjoj/tool/v2/resources/casbinx"
	"github.com/wjoj/tool/v2/resources/jwt"
	"github.com/wjoj/tool/v2/utils"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type fnNameType string

const (
	fnNameConfig  fnNameType = "config"
	fnNameGorm    fnNameType = "gorm"
	fnNameGenGorm fnNameType = "gengorm"
	fnNameLog     fnNameType = "log"
	fnNameRedis   fnNameType = "redis"
	fnNameMongo   fnNameType = "mongo"
	fnNameHttp    fnNameType = "http"
	fnNameCasbin  fnNameType = "casbin"
	fnNameJwt     fnNameType = "jwt"
)

type funcErr struct {
	Fn        func() error
	RekeaseFn func() error
	Name      fnNameType
}

type cmdarg struct {
	config     *string
	configroot *string
}

type App struct {
	isConfig bool
	fnMap    map[fnNameType]funcErr
	cmdarg   *cmdarg
	opt      Options
	rootCmd  *cobra.Command
	cmds     []*cobra.Command
}

func NewApp(opts ...Option) *App {
	return &App{
		isConfig: false,
		opt:      applyOptions(opts...),
		fnMap:    map[fnNameType]funcErr{},
		cmdarg:   &cmdarg{},
		rootCmd: &cobra.Command{
			Use:     "echo [string to echo]",
			Aliases: []string{"say"},
			Short:   "Echo anything to the screen",
			Long:    "an utterly useless command for testing",
			Example: "Just run cobra-test echo",
		},
	}
}

func (a *App) setIsConfig() {
	a.isConfig = true
}

func (a *App) Config() *App {
	a.setIsConfig()
	a.cmdarg.config = a.rootCmd.PersistentFlags().StringP("config", "c", "config.yaml", "configuration file")
	a.fnMap[fnNameConfig] = funcErr{
		Fn: func() error {
			return config.Read(*a.cmdarg.configroot, *a.cmdarg.config)
		},
		RekeaseFn: nil,
		Name:      fnNameConfig,
	}
	return a
}

func (a *App) Log(options ...log.Option) *App {
	a.setIsConfig()
	a.fnMap[fnNameLog] = funcErr{
		Fn: func() error {
			return log.Load(config.GetLogs(), options...)
		},
		RekeaseFn: func() error {
			log.CloseAll()
			return nil
		},
		Name: fnNameLog,
	}
	return a
}

func (a *App) Redis(options ...redisx.Option) *App {
	a.setIsConfig()
	a.fnMap[fnNameRedis] = funcErr{
		Fn: func() error {
			return redisx.Init(config.GetRediss(), options...)
		},
		RekeaseFn: func() error {
			redisx.CloseAll()
			return nil
		},
		Name: fnNameRedis,
	}
	return a
}

func (a *App) Gorm(options ...dbx.Option) *App {
	a.setIsConfig()
	a.fnMap[fnNameGorm] = funcErr{
		Fn: func() error {
			return dbx.Init(config.GetDbs(), options...)
		},
		RekeaseFn: dbx.CloseAll,
		Name:      fnNameGorm,
	}
	return a
}

func (a *App) GenGorm(options ...dbx.GenOption) *App {
	a.setIsConfig()
	a.fnMap[fnNameGenGorm] = funcErr{
		Fn: func() error {
			dbx.GenByGorm(options...)
			return nil
		},
		RekeaseFn: nil,
		Name:      fnNameGenGorm,
	}
	return a
}

func (a *App) Mongo(options ...mongox.Option) *App {
	a.setIsConfig()
	a.fnMap[fnNameMongo] = funcErr{
		Fn: func() error {
			return mongox.Init(config.GetMongos(), options...)
		},
		RekeaseFn: mongox.CloseAll,
		Name:      fnNameMongo,
	}
	return a
}

func (a *App) Casbin(options ...casbinx.Option) *App {
	a.setIsConfig()
	a.fnMap[fnNameCasbin] = funcErr{
		Fn: func() error {
			return casbinx.Init(config.GetCasbins(), options...)
		},
		RekeaseFn: nil,
		Name:      fnNameCasbin,
	}
	return a
}
func (a *App) Jwt(options ...jwt.Option) *App {
	a.setIsConfig()
	a.fnMap[fnNameJwt] = funcErr{
		Fn: func() error {
			return jwt.Init(config.GetJwts(), options...)
		},
		RekeaseFn: nil,
		Name:      fnNameJwt,
	}
	return a
}
func (a *App) HttpServer(options ...httpx.Option) *App {
	a.setIsConfig()
	a.fnMap[fnNameHttp] = funcErr{
		Fn: func() error {
			return httpx.Init(config.GetHttp(), options...)
		},
		RekeaseFn: httpx.ShutdownAll,
		Name:      fnNameHttp,
	}
	return a
}

func (a *App) With(is bool, fn func(a *App) error) *App {
	return a
}

func (a *App) WithFunc(is bool, fn func() error) *App {
	return a
}

func (a *App) WithRekease(is bool, fn func(a *App) error) *App {
	return a
}

func (a *App) WithRekeaseFunc(is bool, fn func() error) *App {
	return a
}

func (a *App) Flags(f func(flag *pflag.FlagSet)) *App {
	f(a.rootCmd.Flags())
	return a
}

func (a *App) AddCommand(cmd *cobra.Command) *App {
	a.cmds = append(a.cmds, cmd)
	return a
}

func (a *App) AddServer(srvName string) *App {
	return a
}

func (a *App) run(fs []funcErr) error {
	for i := range fs {
		f := fs[i]
		if f.Fn == nil {
			continue
		}

		if err := f.Fn(); err != nil {
			fmt.Printf("%+v err:%+v\n", f.Name, err)
			return err
		}
	}
	return nil
}

func (a *App) rekease(fs []funcErr) error {
	rel := func() error {
		for _, f := range fs {
			if f.RekeaseFn == nil {
				continue
			}
			if err := f.RekeaseFn(); err != nil {
				fmt.Printf("%+v err:%+v\n", f.Name, err)
				return err
			}
		}
		return nil
	}
	if a.opt.quit {
		return rel()
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	return rel()
}

func (a *App) Run() error {
	a.cmdarg.configroot = a.rootCmd.PersistentFlags().StringP("configroot", "r", "etc", "configure the root directory")
	fs := []funcErr{}
	if a.isConfig { //需要配置
		fnConfig, is := a.fnMap[fnNameConfig]
		if !is {
			a.Config()
			fnConfig, is = a.fnMap[fnNameConfig]
			if !is {
				return errors.New("config not found")
			}
		}
		fs = append(fs, fnConfig)
	}
	fnLog, is := a.fnMap[fnNameLog]
	if is {
		fs = append(fs, fnLog)
	} else {
		fs = append(fs, funcErr{
			Fn: func() error {
				return log.Load(map[string]log.Config{
					utils.DefaultKey.DefaultKey: {
						Level:      "debug",
						LevelColor: true,
						Out:        log.OutStdout,
						OutFormat:  log.OutFormatConsole,
					},
				})
			},
		})
	}
	fnames := []fnNameType{
		fnNameRedis, fnNameGorm, fnNameMongo,
		fnNameJwt, fnNameCasbin, fnNameHttp,
		fnNameGenGorm,
	}
	for _, fname := range fnames {
		fn, is := a.fnMap[fname]
		if is {
			fs = append(fs, fn)
		}
	}
	a.rootCmd.Run = func(cmd *cobra.Command, args []string) {
		if err := a.run(fs); err != nil {
			return
		}
		if err := a.rekease(fs); err != nil {
			return
		}
	}
	a.rootCmd.AddCommand(a.cmds...)
	err := a.rootCmd.Execute()
	if err != nil {
		fmt.Println("xxxxx")
		return err
	}

	return nil
}
