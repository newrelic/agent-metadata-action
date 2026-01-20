package logging

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func TestLog_WithoutNewRelic(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	Log(ctx, "notice", "Test message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expected := "::notice::Test message\n"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestLog_WithNewRelic(t *testing.T) {
	// Create a test New Relic app (with invalid config so it doesn't actually connect)
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("test-app"),
		newrelic.ConfigLicense("0000000000000000000000000000000000000000"),
		newrelic.ConfigEnabled(false), // Disable to avoid connection attempts
	)
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	// Create a transaction
	txn := app.StartTransaction("test-transaction")
	defer txn.End()

	// Create context with transaction
	ctx := newrelic.NewContext(context.Background(), txn)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Log(ctx, "notice", "Test message")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should contain trace ID if distributed tracing is enabled
	// Format: ::notice::[trace=<id>] Test message
	if !strings.Contains(output, "::notice::") {
		t.Errorf("Output should contain notice level: %s", output)
	}
	if !strings.Contains(output, "Test message") {
		t.Errorf("Output should contain test message: %s", output)
	}

	// Note: Trace ID might be empty if distributed tracing is not enabled in test config
	t.Logf("Output: %s", output)
}

func TestGetTraceID_NoTransaction(t *testing.T) {
	ctx := context.Background()
	traceID := getTraceID(ctx)
	if traceID != "" {
		t.Errorf("Expected empty trace ID without transaction, got %q", traceID)
	}
}

func TestGetTraceID_WithTransaction(t *testing.T) {
	// Create a test New Relic app with distributed tracing enabled
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("test-app"),
		newrelic.ConfigLicense("0000000000000000000000000000000000000000"),
		newrelic.ConfigEnabled(false),
		newrelic.ConfigDistributedTracerEnabled(true),
	)
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	txn := app.StartTransaction("test-transaction")
	defer txn.End()

	ctx := newrelic.NewContext(context.Background(), txn)
	traceID := getTraceID(ctx)

	// Trace ID might be empty in test environment, but function should not panic
	t.Logf("Trace ID: %q", traceID)
}
