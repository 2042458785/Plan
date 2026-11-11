package config

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
)

func InitLogger(options ...LoggerConfig) *zap.SugaredLogger {
	if len(options) == 0 {
		cfg := zap.NewProductionConfig()
		cfg.DisableStacktrace = true
		logger, e := cfg.Build()
		if e != nil {
			panic(e)
		}
		return logger.Sugar()
	}
	cfg := options[0]
	if cfg.MaxSize < 1 {
		cfg.MaxSize = 20
	}
	if e := os.MkdirAll("log", 0700); e != nil {
		panic(e)
	}
	rotation := &lumberjack.Logger{Filename: "log/app.log", MaxSize: cfg.MaxSize, MaxBackups: cfg.MaxBackups, MaxAge: cfg.MaxAge, LocalTime: true}
	encoder := zap.NewProductionEncoderConfig()
	encoder.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoder), zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stderr), zapcore.AddSync(rotation)), zap.InfoLevel)
	return zap.New(core, zap.AddCaller()).Sugar()
}
