package pkg

import (
	"fmt"
	"strings"
)

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SugaredLogger is the Zap SugaredLogger for use withing the harvester.
var SugaredLogger *zap.SugaredLogger

func getZapLevel(level string) zapcore.Level {
	loweredLevel := strings.ToLower(level)
	if loweredLevel == "info" {
		return zapcore.InfoLevel
	}
	if loweredLevel == "warn" || loweredLevel == "warning" {
		return zapcore.WarnLevel
	}
	if loweredLevel == "error" {
		return zapcore.ErrorLevel
	}
	if loweredLevel == "fatal" {
		return zapcore.FatalLevel
	}
	if loweredLevel == "debug" {
		return zapcore.DebugLevel
	}

	panic(fmt.Sprintf("Invalid logging level %s.", level))
}

// InitLoggingWithParams initialises SugaredLogger with params.
func InitLoggingWithParams(logLevel string, logType string, logFilePath string) {
	outputPaths := make([]string, 0, 2)
	errOutputPaths := make([]string, 0, 2)

	// Build console or file logger based on type.
	for _, logType := range strings.Split(logType, ",") {
		if logType == "console" {
			outputPaths = append(outputPaths, "stdout")
			errOutputPaths = append(errOutputPaths, "stderr")
		} else if logType == "file" {
			outputPaths = append(outputPaths, logFilePath)
			errOutputPaths = append(errOutputPaths, logFilePath)
		}
	}

	zapProduction := zap.Config{
		Level:       zap.NewAtomicLevelAt(getZapLevel(logLevel)),
		Development: false,
		Encoding:    "console",
		EncoderConfig: zapcore.EncoderConfig{
			// Keys can be anything except the empty string.
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "name",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      outputPaths,
		ErrorOutputPaths: errOutputPaths,
	}
	UpdateLogger(zapProduction)
}

// UpdateLogger updates the logger's configuration.
func UpdateLogger(cfg zap.Config) {
	logger, _ := cfg.Build()
	SugaredLogger = logger.Sugar()
}

// LeveledSugaredLogger is an adapter for adapting the SugaredLogger to LeveledLogger interface.
type LeveledSugaredLogger struct {
}

// Error is a wrapper over SugaredLogger's Errorf
func (a *LeveledSugaredLogger) Error(msg string, keysAndValues ...interface{}) {
	SugaredLogger.Errorf(msg, keysAndValues)
}

// Info is a wrapper over SugaredLogger's Infof
func (a *LeveledSugaredLogger) Info(msg string, keysAndValues ...interface{}) {
	SugaredLogger.Infof(msg, keysAndValues)
}

// Debug is a wrapper over SugaredLogger's Debugf
func (a *LeveledSugaredLogger) Debug(msg string, keysAndValues ...interface{}) {
	SugaredLogger.Debugf(msg, keysAndValues)
}

// Warn is a wrapper over SugaredLogger's Warnf
func (a *LeveledSugaredLogger) Warn(msg string, keysAndValues ...interface{}) {
	SugaredLogger.Warnf(msg, keysAndValues)
}

// init initialises logging with default values.
func init() {
	InitLoggingWithParams("info", "console", "")
}
