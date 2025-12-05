// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localnet

import (
	"context"
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
func (l *loggerAdapter) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
}

// Warn implements luxlog.Logger
func (l *loggerAdapter) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}

// Info implements luxlog.Logger
func (l *loggerAdapter) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

// Trace implements luxlog.Logger
func (l *loggerAdapter) Trace(msg string, args ...interface{}) {
	l.logger.Trace(msg, args...)
}

// Debug implements luxlog.Logger
func (l *loggerAdapter) Debug(msg string, args ...interface{}) {
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

// Crit implements luxlog.Logger
func (l *loggerAdapter) Crit(msg string, args ...interface{}) {
	l.logger.Crit(msg, args...)
}

// SetLevel implements luxlog.Logger
func (l *loggerAdapter) SetLevel(level slog.Level) {
	l.logger.SetLevel(level)
}

// Enabled implements luxlog.Logger
func (l *loggerAdapter) Enabled(ctx context.Context, lvl slog.Level) bool {
	return l.logger.Enabled(ctx, lvl)
}

// EnabledLevel implements luxlog.Logger (node compatibility)
func (l *loggerAdapter) EnabledLevel(lvl slog.Level) bool {
	return l.logger.EnabledLevel(lvl)
}

// GetLevel implements luxlog.Logger
func (l *loggerAdapter) GetLevel() slog.Level {
	return l.logger.GetLevel()
}

// With implements luxlog.Logger
func (l *loggerAdapter) With(ctx ...interface{}) luxlog.Logger {
	return &loggerAdapter{logger: l.logger.With(ctx...)}
}

// New implements luxlog.Logger
func (l *loggerAdapter) New(ctx ...interface{}) luxlog.Logger {
	return &loggerAdapter{logger: l.logger.New(ctx...)}
}

// Log implements luxlog.Logger
func (l *loggerAdapter) Log(level slog.Level, msg string, ctx ...interface{}) {
	l.logger.Log(level, msg, ctx...)
}

// WriteLog implements luxlog.Logger
func (l *loggerAdapter) WriteLog(level slog.Level, msg string, attrs ...any) {
	l.logger.WriteLog(level, msg, attrs...)
}

// Handler implements luxlog.Logger
func (l *loggerAdapter) Handler() slog.Handler {
	return l.logger.Handler()
}

// WithFields implements luxlog.Logger
func (l *loggerAdapter) WithFields(fields ...log.Field) luxlog.Logger {
	return &loggerAdapter{logger: l.logger.WithFields(fields...)}
}

// WithOptions implements luxlog.Logger
func (l *loggerAdapter) WithOptions(opts ...log.Option) luxlog.Logger {
	return &loggerAdapter{logger: l.logger.WithOptions(opts...)}
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
