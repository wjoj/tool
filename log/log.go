package log

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/natefinch/lumberjack"
	"github.com/wjoj/tool/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LevelType string

func (t LevelType) ZapLevel() zapcore.Level {
	t = LevelType(strings.ToLower(string(t)))
	if t == "debug" {
		return zapcore.DebugLevel
	} else if t == "info" {
		return zapcore.InfoLevel
	} else if t == "warn" {
		return zapcore.WarnLevel
	} else if t == "error" {
		return zapcore.ErrorLevel
	} else if t == "dpanic" {
		return zapcore.DPanicLevel
	} else if t == "panic" {
		return zapcore.PanicLevel
	} else if t == "fatal" {
		return zapcore.FatalLevel
	}
	return zapcore.DebugLevel
}

// OutType 输出类型
type OutType string

const (
	OutFile       OutType = "file"
	OutStdout     OutType = "stdout"
	OutFileStdout OutType = "fileStdout" // 系统日志
)

// OutFormatType 输出格式
type OutFormatType string

const (
	OutFormatJson    OutFormatType = "json" //json
	OutFormatConsole OutFormatType = "console"
)

type Config struct {
	Level      LevelType     `json:"level" yaml:"level"`           //等级
	LevelColor bool          `json:"levelColor" yaml:"levelColor"` //是否开启等级颜色
	Out        OutType       `json:"out" yaml:"out"`               //输出类型
	OutFormat  OutFormatType `json:"outFormat" yaml:"outFormat"`   //输出格式
	Path       string        `json:"path" yaml:"path"`             //日志路径
	MaxSize    int           `json:"maxSize" yaml:"maxSize"`       // 在进行切割之前，日志文件的最大大小（以MB为单位）
	MaxBackups int           `json:"maxBackups" yaml:"maxBackups"` // 保留旧文件的最大个数
	MaxAge     int           `json:"maxAge" yaml:"maxAge"`         // 保留旧文件的最大天数
	Compress   bool          `json:"compress" yaml:"compress"`     // 是否压缩/归档旧文件
}

func New(cfg *Config) (*zap.Logger, error) {
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 7
	}
	if cfg.MaxBackups == 0 {
		cfg.MaxBackups = 10
	}
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 512
	}
	if len(cfg.Level) == 0 {
		cfg.Level = LevelType("debug")
	}
	if len(cfg.Out) == 0 {
		cfg.Out = OutStdout
	}
	if len(cfg.OutFormat) == 0 {
		cfg.OutFormat = OutFormatConsole
	}
	if (cfg.Out == OutFile || cfg.Out == OutFileStdout) && len(cfg.Path) == 0 {
		return nil, errors.New("path is empty")
	}
	enccfg := zap.NewProductionEncoderConfig()
	enccfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05") //zapcore.ISO8601TimeEncoder
	if cfg.LevelColor &&
		cfg.OutFormat == OutFormatConsole &&
		cfg.Out == OutStdout {
		enccfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		enccfg.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	var enc zapcore.Encoder
	var file io.Writer

	if cfg.Out == OutFile || cfg.Out == OutFileStdout {
		lumberJackLogger := &lumberjack.Logger{
			Filename:   cfg.Path,
			MaxSize:    cfg.MaxSize,    //在进行切割之前，日志文件的最大大小（以MB为单位）
			MaxBackups: cfg.MaxBackups, //保留旧文件的最大个数
			MaxAge:     cfg.MaxAge,     //保留旧文件的最大天数
			Compress:   cfg.Compress,   //是否压缩/归档旧文件
		}
		file = lumberJackLogger
	} else {
		file = os.Stdout
	}
	if cfg.OutFormat == OutFormatConsole {
		enc = zapcore.NewConsoleEncoder(enccfg)
	} else {
		enc = zapcore.NewJSONEncoder(enccfg)
	}
	core := zapcore.NewCore(enc,
		zapcore.AddSync(file), cfg.Level.ZapLevel())
	ler := zap.New(core, zap.AddCaller(), zap.Development(), zap.AddCallerSkip(1))
	logsugared = ler.Sugar()
	return ler, nil
}

var logsugared *zap.SugaredLogger
var logg *zap.Logger
var logsugaredMap map[string]*zap.SugaredLogger
var defaultKey = utils.DefaultKey.DefaultKey

func Load(logs map[string]Config, options ...Option) error {
	opt := applyGenGormOptions(options...)
	defaultKey = opt.defKey.DefaultKey
	logsugaredMap = make(map[string]*zap.SugaredLogger)
	if len(opt.defKey.Keys) != 0 {
		opt.defKey.Keys = append(opt.defKey.Keys, opt.defKey.DefaultKey)
		for _, key := range opt.defKey.Keys {
			_, is := logsugaredMap[key]
			if is {
				continue
			}
			cfg, is := logs[key]
			if !is {
				return errors.New(key + " log key not found")
			}
			zaplog, err := New(&cfg)
			if err != nil {
				return err
			}
			logsugaredMap[key] = zaplog.Sugar()
			if key == opt.defKey.DefaultKey {
				logg = zaplog
				logsugared = zaplog.Sugar()
			}
		}
		return nil
	}
	for name := range logs {
		cfg := logs[name]
		zaplog, err := New(&cfg)
		if err != nil {
			return err
		}
		logsugaredMap[name] = zaplog.Sugar()
		if name == opt.defKey.DefaultKey {
			logg = zaplog
			logsugared = zaplog.Sugar()
		}
	}
	return nil
}

func GetLogger(key ...string) *zap.SugaredLogger {
	k := defaultKey
	if len(key) != 0 {
		k = key[0]
	}
	log, is := logsugaredMap[k]
	if is {
		return log
	}
	panic(k + "log key not found")
}

func NewGlobal(cfg Config) error {
	log, err := New(&cfg)
	if err != nil {
		return err
	}
	logsugared = log.Sugar()
	return nil
}

func FieldDebugf(msg string, a ...zap.Field) {
	logg.Info(msg, a...)
}
func FieldDebug(msg string, a ...zap.Field) {
	logg.Debug(msg, a...)
}
func FieldInfo(msg string, a ...zap.Field) {
	logg.Info(msg, a...)
}

func FieldWarn(msg string, a ...zap.Field) {
	logg.Warn(msg, a...)
}

func FieldError(msg string, a ...zap.Field) {
	logg.Error(msg, a...)
}

func FieldPanic(msg string, a ...zap.Field) {
	logg.Panic(msg, a...)
}
func FieldFatal(msg string, a ...zap.Field) {
	logg.Fatal(msg, a...)
}
func FieldDPanic(msg string, a ...zap.Field) {
	logg.DPanic(msg, a...)
}

func Debug(a ...any) {
	logsugared.Debug(a...)
}

func Debugf(format string, a ...any) {
	logsugared.Debugf(format, a...)
}

func Debugw(msg string, keysAndValues ...any) {
	logsugared.Debugw(msg, keysAndValues...)
}

func Info(a ...any) {
	logsugared.Info(a...)
}

func Infof(format string, a ...any) {
	logsugared.Infof(format, a...)
}
func Infow(msg string, keysAndValues ...any) {
	logsugared.Infow(msg, keysAndValues...)
}

func Warn(a ...any) {
	logsugared.Warn(a...)
}
func Warnf(format string, a ...any) {
	logsugared.Warnf(format, a...)
}
func Warnw(msg string, keysAndValues ...any) {
	logsugared.Warnw(msg, keysAndValues...)
}

func Error(a ...any) {
	logsugared.Error(a...)
}
func Errorf(format string, keysAndValues ...any) {
	logsugared.Errorf(format, keysAndValues...)
}
func Errorw(msg string, keysAndValues ...any) {
	logsugared.Errorw(msg, keysAndValues...)
}

func DPanic(a ...any) {
	logsugared.DPanic(a...)
}
func Panicf(format string, a ...any) {
	logsugared.Panicf(format, a...)
}
func Panicw(msg string, keysAndValues ...any) {
	logsugared.Panicw(msg, keysAndValues...)
}

func Close() {
	logsugared.Desugar().Sync()
}

func CloseAll() {
	for _, log := range logsugaredMap {
		log.Desugar().Sync()
	}
}

func LoggerSugared() *zap.SugaredLogger {
	return logsugared
}
