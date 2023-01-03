package log

import (
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func encodeLevelForCloudLogging(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch l {
	case zapcore.DebugLevel:
		enc.AppendString("DEBUG")
	case zapcore.InfoLevel:
		enc.AppendString("INFO")
	case zapcore.WarnLevel:
		enc.AppendString("WARNING")
	case zapcore.ErrorLevel:
		enc.AppendString("ERROR")
	case zapcore.DPanicLevel:
		enc.AppendString("CRITICAL")
	case zapcore.PanicLevel:
		enc.AppendString("ALERT")
	case zapcore.FatalLevel:
		enc.AppendString("EMERGENCY")
	}

}

func NewEncoderConfigOptionForCloudLogging() zap.EncoderConfigOption {
	return func(cfg *zapcore.EncoderConfig) {
		cfg.TimeKey = "time"
		cfg.LevelKey = "severity"
		cfg.MessageKey = "message"
		cfg.EncodeLevel = encodeLevelForCloudLogging
		cfg.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	}
}
