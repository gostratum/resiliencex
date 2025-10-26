package resilience

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is open
	ErrCircuitOpen = errors.New("resilience: circuit breaker is open")

	// ErrMaxRetriesExceeded is returned when max retries are exceeded
	ErrMaxRetriesExceeded = errors.New("resilience: max retries exceeded")

	// ErrRateLimitExceeded is returned when rate limit is exceeded
	ErrRateLimitExceeded = errors.New("resilience: rate limit exceeded")

	// ErrBulkheadFull is returned when bulkhead is at capacity
	ErrBulkheadFull = errors.New("resilience: bulkhead at capacity")

	// ErrTimeout is returned when operation times out
	ErrTimeout = errors.New("resilience: operation timed out")
)

// Executor executes functions with resilience patterns applied
type Executor interface {
	// Execute runs the function with configured resilience patterns
	Execute(ctx context.Context, fn func(context.Context) error) error

	// ExecuteWithResult runs the function and returns a result
	ExecuteWithResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error)

	// Name returns the executor name
	Name() string
}

// CircuitBreaker manages circuit breaker state and executes functions
type CircuitBreaker interface {
	// Execute runs the function if the circuit is closed
	Execute(ctx context.Context, fn func(context.Context) error) error

	// State returns the current circuit state
	State() CircuitState

	// Reset manually resets the circuit to closed state
	Reset()

	// Name returns the circuit breaker name
	Name() string
}

// CircuitState represents the circuit breaker state
type CircuitState int

const (
	// StateClosed means the circuit is closed and requests flow normally
	StateClosed CircuitState = iota

	// StateOpen means the circuit is open and requests are rejected
	StateOpen

	// StateHalfOpen means the circuit is testing if service recovered
	StateHalfOpen
)

// String returns the string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Retry executes functions with retry logic
type Retry interface {
	// Execute runs the function with retry logic
	Execute(ctx context.Context, fn func(context.Context) error) error

	// Name returns the retry name
	Name() string
}

// RateLimiter controls the rate of operations
type RateLimiter interface {
	// Allow returns true if the operation is allowed
	Allow() bool

	// Wait blocks until the operation is allowed or context is done
	Wait(ctx context.Context) error

	// Name returns the rate limiter name
	Name() string
}

// Bulkhead limits concurrent operations
type Bulkhead interface {
	// Execute runs the function if capacity is available
	Execute(ctx context.Context, fn func(context.Context) error) error

	// Available returns the number of available slots
	Available() int

	// Name returns the bulkhead name
	Name() string
}

// Timeout wraps operations with a timeout
type Timeout interface {
	// Execute runs the function with a timeout
	Execute(ctx context.Context, fn func(context.Context) error) error

	// ExecuteWithResult runs the function with a timeout and returns result
	ExecuteWithResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error)

	// Name returns the timeout name
	Name() string
}

// Builder builds an Executor with multiple resilience patterns
type Builder interface {
	// WithCircuitBreaker adds circuit breaker pattern
	WithCircuitBreaker(config CircuitBreakerConfig) Builder

	// WithRetry adds retry pattern
	WithRetry(config RetryConfig) Builder

	// WithRateLimiter adds rate limiter pattern
	WithRateLimiter(config RateLimiterConfig) Builder

	// WithBulkhead adds bulkhead pattern
	WithBulkhead(config BulkheadConfig) Builder

	// WithTimeout adds timeout pattern
	WithTimeout(duration time.Duration) Builder

	// WithName sets the executor name
	WithName(name string) Builder

	// Build creates the executor
	Build() Executor
}

// BackoffStrategy defines how to calculate backoff delays
type BackoffStrategy interface {
	// Next returns the next backoff duration
	Next(attempt int) time.Duration
}

// ShouldRetry determines if an error should trigger a retry
type ShouldRetry func(error) bool

// OnStateChange is called when circuit breaker state changes
type OnStateChange func(name string, from, to CircuitState)

// OnRetry is called before each retry attempt
type OnRetry func(attempt int, err error)

// OnRateLimit is called when rate limit is exceeded
type OnRateLimit func(name string)

// OnBulkheadFull is called when bulkhead is at capacity
type OnBulkheadFull func(name string)
