package retry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"agent-metadata-action/internal/logging"
)

// NonRetryableError wraps an error that should not be retried
type NonRetryableError struct {
	Err error
}

func (e *NonRetryableError) Error() string {
	return e.Err.Error()
}

func (e *NonRetryableError) Unwrap() error {
	return e.Err
}

// NewNonRetryableError creates a non-retryable error
func NewNonRetryableError(err error) error {
	if err == nil {
		return nil
	}
	return &NonRetryableError{Err: err}
}

// IsNonRetryable checks if an error should not be retried
func IsNonRetryable(err error) bool {
	var nonRetryable *NonRetryableError
	return errors.As(err, &nonRetryable)
}

// Config holds retry configuration
type Config struct {
	MaxAttempts int           // Maximum number of attempts (including initial attempt)
	BaseDelay   time.Duration // Base delay between retries (will be multiplied by attempt number)
	Operation   string        // Human-readable operation name for logging
}

// DefaultConfig returns a standard retry configuration
func DefaultConfig() Config {
	return Config{
		MaxAttempts: 3,
		BaseDelay:   2 * time.Second,
		Operation:   "operation",
	}
}

// Do executes a function with retry logic
// The function should return an error to trigger a retry
// Returns nil on success, or the last error if all retries fail
func Do(ctx context.Context, config Config, fn func() error) error {
	if config.MaxAttempts < 1 {
		config.MaxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Add delay before retry (not on first attempt)
		if attempt > 1 {
			delay := time.Duration(attempt-1) * config.BaseDelay
			logging.Debugf(ctx, "Retry attempt %d/%d after %s delay...", attempt, config.MaxAttempts, delay)

			select {
			case <-time.After(delay):
				// Continue with retry
			case <-ctx.Done():
				return fmt.Errorf("retry cancelled: %w", ctx.Err())
			}
		}

		// Execute the function
		lastErr = fn()
		if lastErr == nil {
			// Success!
			if attempt > 1 {
				logging.Debugf(ctx, "Succeeded on attempt %d/%d", attempt, config.MaxAttempts)
			}
			return nil
		}

		// Check if error is non-retryable
		if IsNonRetryable(lastErr) {
			logging.Debugf(ctx, "%s failed with non-retryable error: %v", config.Operation, lastErr)
			return lastErr
		}

		// Log retry info
		if attempt < config.MaxAttempts {
			logging.Warnf(ctx, "%s attempt %d failed: %v - will retry", config.Operation, attempt, lastErr)
		}
	}

	// All retries failed
	return fmt.Errorf("failed %s after %d attempts: %w", config.Operation, config.MaxAttempts, lastErr)
}
