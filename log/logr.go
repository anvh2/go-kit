package log

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogrConfig struct {
	// Level is debug, info, warn or error. Empty means info.
	Level string `json:"level"`
	// Encoding is json or console. Empty means json.
	Encoding string `json:"encoding"`
	// MessageKey names the field that holds the log message. Empty means "msg".
	MessageKey string `json:"messageKey"`
	// Nil means os.Stderr, which keeps stdout free for programs that use it as a protocol channel.
	Output io.Writer `json:"-"`

	// When set, other keys and non-primitive values are dropped; only their names go to "droppedKeys".
	AllowedKeys   []string `json:"allowedKeys"`
	OmitErrorText bool     `json:"omitErrorText"`
	// When set, a message that does not match becomes "invalid_message", so formatted text cannot carry data.
	MessagePattern string `json:"messagePattern"`
}

// NewLogr writes one line per entry: ts (UTC, milliseconds), level, message, logger name and the key-value pairs.
func NewLogr(cfg LogrConfig) (logr.Logger, error) {
	level := zapcore.InfoLevel
	if cfg.Level != "" {
		parsed, err := zapcore.ParseLevel(cfg.Level)
		if err != nil {
			return logr.Discard(), err
		}
		level = parsed
	}

	messageKey := cfg.MessageKey
	if messageKey == "" {
		messageKey = "msg"
	}
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		MessageKey:     messageKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     encodeUTCMillis,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	var encoder zapcore.Encoder
	switch cfg.Encoding {
	case "", "json":
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case "console":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		return logr.Discard(), fmt.Errorf("log: unknown encoding %q (want json or console)", cfg.Encoding)
	}

	output := cfg.Output
	if output == nil {
		output = os.Stderr
	}
	core := zapcore.NewCore(encoder, zapcore.AddSync(output), level)
	logger := zapr.NewLogger(zap.New(core))
	if len(cfg.AllowedKeys) == 0 && !cfg.OmitErrorText && cfg.MessagePattern == "" {
		return logger, nil
	}
	sink, err := newGuard(logger.GetSink(), cfg)
	if err != nil {
		return logr.Discard(), err
	}
	return logr.New(sink), nil
}

func encodeUTCMillis(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.UTC().Format("2006-01-02T15:04:05.000Z07:00"))
}
