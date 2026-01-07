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
	return captureOutput(t, false)
}

// CaptureOutputWithDisplay captures stdout and stderr for the duration of the test,
// while also displaying the output in real-time to the test log.
// Returns functions to retrieve the captured output.
//
// Usage:
//
//	func TestMyFunction(t *testing.T) {
//	    getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)
//
//	    // Run code that writes to stdout/stderr (will be visible when running with -v)
//	    myFunction()
//
//	    // Retrieve captured output (this closes pipes and restores stdout/stderr)
//	    stdout := getStdout()
//	    stderr := getStderr()
//
//	    assert.Contains(t, stdout, "expected output")
//	}
func CaptureOutputWithDisplay(t *testing.T) (getStdout, getStderr func() string) {
	return captureOutput(t, true)
}

func captureOutput(t *testing.T, display bool) (getStdout, getStderr func() string) {
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
		if display {
			// Use TeeReader to write to both buffer and original stdout
			io.Copy(&buf, io.TeeReader(rOut, oldStdout))
		} else {
			io.Copy(&buf, rOut)
		}
		outChan <- buf.String()
		rOut.Close()
	}()

	go func() {
		var buf bytes.Buffer
		if display {
			// Use TeeReader to write to both buffer and original stderr
			io.Copy(&buf, io.TeeReader(rErr, oldStderr))
		} else {
			io.Copy(&buf, rErr)
		}
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
