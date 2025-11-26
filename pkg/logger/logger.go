package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"prometheus/pkg/errors"
)

var globalLogger *Logger

// Logger wraps zap.SugaredLogger with optional error tracking
type Logger struct {
	*zap.SugaredLogger
	errorTracker errors.Tracker
}

// Init initializes the global logger
func Init(level string, env string) error {
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Parse level
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	logger, err := config.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
	}

	globalLogger = &Logger{
		SugaredLogger: logger.Sugar(),
		errorTracker:  nil, // Will be set via SetErrorTracker
	}
	return nil
}

// SetErrorTracker sets the error tracker for automatic error reporting
func SetErrorTracker(tracker errors.Tracker) {
	if globalLogger != nil {
		globalLogger.errorTracker = tracker
	}
}

// Get returns the global logger
func Get() *Logger {
	if globalLogger == nil {
		// Fallback to basic logger
		logger, _ := zap.NewDevelopment()
		globalLogger = &Logger{
			SugaredLogger: logger.Sugar(),
			errorTracker:  nil,
		}
	}
	return globalLogger
}

// With creates a child logger with additional fields
func (l *Logger) With(args ...interface{}) *Logger {
	return &Logger{
		SugaredLogger: l.SugaredLogger.With(args...),
		errorTracker:  l.errorTracker,
	}
}

// WithFields creates a child logger with a map of fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{
		SugaredLogger: l.SugaredLogger.With(args...),
		errorTracker:  l.errorTracker,
	}
}

// Error logs an error and optionally sends it to error tracker
func (l *Logger) Error(args ...interface{}) {
	l.SugaredLogger.Error(args...)

	// Auto-track errors if tracker is set
	if l.errorTracker != nil {
		err := errors.Wrapf(errors.ErrInternal, "%v", args)
		l.errorTracker.CaptureError(context.Background(), err, map[string]string{
			"component": "logger",
		})
	}
}

// Errorf logs a formatted error and optionally sends it to error tracker
func (l *Logger) Errorf(template string, args ...interface{}) {
	l.SugaredLogger.Errorf(template, args...)

	if l.errorTracker != nil {
		err := fmt.Errorf(template, args...)
		l.errorTracker.CaptureError(context.Background(), err, map[string]string{
			"component": "logger",
		})
	}
}

// ErrorWithContext logs an error with context and sends to error tracker
func (l *Logger) ErrorWithContext(ctx context.Context, err error, tags map[string]string) {
	l.SugaredLogger.Error(err)

	if l.errorTracker != nil {
		l.errorTracker.CaptureError(ctx, err, tags)
	}
}

// Convenience functions that use the global logger
func Debug(args ...interface{})                   { Get().Debug(args...) }
func Debugf(template string, args ...interface{}) { Get().Debugf(template, args...) }
func Info(args ...interface{})                    { Get().Info(args...) }
func Infof(template string, args ...interface{})  { Get().Infof(template, args...) }
func Warn(args ...interface{})                    { Get().Warn(args...) }
func Warnf(template string, args ...interface{})  { Get().Warnf(template, args...) }
func Error(args ...interface{})                   { Get().Error(args...) }
func Errorf(template string, args ...interface{}) { Get().Errorf(template, args...) }
func Fatal(args ...interface{})                   { Get().Fatal(args...) }
func Fatalf(template string, args ...interface{}) { Get().Fatalf(template, args...) }

// Sync flushes any buffered log entries
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}
