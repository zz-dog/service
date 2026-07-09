package global

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitZap() {
	// 开发环境控制台输出配置
	encoderCfg := zap.NewDevelopmentEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)

	// 输出到控制台
	consoleOutput := zapcore.AddSync(zapcore.Lock(os.Stdout))

	// 日志级别 Info
	level := zapcore.InfoLevel

	core := zapcore.NewCore(consoleEncoder, consoleOutput, level)
	Logger = zap.New(core, zap.AddCaller())
}
