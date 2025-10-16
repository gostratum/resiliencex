package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRetry(t *testing.T) {
	t.Run("creates with default config", func(t *testing.T) {
		config := DefaultRetryConfig()
		retry := NewRetry(config)

		assert.NotNil(t, retry)
		assert.Equal(t, "default", retry.Name())
	})

	t.Run("creates with custom config", func(t *testing.T) {
		config := RetryConfig{
			Name:                "custom-retry",
			MaxAttempts:         5,
			InitialInterval:     50 * time.Millisecond,
			MaxInterval:         5 * time.Second,
			Multiplier:          1.5,
			RandomizationFactor: 0.3,
		}
		retry := NewRetry(config)

		assert.NotNil(t, retry)
		assert.Equal(t, "custom-retry", retry.Name())
	})

	t.Run("fills in zero values with defaults", func(t *testing.T) {
		config := RetryConfig{
			Name: "test",
			// All other fields zero
		}
		retry := NewRetry(config)

		assert.NotNil(t, retry)
		assert.Equal(t, "test", retry.Name())
	})
}

func TestRetryExecution(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		retry := NewRetry(DefaultRetryConfig())
		ctx := context.Background()

		attempts := 0
		err := retry.Execute(ctx, func(ctx context.Context) error {
			attempts++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("retries on failure then succeeds", func(t *testing.T) {
		config := RetryConfig{
			Name:            "test",
			MaxAttempts:     3,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
		}
		retry := NewRetry(config)
		ctx := context.Background()

		attempts := 0
		err := retry.Execute(ctx, func(ctx context.Context) error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary error")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("returns error after max retries", func(t *testing.T) {
		config := RetryConfig{
			Name:            "test",
			MaxAttempts:     3,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
		}
		retry := NewRetry(config)
		ctx := context.Background()

		attempts := 0
		testErr := errors.New("persistent error")
		err := retry.Execute(ctx, func(ctx context.Context) error {
			attempts++
			return testErr
		})

		assert.Error(t, err)
		assert.Equal(t, testErr, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		config := RetryConfig{
			Name:            "test",
			MaxAttempts:     10,
			InitialInterval: 50 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			Multiplier:      2.0,
		}
		retry := NewRetry(config)

		ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
		defer cancel()

		attempts := 0
		err := retry.Execute(ctx, func(ctx context.Context) error {
			attempts++
			return errors.New("error")
		})

		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		assert.Less(t, attempts, 10) // Should not reach max attempts
	})

	t.Run("calls OnRetry callback", func(t *testing.T) {
		retryAttempts := []int{}

		config := RetryConfig{
			Name:            "test",
			MaxAttempts:     3,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
			OnRetry: func(attempt int, err error) {
				retryAttempts = append(retryAttempts, attempt)
			},
		}
		retry := NewRetry(config)
		ctx := context.Background()

		retry.Execute(ctx, func(ctx context.Context) error {
			return errors.New("error")
		})

		// Should have called OnRetry for attempts 1 and 2 (not for attempt 3, which is the last)
		assert.Equal(t, []int{1, 2}, retryAttempts)
	})

	t.Run("respects ShouldRetry predicate", func(t *testing.T) {
		permanentErr := errors.New("permanent error")
		temporaryErr := errors.New("temporary error")

		config := RetryConfig{
			Name:            "test",
			MaxAttempts:     5,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
			ShouldRetry: func(err error) bool {
				return err == temporaryErr
			},
		}
		retry := NewRetry(config)
		ctx := context.Background()

		// Permanent error - should not retry
		attempts := 0
		err := retry.Execute(ctx, func(ctx context.Context) error {
			attempts++
			return permanentErr
		})

		assert.Error(t, err)
		assert.Equal(t, permanentErr, err)
		assert.Equal(t, 1, attempts) // Only one attempt

		// Temporary error - should retry
		attempts = 0
		err = retry.Execute(ctx, func(ctx context.Context) error {
			attempts++
			return temporaryErr
		})

		assert.Error(t, err)
		assert.Equal(t, temporaryErr, err)
		assert.Equal(t, 5, attempts) // All attempts
	})
}

func TestExponentialBackoff(t *testing.T) {
	t.Run("calculates increasing backoff", func(t *testing.T) {
		backoff := &exponentialBackoff{
			initialInterval:     100 * time.Millisecond,
			maxInterval:         10 * time.Second,
			multiplier:          2.0,
			randomizationFactor: 0.0, // No jitter for predictable testing
		}

		delay0 := backoff.Next(0)
		delay1 := backoff.Next(1)
		delay2 := backoff.Next(2)

		// Each delay should be roughly double the previous (with jitter)
		assert.GreaterOrEqual(t, delay0, 50*time.Millisecond)
		assert.LessOrEqual(t, delay0, 150*time.Millisecond)

		assert.GreaterOrEqual(t, delay1, 100*time.Millisecond)
		assert.LessOrEqual(t, delay1, 300*time.Millisecond)

		assert.GreaterOrEqual(t, delay2, 200*time.Millisecond)
		assert.LessOrEqual(t, delay2, 600*time.Millisecond)
	})

	t.Run("caps at max interval", func(t *testing.T) {
		backoff := &exponentialBackoff{
			initialInterval:     100 * time.Millisecond,
			maxInterval:         500 * time.Millisecond,
			multiplier:          2.0,
			randomizationFactor: 0.0,
		}

		// High attempt number should cap at max
		delay := backoff.Next(10)
		assert.LessOrEqual(t, delay, 500*time.Millisecond)
	})
}

func TestDefaultRetryConfig(t *testing.T) {
	t.Run("returns valid defaults", func(t *testing.T) {
		config := DefaultRetryConfig()

		assert.True(t, config.Enabled)
		assert.Equal(t, "default", config.Name)
		assert.Equal(t, 3, config.MaxAttempts)
		assert.Equal(t, 100*time.Millisecond, config.InitialInterval)
		assert.Equal(t, 10*time.Second, config.MaxInterval)
		assert.Equal(t, 2.0, config.Multiplier)
		assert.Equal(t, 0.5, config.RandomizationFactor)
	})
}
