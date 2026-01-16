package logging

import (
	"context"
	"fmt"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// Log logs to both console (GitHub Actions format) and New Relic
// Extracts the New Relic transaction from context if available
func Log(ctx context.Context, level, message string) {
	// Always log to console for GitHub Actions
	fmt.Printf("::%s::%s\n", level, message)

	// Also send to New Relic if transaction exists in context
	if txn := newrelic.FromContext(ctx); txn != nil {
		txn.RecordLog(newrelic.LogData{
			Message:  message,
			Severity: level,
		})
	}
}

// Logf is like Log but supports formatting
func Logf(ctx context.Context, level, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	Log(ctx, level, message)
}

// Convenience functions for common log levels
func Notice(ctx context.Context, message string) {
	Log(ctx, "notice", message)
}

func Noticef(ctx context.Context, format string, args ...interface{}) {
	Logf(ctx, "notice", format, args...)
}

func Debug(ctx context.Context, message string) {
	Log(ctx, "debug", message)
}

func Debugf(ctx context.Context, format string, args ...interface{}) {
	Logf(ctx, "debug", format, args...)
}

func Error(ctx context.Context, message string) {
	Log(ctx, "error", message)
}

func Errorf(ctx context.Context, format string, args ...interface{}) {
	Logf(ctx, "error", format, args...)
}

func Warn(ctx context.Context, message string) {
	Log(ctx, "warn", message)
}

func Warnf(ctx context.Context, format string, args ...interface{}) {
	Logf(ctx, "warn", format, args...)
}
