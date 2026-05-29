package logger

import (
	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ log.Logger = (*ZapLogger)(nil)

type ZapLogger struct {
	zapLog *zap.Logger
	level  zap.AtomicLevel
}

func NewZapLogger(level string) *ZapLogger {
	// 设置日志级别
	atom := zap.NewAtomicLevel()
	switch level {
	case "debug":
		atom.SetLevel(zap.DebugLevel)
	case "info":
		atom.SetLevel(zap.InfoLevel)
	case "warn":
		atom.SetLevel(zap.WarnLevel)
	case "error":
		atom.SetLevel(zap.ErrorLevel)
	default:
		atom.SetLevel(zap.InfoLevel)
	}

	// 配置zap
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	config := zap.Config{
		Level:            atom,
		Development:      false,
		Encoding:         "json", // 或 "console"
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	zapLogger, _ := config.Build()
	return &ZapLogger{
		zapLog: zapLogger,
		level:  atom,
	}
}

func (l *ZapLogger) Log(level log.Level, keyvals ...interface{}) error {
	if len(keyvals) == 0 || len(keyvals)%2 != 0 {
		l.zapLog.Warn("keyvals must appear in pairs")
		return nil
	}

	var fields []zap.Field
	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			continue
		}
		fields = append(fields, zap.Any(key, keyvals[i+1]))
	}

	switch level {
	case log.LevelDebug:
		l.zapLog.Debug("", fields...)
	case log.LevelInfo:
		l.zapLog.Info("", fields...)
	case log.LevelWarn:
		l.zapLog.Warn("", fields...)
	case log.LevelError:
		l.zapLog.Error("", fields...)
	case log.LevelFatal:
		l.zapLog.Fatal("", fields...)
	}
	return nil
}

func (l *ZapLogger) Sync() error {
	return l.zapLog.Sync()
}
