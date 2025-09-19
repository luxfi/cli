package binutils

import (
	"log/slog"
	
	"github.com/luxfi/log"
	luxlog "github.com/luxfi/log"
	"go.uber.org/zap"
)

// loggerAdapter adapts luxfi/log.Logger to node's luxlog.Logger interface
type loggerAdapter struct {
	logger log.Logger
}

// NewLoggerAdapter creates a new adapter
func NewLoggerAdapter(logger log.Logger) luxlog.Logger {
	return &loggerAdapter{logger: logger}
}

// Write implements io.Writer
func (l *loggerAdapter) Write(p []byte) (n int, err error) {
	l.logger.Info(string(p))
	return len(p), nil
}

// Fatal implements luxlog.Logger
func (l *loggerAdapter) Fatal(msg string, fields ...zap.Field) {
	// Convert zap.Field to log.Field (they're the same type)
	logFields := make([]log.Field, len(fields))
	for i, f := range fields {
		logFields[i] = log.Field(f)
	}
	l.logger.Fatal(msg, logFields...)
}

// Error implements luxlog.Logger
func (l *loggerAdapter) Error(msg string, fields ...zap.Field) {
	// Convert fields to interface{} for the Error method that expects ...interface{}
	args := fieldsToInterface(fields)
	l.logger.Error(msg, args...)
}

// Warn implements luxlog.Logger
func (l *loggerAdapter) Warn(msg string, fields ...zap.Field) {
	args := fieldsToInterface(fields)
	l.logger.Warn(msg, args...)
}

// Info implements luxlog.Logger
func (l *loggerAdapter) Info(msg string, fields ...zap.Field) {
	args := fieldsToInterface(fields)
	l.logger.Info(msg, args...)
}

// Trace implements luxlog.Logger
func (l *loggerAdapter) Trace(msg string, fields ...zap.Field) {
	args := fieldsToInterface(fields)
	l.logger.Trace(msg, args...)
}

// Debug implements luxlog.Logger
func (l *loggerAdapter) Debug(msg string, fields ...zap.Field) {
	args := fieldsToInterface(fields)
	l.logger.Debug(msg, args...)
}

// Verbo implements luxlog.Logger
func (l *loggerAdapter) Verbo(msg string, fields ...zap.Field) {
	// Convert zap.Field to log.Field (they're the same type)
	logFields := make([]log.Field, len(fields))
	for i, f := range fields {
		logFields[i] = log.Field(f)
	}
	l.logger.Verbo(msg, logFields...)
}

// SetLevel implements luxlog.Logger
func (l *loggerAdapter) SetLevel(level logging.Level) {
	// Convert logging.Level (int8) to slog.Level
	slogLevel := slog.Level(level)
	l.logger.SetLevel(slogLevel)
}

// Enabled implements luxlog.Logger
func (l *loggerAdapter) Enabled(lvl logging.Level) bool {
	// Convert logging.Level (int8) to slog.Level
	slogLevel := slog.Level(lvl)
	return l.logger.EnabledLevel(slogLevel)
}

// StopOnPanic implements luxlog.Logger
func (l *loggerAdapter) StopOnPanic() {
	// Call the logger's StopOnPanic method
	l.logger.StopOnPanic()
}

// RecoverAndPanic implements luxlog.Logger
func (l *loggerAdapter) RecoverAndPanic(f func()) {
	// Call the logger's RecoverAndPanic method
	l.logger.RecoverAndPanic(f)
}

// RecoverAndExit implements luxlog.Logger
func (l *loggerAdapter) RecoverAndExit(f func(), exit func()) {
	// Call the logger's RecoverAndExit method
	l.logger.RecoverAndExit(f, exit)
}

// Stop implements luxlog.Logger
func (l *loggerAdapter) Stop() {
	// Implementation depends on how luxfi/log handles cleanup
	l.logger.Stop()
}

// fieldsToInterface converts zap.Field slice to interface{} slice
func fieldsToInterface(fields []zap.Field) []interface{} {
	result := make([]interface{}, 0, len(fields)*2)
	for _, f := range fields {
		result = append(result, f.Key, f.Interface)
	}
	return result
}