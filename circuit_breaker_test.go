package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreakerStates(t *testing.T) {
	t.Run("state string representations", func(t *testing.T) {
		assert.Equal(t, "closed", StateClosed.String())
		assert.Equal(t, "open", StateOpen.String())
		assert.Equal(t, "half-open", StateHalfOpen.String())
		assert.Equal(t, "unknown", CircuitState(999).String())
	})
}

func TestNewCircuitBreaker(t *testing.T) {
	t.Run("creates with default config", func(t *testing.T) {
		config := DefaultCircuitBreakerConfig()
		cb := NewCircuitBreaker(config)

		assert.NotNil(t, cb)
		assert.Equal(t, "default", cb.Name())
		assert.Equal(t, StateClosed, cb.State())
	})

	t.Run("creates with custom config", func(t *testing.T) {
		config := CircuitBreakerConfig{
			Name:             "custom-breaker",
			MaxRequests:      10,
			Interval:         30 * time.Second,
			Timeout:          15 * time.Second,
			FailureThreshold: 0.5,
			MinRequests:      5,
		}
		cb := NewCircuitBreaker(config)

		assert.NotNil(t, cb)
		assert.Equal(t, "custom-breaker", cb.Name())
		assert.Equal(t, StateClosed, cb.State())
	})

	t.Run("fills in zero values with defaults", func(t *testing.T) {
		config := CircuitBreakerConfig{
			Name: "test",
			// All other fields zero - should use defaults
		}
		cb := NewCircuitBreaker(config)

		assert.NotNil(t, cb)
		assert.Equal(t, "test", cb.Name())
	})
}

func TestCircuitBreakerExecution(t *testing.T) {
	t.Run("executes successfully in closed state", func(t *testing.T) {
		cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
		ctx := context.Background()

		called := false
		err := cb.Execute(ctx, func(ctx context.Context) error {
			called = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, StateClosed, cb.State())
	})

	t.Run("records failures", func(t *testing.T) {
		cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
		ctx := context.Background()

		testErr := errors.New("test error")
		err := cb.Execute(ctx, func(ctx context.Context) error {
			return testErr
		})

		assert.Error(t, err)
		assert.Equal(t, testErr, err)
		assert.Equal(t, StateClosed, cb.State()) // Still closed, need more failures
	})

	t.Run("opens circuit after threshold failures", func(t *testing.T) {
		config := CircuitBreakerConfig{
			Name:             "test",
			MaxRequests:      2,
			Interval:         1 * time.Minute,
			Timeout:          100 * time.Millisecond,
			FailureThreshold: 0.5, // 50%
			MinRequests:      3,   // Need at least 3 requests
		}
		cb := NewCircuitBreaker(config)
		ctx := context.Background()

		// Execute 5 requests - 3 failures, 2 successes (60% failure rate)
		cb.Execute(ctx, func(ctx context.Context) error { return errors.New("failure") })
		cb.Execute(ctx, func(ctx context.Context) error { return errors.New("failure") })
		cb.Execute(ctx, func(ctx context.Context) error { return errors.New("failure") })
		cb.Execute(ctx, func(ctx context.Context) error { return nil })
		cb.Execute(ctx, func(ctx context.Context) error { return nil })

		// Circuit should now be open (3 failures out of 5 = 60% failure rate)
		assert.Equal(t, StateOpen, cb.State())

		// Subsequent request should fail immediately
		called := false
		err := cb.Execute(ctx, func(ctx context.Context) error {
			called = true
			return nil
		})

		assert.Error(t, err)
		assert.Equal(t, ErrCircuitOpen, err)
		assert.False(t, called) // Function should not be called
	})

	t.Run("transitions to half-open after timeout", func(t *testing.T) {
		config := CircuitBreakerConfig{
			Name:             "test",
			MaxRequests:      2,
			Interval:         1 * time.Minute,
			Timeout:          50 * time.Millisecond, // Short timeout
			FailureThreshold: 0.5,
			MinRequests:      2,
		}
		cb := NewCircuitBreaker(config)
		ctx := context.Background()

		// Trigger circuit open
		cb.Execute(ctx, func(ctx context.Context) error { return errors.New("error") })
		cb.Execute(ctx, func(ctx context.Context) error { return errors.New("error") })

		assert.Equal(t, StateOpen, cb.State())

		// Wait for timeout
		time.Sleep(60 * time.Millisecond)

		// Next request should transition to half-open
		cb.Execute(ctx, func(ctx context.Context) error { return nil })

		// Should now be in half-open or closed (depending on result)
		state := cb.State()
		assert.True(t, state == StateHalfOpen || state == StateClosed)
	})

	t.Run("manual reset closes circuit", func(t *testing.T) {
		config := CircuitBreakerConfig{
			Name:             "test",
			MaxRequests:      2,
			Interval:         1 * time.Minute,
			Timeout:          1 * time.Second,
			FailureThreshold: 0.5,
			MinRequests:      2,
		}
		cb := NewCircuitBreaker(config)
		ctx := context.Background()

		// Trigger circuit open
		cb.Execute(ctx, func(ctx context.Context) error { return errors.New("error") })
		cb.Execute(ctx, func(ctx context.Context) error { return errors.New("error") })

		assert.Equal(t, StateOpen, cb.State())

		// Manual reset
		cb.Reset()

		assert.Equal(t, StateClosed, cb.State())

		// Should now accept requests
		called := false
		err := cb.Execute(ctx, func(ctx context.Context) error {
			called = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, called)
	})
}

func TestCircuitBreakerStateCallback(t *testing.T) {
	t.Run("calls state change callback", func(t *testing.T) {
		stateChanges := []CircuitState{}

		config := CircuitBreakerConfig{
			Name:             "test",
			MaxRequests:      2,
			Interval:         1 * time.Minute,
			Timeout:          50 * time.Millisecond,
			FailureThreshold: 0.5,
			MinRequests:      2,
			OnStateChange: func(name string, from, to CircuitState) {
				stateChanges = append(stateChanges, to)
			},
		}
		cb := NewCircuitBreaker(config)
		ctx := context.Background()

		// Trigger state changes
		cb.Execute(ctx, func(ctx context.Context) error { return errors.New("error") })
		cb.Execute(ctx, func(ctx context.Context) error { return errors.New("error") })

		// Should have transitioned to open
		assert.Contains(t, stateChanges, StateOpen)
	})
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	t.Run("returns valid defaults", func(t *testing.T) {
		config := DefaultCircuitBreakerConfig()

		assert.True(t, config.Enabled)
		assert.Equal(t, "default", config.Name)
		assert.Equal(t, uint32(5), config.MaxRequests)
		assert.Equal(t, 60*time.Second, config.Interval)
		assert.Equal(t, 30*time.Second, config.Timeout)
		assert.Equal(t, 0.6, config.FailureThreshold)
		assert.Equal(t, uint32(10), config.MinRequests)
	})
}
