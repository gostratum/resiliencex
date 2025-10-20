package resilience

import (
	"context"
	"time"
)

// builder implements the Builder interface
type builder struct {
	name              string
	circuitBreaker    CircuitBreaker
	retry             Retry
	rateLimiter       RateLimiter
	bulkhead          Bulkhead
	timeout           Timeout
	hasCircuitBreaker bool
	hasRetry          bool
	hasRateLimiter    bool
	hasBulkhead       bool
	hasTimeout        bool
}

// NewBuilder creates a new builder
func NewBuilder() Builder {
	return &builder{
		name: "executor",
	}
}

func (b *builder) WithName(name string) Builder {
	b.name = name
	return b
}

func (b *builder) WithCircuitBreaker(config CircuitBreakerConfig) Builder {
	b.circuitBreaker = NewCircuitBreaker(config)
	b.hasCircuitBreaker = true
	return b
}

func (b *builder) WithRetry(config RetryConfig) Builder {
	b.retry = NewRetry(config)
	b.hasRetry = true
	return b
}

func (b *builder) WithRateLimiter(config RateLimiterConfig) Builder {
	b.rateLimiter = NewRateLimiter(config)
	b.hasRateLimiter = true
	return b
}

func (b *builder) WithBulkhead(config BulkheadConfig) Builder {
	b.bulkhead = NewBulkhead(config)
	b.hasBulkhead = true
	return b
}

func (b *builder) WithTimeout(duration time.Duration) Builder {
	b.timeout = NewTimeout(duration, b.name)
	b.hasTimeout = true
	return b
}

func (b *builder) Build() Executor {
	return &executor{
		name:              b.name,
		circuitBreaker:    b.circuitBreaker,
		retry:             b.retry,
		rateLimiter:       b.rateLimiter,
		bulkhead:          b.bulkhead,
		timeout:           b.timeout,
		hasCircuitBreaker: b.hasCircuitBreaker,
		hasRetry:          b.hasRetry,
		hasRateLimiter:    b.hasRateLimiter,
		hasBulkhead:       b.hasBulkhead,
		hasTimeout:        b.hasTimeout,
	}
}

// executor implements the Executor interface
type executor struct {
	name              string
	circuitBreaker    CircuitBreaker
	retry             Retry
	rateLimiter       RateLimiter
	bulkhead          Bulkhead
	timeout           Timeout
	hasCircuitBreaker bool
	hasRetry          bool
	hasRateLimiter    bool
	hasBulkhead       bool
	hasTimeout        bool
}

func (e *executor) Name() string {
	return e.name
}

func (e *executor) Execute(ctx context.Context, fn func(context.Context) error) error {
	_, err := e.ExecuteWithResult(ctx, func(ctx context.Context) (any, error) {
		return nil, fn(ctx)
	})
	return err
}

func (e *executor) ExecuteWithResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	// Wrap the function with all patterns in order:
	// 1. Rate Limiter (outermost - control admission)
	// 2. Bulkhead (limit concurrency)
	// 3. Timeout (add deadline)
	// 4. Circuit Breaker (protect downstream)
	// 5. Retry (innermost - retry failures)

	wrappedFn := func(ctx context.Context) (any, error) {
		return fn(ctx)
	}

	// Apply retry (innermost)
	if e.hasRetry {
		originalFn := wrappedFn
		wrappedFn = func(ctx context.Context) (any, error) {
			var result any
			err := e.retry.Execute(ctx, func(ctx context.Context) error {
				var execErr error
				result, execErr = originalFn(ctx)
				return execErr
			})
			return result, err
		}
	}

	// Apply circuit breaker
	if e.hasCircuitBreaker {
		originalFn := wrappedFn
		wrappedFn = func(ctx context.Context) (any, error) {
			var result any
			err := e.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
				var execErr error
				result, execErr = originalFn(ctx)
				return execErr
			})
			return result, err
		}
	}

	// Apply timeout
	if e.hasTimeout {
		originalFn := wrappedFn
		wrappedFn = func(ctx context.Context) (any, error) {
			var result any
			err := e.timeout.Execute(ctx, func(ctx context.Context) error {
				var execErr error
				result, execErr = originalFn(ctx)
				return execErr
			})
			return result, err
		}
	}

	// Apply bulkhead
	if e.hasBulkhead {
		originalFn := wrappedFn
		wrappedFn = func(ctx context.Context) (any, error) {
			var result any
			err := e.bulkhead.Execute(ctx, func(ctx context.Context) error {
				var execErr error
				result, execErr = originalFn(ctx)
				return execErr
			})
			return result, err
		}
	}

	// Apply rate limiter (outermost)
	if e.hasRateLimiter {
		if err := e.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	return wrappedFn(ctx)
}
