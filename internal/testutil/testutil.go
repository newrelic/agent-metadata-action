package testutil

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// CaptureOutput captures stdout and stderr for the duration of the test.
// Returns functions to retrieve the captured output.
//
// Usage:
//
//	func TestMyFunction(t *testing.T) {
//	    getStdout, getStderr := testutil.CaptureOutput(t)
//
//	    // Run code that writes to stdout/stderr
//	    myFunction()
//
//	    // Retrieve captured output (this closes pipes and restores stdout/stderr)
//	    stdout := getStdout()
//	    stderr := getStderr()
//
//	    assert.Contains(t, stdout, "expected output")
//	}
func CaptureOutput(t *testing.T) (getStdout, getStderr func() string) {
	t.Helper()

	// Save original stdout/stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create pipes for capturing output
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	// Redirect stdout/stderr to pipes
	os.Stdout = wOut
	os.Stderr = wErr

	// Channels to store captured output
	outChan := make(chan string, 1)
	errChan := make(chan string, 1)

	// Flags to track if we've already closed/restored
	var outClosed, errClosed bool

	// Read from pipes in goroutines
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rOut)
		outChan <- buf.String()
		rOut.Close()
	}()

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rErr)
		errChan <- buf.String()
		rErr.Close()
	}()

	// Cleanup: ensure everything is cleaned up even if not explicitly called
	t.Cleanup(func() {
		if !outClosed {
			wOut.Close()
			os.Stdout = oldStdout
		}
		if !errClosed {
			wErr.Close()
			os.Stderr = oldStderr
		}
	})

	// Return functions to retrieve captured output
	return func() string {
			if !outClosed {
				// Close write end to signal EOF, restore stdout
				wOut.Close()
				os.Stdout = oldStdout
				outClosed = true
			}
			return <-outChan
		},
		func() string {
			if !errClosed {
				// Close write end to signal EOF, restore stderr
				wErr.Close()
				os.Stderr = oldStderr
				errClosed = true
			}
			return <-errChan
		}
}
