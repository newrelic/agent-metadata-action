package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDo_Success_FirstAttempt(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		Operation:   "test operation",
	}

	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	err := Do(ctx, config, fn)

	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "Should only be called once on first success")
}

func TestDo_Success_SecondAttempt(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond, // Short delay for fast tests
		Operation:   "test operation",
	}

	callCount := 0
	fn := func() error {
		callCount++
		if callCount < 2 {
			return errors.New("temporary failure")
		}
		return nil
	}

	start := time.Now()
	err := Do(ctx, config, fn)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, 2, callCount, "Should be called twice (fail + success)")
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond, "Should have delayed before retry")
}

func TestDo_Success_ThirdAttempt(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		Operation:   "test operation",
	}

	callCount := 0
	fn := func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}

	start := time.Now()
	err := Do(ctx, config, fn)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, 3, callCount, "Should be called 3 times")
	// Total delay should be: 10ms (attempt 2) + 20ms (attempt 3) = 30ms
	assert.GreaterOrEqual(t, duration, 30*time.Millisecond, "Should have cumulative delay")
}

func TestDo_FailAfterAllRetries(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		Operation:   "test operation",
	}

	callCount := 0
	expectedErr := errors.New("persistent failure")
	fn := func() error {
		callCount++
		return expectedErr
	}

	err := Do(ctx, config, fn)

	require.Error(t, err)
	assert.Equal(t, 3, callCount, "Should attempt all retries")
	assert.Contains(t, err.Error(), "failed test operation after 3 attempts")
	assert.Contains(t, err.Error(), "persistent failure")
}

func TestDo_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := Config{
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		Operation:   "test operation",
	}

	callCount := 0
	fn := func() error {
		callCount++
		if callCount == 2 {
			// Cancel context after second attempt
			cancel()
		}
		return errors.New("failure")
	}

	err := Do(ctx, config, fn)

	require.Error(t, err)
	assert.Equal(t, 2, callCount, "Should stop after context cancellation")
	assert.Contains(t, err.Error(), "retry cancelled")
}

func TestDo_MaxAttemptsLessThanOne(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 0, // Invalid
		BaseDelay:   10 * time.Millisecond,
		Operation:   "test operation",
	}

	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	err := Do(ctx, config, fn)

	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "Should default to 1 attempt")
}

func TestDo_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 4,
		BaseDelay:   50 * time.Millisecond,
		Operation:   "test operation",
	}

	callCount := 0
	delays := []time.Duration{}
	var lastTime time.Time

	fn := func() error {
		callCount++
		now := time.Now()
		if callCount > 1 {
			delays = append(delays, now.Sub(lastTime))
		}
		lastTime = now

		if callCount < 4 {
			return errors.New("failure")
		}
		return nil
	}

	err := Do(ctx, config, fn)

	require.NoError(t, err)
	assert.Equal(t, 4, callCount)
	assert.Len(t, delays, 3, "Should have 3 delays (between 4 attempts)")

	// Verify exponential backoff: delay1 = 50ms, delay2 = 100ms, delay3 = 150ms
	assert.GreaterOrEqual(t, delays[0], 50*time.Millisecond, "First retry delay")
	assert.GreaterOrEqual(t, delays[1], 100*time.Millisecond, "Second retry delay")
	assert.GreaterOrEqual(t, delays[2], 150*time.Millisecond, "Third retry delay")
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 2*time.Second, config.BaseDelay)
	assert.Equal(t, "operation", config.Operation)
}

func TestNonRetryableError(t *testing.T) {
	originalErr := errors.New("validation failed")
	nonRetryableErr := NewNonRetryableError(originalErr)

	assert.Error(t, nonRetryableErr)
	assert.Contains(t, nonRetryableErr.Error(), "validation failed")
	assert.True(t, IsNonRetryable(nonRetryableErr))

	// Check unwrapping
	assert.ErrorIs(t, nonRetryableErr, originalErr)
}

func TestNonRetryableError_NilError(t *testing.T) {
	nonRetryableErr := NewNonRetryableError(nil)
	assert.Nil(t, nonRetryableErr)
}

func TestIsNonRetryable_RegularError(t *testing.T) {
	regularErr := errors.New("network timeout")
	assert.False(t, IsNonRetryable(regularErr))
}

func TestDo_NonRetryableError_FirstAttempt(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		Operation:   "test operation",
	}

	callCount := 0
	nonRetryableErr := NewNonRetryableError(errors.New("bad request"))
	fn := func() error {
		callCount++
		return nonRetryableErr
	}

	err := Do(ctx, config, fn)

	require.Error(t, err)
	assert.Equal(t, 1, callCount, "Should only be called once for non-retryable error")
	assert.True(t, IsNonRetryable(err))
	assert.Contains(t, err.Error(), "bad request")
}

func TestDo_NonRetryableError_SecondAttempt(t *testing.T) {
	ctx := context.Background()
	config := Config{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		Operation:   "test operation",
	}

	callCount := 0
	fn := func() error {
		callCount++
		if callCount == 1 {
			return errors.New("transient error") // Retryable
		}
		return NewNonRetryableError(errors.New("permanent error")) // Non-retryable
	}

	err := Do(ctx, config, fn)

	require.Error(t, err)
	assert.Equal(t, 2, callCount, "Should stop after non-retryable error on second attempt")
	assert.True(t, IsNonRetryable(err))
	assert.Contains(t, err.Error(), "permanent error")
}
