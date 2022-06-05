package log

import (
	"io"
	"os"
	"strings"
	"sync"

	"github.com/natefinch/lumberjack"
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

type Config struct {
	Level      LevelType
	Path       string
	MaxSize    int  // 在进行切割之前，日志文件的最大大小（以MB为单位）
	MaxBackups int  // 保留旧文件的最大个数
	MaxAge     int  // 保留旧文件的最大天数
	Compress   bool // 是否压缩/归档旧文件
}

func New(cfg *Config) *zap.Logger {
	enccfg := zap.NewProductionEncoderConfig()
	enccfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05") //zapcore.ISO8601TimeEncoder
	enccfg.EncodeLevel = zapcore.CapitalLevelEncoder
	var enc zapcore.Encoder
	var file io.Writer

	if len(cfg.Path) != 0 {
		lumberJackLogger := &lumberjack.Logger{
			Filename:   cfg.Path,
			MaxSize:    cfg.MaxSize,    //在进行切割之前，日志文件的最大大小（以MB为单位）
			MaxBackups: cfg.MaxBackups, //保留旧文件的最大个数
			MaxAge:     cfg.MaxAge,     //保留旧文件的最大天数
			Compress:   cfg.Compress,   //是否压缩/归档旧文件
		}
		file = lumberJackLogger
		enc = zapcore.NewJSONEncoder(enccfg)
	} else {
		file = os.Stdout
		enc = zapcore.NewConsoleEncoder(enccfg)
	}
	// enccfg := zap.NewProductionEncoderConfig()
	// enccfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05") //zapcore.ISO8601TimeEncoder
	// enccfg.EncodeLevel = zapcore.CapitalLevelEncoder
	// zapcore.NewConsoleEncoder(enccfg)
	// enc := zapcore.NewJSONEncoder(enccfg)
	core := zapcore.NewCore(enc,
		zapcore.AddSync(file), cfg.Level.ZapLevel())
	ler := zap.New(core, zap.AddCaller(), zap.Development())
	logsugared = ler.Sugar()
	return ler
}

var logsugared *zap.SugaredLogger
var one sync.Once

func NewGlobal(cfg *Config) {
	one.Do(func() {
		logsugared = New(cfg).Sugar()
	})
}

func Debugf(format string, a ...any) {
	logsugared.Debugf(format, a...)
}

func Infof(format string, a ...any) {
	logsugared.Infof(format, a...)
}

func Warnf(format string, a ...any) {
	logsugared.Warnf(format, a...)
}

func Errorf(format string, a ...any) {
	logsugared.Errorf(format, a...)
}

func Panicf(format string, a ...any) {
	logsugared.Panicf(format, a...)
}

func Close() {
	logsugared.Desugar().Sync()
}

func LoggerSugared() *zap.SugaredLogger {
	return logsugared
}
