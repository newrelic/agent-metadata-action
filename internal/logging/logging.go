package logging

import (
	"context"
	"fmt"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// Log logs to both console (GitHub Actions format) and New Relic
// Extracts the New Relic transaction from context if available
func Log(ctx context.Context, level, message string) {
	// Get trace ID from New Relic transaction for correlation
	traceID := getTraceID(ctx)

	// Format message with trace ID if available
	formattedMessage := message
	if traceID != "" {
		formattedMessage = fmt.Sprintf("[trace=%s] %s", traceID, message)
	}

	// Always log to console for GitHub Actions
	fmt.Printf("::%s::%s\n", level, formattedMessage)

	// Also send to New Relic if transaction exists in context
	if txn := newrelic.FromContext(ctx); txn != nil {
		txn.RecordLog(newrelic.LogData{
			Message:  message,
			Severity: level,
		})
	}
}

// getTraceID extracts the trace ID from the New Relic transaction in the context
func getTraceID(ctx context.Context) string {
	if txn := newrelic.FromContext(ctx); txn != nil {
		metadata := txn.GetTraceMetadata()
		return metadata.TraceID
	}
	return ""
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

// NoticeError records an error in New Relic with contextual attributes
// This should be called in addition to logging.Error/Errorf, not instead of it
func NoticeError(ctx context.Context, err error, attributes map[string]interface{}) {
	if err == nil {
		return
	}

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return // No transaction available, skip error noticing
	}

	// Create New Relic error with attributes
	nrErr := newrelic.Error{
		Message:    err.Error(),
		Class:      "ApplicationError",
		Attributes: attributes,
	}

	txn.NoticeError(nrErr)
}

// NoticeErrorWithCategory is a convenience wrapper that adds a category attribute
func NoticeErrorWithCategory(ctx context.Context, err error, category string, additionalAttrs map[string]interface{}) {
	if err == nil {
		return
	}

	attrs := map[string]interface{}{
		"error.category": category,
	}

	// Merge additional attributes
	for k, v := range additionalAttrs {
		attrs[k] = v
	}

	NoticeError(ctx, err, attrs)
}
