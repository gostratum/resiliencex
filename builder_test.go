package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()
	assert.NotNil(t, builder)
}

func TestBuilderWithName(t *testing.T) {
	executor := NewBuilder().
		WithName("my-executor").
		Build()

	assert.Equal(t, "my-executor", executor.Name())
}

func TestBuilderWithCircuitBreaker(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	executor := NewBuilder().
		WithCircuitBreaker(config).
		Build()

	assert.NotNil(t, executor)
	assert.Equal(t, "executor", executor.Name())
}

func TestBuilderWithRetry(t *testing.T) {
	config := DefaultRetryConfig()
	executor := NewBuilder().
		WithRetry(config).
		Build()

	assert.NotNil(t, executor)
}

func TestBuilderWithRateLimiter(t *testing.T) {
	config := DefaultRateLimiterConfig()
	executor := NewBuilder().
		WithRateLimiter(config).
		Build()

	assert.NotNil(t, executor)
}

func TestBuilderWithBulkhead(t *testing.T) {
	config := DefaultBulkheadConfig()
	executor := NewBuilder().
		WithBulkhead(config).
		Build()

	assert.NotNil(t, executor)
}

func TestBuilderWithTimeout(t *testing.T) {
	executor := NewBuilder().
		WithTimeout(1 * time.Second).
		Build()

	assert.NotNil(t, executor)
}

func TestBuilderChaining(t *testing.T) {
	executor := NewBuilder().
		WithName("test-executor").
		WithCircuitBreaker(DefaultCircuitBreakerConfig()).
		WithRetry(DefaultRetryConfig()).
		WithRateLimiter(DefaultRateLimiterConfig()).
		WithBulkhead(DefaultBulkheadConfig()).
		WithTimeout(1 * time.Second).
		Build()

	assert.NotNil(t, executor)
	assert.Equal(t, "test-executor", executor.Name())
}

func TestExecutorExecution(t *testing.T) {
	t.Run("executes function successfully", func(t *testing.T) {
		executor := NewBuilder().Build()
		ctx := context.Background()

		called := false
		err := executor.Execute(ctx, func(ctx context.Context) error {
			called = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("returns error from function", func(t *testing.T) {
		executor := NewBuilder().Build()
		ctx := context.Background()

		testErr := errors.New("test error")
		err := executor.Execute(ctx, func(ctx context.Context) error {
			return testErr
		})

		assert.Error(t, err)
		assert.Equal(t, testErr, err)
	})

	t.Run("applies timeout", func(t *testing.T) {
		executor := NewBuilder().
			WithTimeout(50 * time.Millisecond).
			Build()
		ctx := context.Background()

		err := executor.Execute(ctx, func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		assert.Error(t, err)
	})
}

func TestExecutorExecuteWithResult(t *testing.T) {
	t.Run("returns result on success", func(t *testing.T) {
		executor := NewBuilder().Build()
		ctx := context.Background()

		result, err := executor.ExecuteWithResult(ctx, func(ctx context.Context) (any, error) {
			return "success", nil
		})

		assert.NoError(t, err)
		assert.Equal(t, "success", result)
	})

	t.Run("returns error on failure", func(t *testing.T) {
		executor := NewBuilder().Build()
		ctx := context.Background()

		testErr := errors.New("test error")
		result, err := executor.ExecuteWithResult(ctx, func(ctx context.Context) (any, error) {
			return nil, testErr
		})

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
