package log

import (
	"io"
	"os"
	"sync"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Level      string
	Path       string
	MaxSize    int  // 在进行切割之前，日志文件的最大大小（以MB为单位）
	MaxBackups int  // 保留旧文件的最大个数
	MaxAge     int  // 保留旧文件的最大天数
	Compress   bool // 是否压缩/归档旧文件
}

func New(cfg *Config) *zap.Logger {
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
	} else {
		file = os.Stdout
	}
	enccfg := zap.NewProductionEncoderConfig()
	enccfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05") //zapcore.ISO8601TimeEncoder
	enccfg.EncodeLevel = zapcore.CapitalLevelEncoder
	enc := zapcore.NewJSONEncoder(enccfg)
	core := zapcore.NewCore(enc,
		zapcore.AddSync(file), zapcore.DebugLevel)
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
