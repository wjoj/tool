package dbx

import (
	// ... existing imports ...
	"context"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm/logger"
)

type zapLogger struct {
	*zap.SugaredLogger
}

func (l *zapLogger) LogMode(level logger.LogLevel) logger.Interface {
	if level == logger.Silent {
		l.SugaredLogger = l.WithOptions(zap.IncreaseLevel(zapcore.Level(zapcore.InvalidLevel)))
	} else if level == logger.Info {
		l.SugaredLogger = l.WithOptions(zap.IncreaseLevel(zapcore.Level(zapcore.InfoLevel)))
	} else if level == logger.Warn {
		l.SugaredLogger = l.WithOptions(zap.IncreaseLevel(zapcore.Level(zapcore.WarnLevel)))
	} else if level == logger.Error {
		l.SugaredLogger = l.WithOptions(zap.IncreaseLevel(zapcore.Level(zapcore.ErrorLevel)))
	}
	return l
}

func (l *zapLogger) Info(ctx context.Context, msg string, data ...any) {
	l.SugaredLogger.Infof(msg, data...)
}

func (l *zapLogger) Warn(ctx context.Context, msg string, data ...any) {
	l.SugaredLogger.Warnf(msg, data...)
}

func (l *zapLogger) Error(ctx context.Context, msg string, data ...any) {
	l.SugaredLogger.Errorf(msg, data...)
}

func (l *zapLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, rows := fc()
	l.Desugar().Debug("gorm trace",
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Error(err),
		zap.Duration("duration", time.Since(begin)),
	)
}

func (l *zapLogger) Printf(f string, msg ...interface{}) {
	l.SugaredLogger.Logf(zap.InfoLevel, f, msg...)
}

func WithLogger(logger *zap.SugaredLogger) logger.Interface {
	return &zapLogger{logger}
}
