# Resilience Module

The resilience module provides fault-tolerance patterns for building reliable distributed applications. It implements circuit breaker, retry, rate limiter, bulkhead, and timeout patterns that can be composed together.

## Features

- **Circuit Breaker**: Prevent cascading failures by stopping calls to failing services
- **Retry**: Automatically retry failed operations with exponential backoff
- **Rate Limiter**: Control the rate of operations using token bucket algorithm
- **Bulkhead**: Limit concurrent operations to prevent resource exhaustion
- **Timeout**: Add deadlines to operations
- **Composable**: Combine multiple patterns using the Builder
- **Metrics Ready**: Designed for integration with metricsx module
- **Context Aware**: Full context.Context support for cancellation

## Installation

```bash
go get github.com/gostratum/resilience
```

## Quick Start

### Using with Fx

```go
package main

import (
    "context"
    "github.com/gostratum/core"
    "github.com/gostratum/resilience"
    "go.uber.org/fx"
)

func main() {
    fx.New(
        core.Module,
        resilience.Module,
        fx.Invoke(func(builder resilience.Builder) {
            executor := builder.Build()
            
            err := executor.Execute(context.Background(), func(ctx context.Context) error {
                // Your operation here
                return callExternalService(ctx)
            })
        }),
    ).Run()
}
```

### Manual Usage

```go
// Circuit Breaker
cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
    Name:             "api-circuit",
    FailureThreshold: 0.6,  // Trip at 60% failure rate
    MinRequests:      10,   // Minimum requests before checking
    Timeout:          30 * time.Second,
})

err := cb.Execute(ctx, func(ctx context.Context) error {
    return makeAPICall(ctx)
})

// Retry with exponential backoff
retry := resilience.NewRetry(resilience.RetryConfig{
    Name:            "api-retry",
    MaxAttempts:     3,
    InitialInterval: 100 * time.Millisecond,
    Multiplier:      2.0,
})

err = retry.Execute(ctx, func(ctx context.Context) error {
    return makeAPICall(ctx)
})

// Rate Limiter
limiter := resilience.NewRateLimiter(resilience.RateLimiterConfig{
    Name:  "api-limiter",
    Rate:  100.0,  // 100 requests per second
    Burst: 200,    // Allow burst of 200
})

if limiter.Allow() {
    makeAPICall(ctx)
}

// Or wait until allowed
err = limiter.Wait(ctx)
if err == nil {
    makeAPICall(ctx)
}

// Bulkhead - limit concurrency
bulkhead := resilience.NewBulkhead(resilience.BulkheadConfig{
    Name:          "api-bulkhead",
    MaxConcurrent: 10,
    MaxQueueSize:  100,
})

err = bulkhead.Execute(ctx, func(ctx context.Context) error {
    return makeAPICall(ctx)
})
```

## Composing Patterns

Use the Builder to combine multiple patterns:

```go
executor := resilience.NewBuilder().
    WithName("payment-service").
    WithCircuitBreaker(resilience.CircuitBreakerConfig{
        FailureThreshold: 0.5,
        Timeout:          30 * time.Second,
    }).
    WithRetry(resilience.RetryConfig{
        MaxAttempts: 3,
    }).
    WithRateLimiter(resilience.RateLimiterConfig{
        Rate:  50.0,
        Burst: 100,
    }).
    WithBulkhead(resilience.BulkheadConfig{
        MaxConcurrent: 5,
    }).
    WithTimeout(5 * time.Second).
    Build()

// Execute with all patterns applied
result, err := executor.ExecuteWithResult(ctx, func(ctx context.Context) (any, error) {
    return fetchPaymentData(ctx)
})
```

## Pattern Order

When multiple patterns are composed, they are applied in this order (outermost to innermost):

1. **Rate Limiter** - Control admission
2. **Bulkhead** - Limit concurrency
3. **Timeout** - Add deadline
4. **Circuit Breaker** - Protect downstream
5. **Retry** - Retry failures

This order ensures optimal fault tolerance and resource protection.

## Configuration

### Circuit Breaker

```go
type CircuitBreakerConfig struct {
    Enabled          bool          // Enable circuit breaker
    Name             string        // Identifier
    MaxRequests      uint32        // Max requests in half-open state
    Interval         time.Duration // Reset interval for counters
    Timeout          time.Duration // Time before half-open
    FailureThreshold float64       // Failure ratio to trip (0.0-1.0)
    MinRequests      uint32        // Min requests before checking ratio
    OnStateChange    OnStateChange // State change callback
}
```

**States:**
- **Closed**: Normal operation, requests flow through
- **Open**: Circuit tripped, requests fail immediately
- **Half-Open**: Testing if service recovered

### Retry

```go
type RetryConfig struct {
    Enabled             bool          // Enable retry
    Name                string        // Identifier
    MaxAttempts         int           // Maximum retry attempts
    InitialInterval     time.Duration // Initial backoff
    MaxInterval         time.Duration // Maximum backoff
    Multiplier          float64       // Backoff multiplier
    RandomizationFactor float64       // Jitter factor (0.0-1.0)
    ShouldRetry         ShouldRetry   // Error filter
    OnRetry             OnRetry       // Retry callback
}
```

**Backoff Strategies:**
- Exponential with jitter (default)
- Constant
- Linear

### Rate Limiter

```go
type RateLimiterConfig struct {
    Enabled     bool        // Enable rate limiter
    Name        string      // Identifier
    Rate        float64     // Requests per second
    Burst       int         // Maximum burst size
    OnRateLimit OnRateLimit // Rate limit callback
}
```

Uses **token bucket** algorithm for smooth rate limiting with bursts.

### Bulkhead

```go
type BulkheadConfig struct {
    Enabled        bool           // Enable bulkhead
    Name           string         // Identifier
    MaxConcurrent  int            // Max concurrent operations
    MaxQueueSize   int            // Max queue size
    OnBulkheadFull OnBulkheadFull // Full callback
}
```

Uses **semaphore** pattern to limit concurrency and prevent resource exhaustion.

### Timeout

```go
type TimeoutConfig struct {
    Enabled  bool          // Enable timeout
    Duration time.Duration // Timeout duration
}
```

## YAML Configuration

```yaml
resilience:
  circuit_breaker:
    enabled: true
    name: "api-circuit"
    max_requests: 5
    interval: 60s
    timeout: 30s
    failure_threshold: 0.6
    min_requests: 10

  retry:
    enabled: true
    name: "api-retry"
    max_attempts: 3
    initial_interval: 100ms
    max_interval: 10s
    multiplier: 2.0
    randomization_factor: 0.5

  rate_limiter:
    enabled: true
    name: "api-limiter"
    rate: 100.0
    burst: 200

  bulkhead:
    enabled: true
    name: "api-bulkhead"
    max_concurrent: 10
    max_queue_size: 100

  timeout:
    enabled: true
    duration: 30s
```

## Advanced Usage

### Custom Retry Logic

```go
retry := resilience.NewRetry(resilience.RetryConfig{
    MaxAttempts: 5,
    ShouldRetry: func(err error) bool {
        // Only retry on specific errors
        return errors.Is(err, ErrTemporary)
    },
    OnRetry: func(attempt int, err error) {
        log.Printf("Retry attempt %d after error: %v", attempt, err)
    },
})
```

### Circuit Breaker State Monitoring

```go
cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
    OnStateChange: func(name string, from, to resilience.CircuitState) {
        log.Printf("Circuit %s: %s -> %s", name, from, to)
        // Send metrics, alerts, etc.
    },
})

// Check current state
if cb.State() == resilience.StateOpen {
    return errors.New("circuit is open")
}

// Manual reset
cb.Reset()
```

### With Metrics Integration

```go
// Coming soon: Integration with metricsx module
executor := resilience.NewBuilder().
    WithCircuitBreaker(resilience.CircuitBreakerConfig{
        OnStateChange: func(name string, from, to resilience.CircuitState) {
            metrics.Counter("circuit_breaker_state_changes").
                WithLabels("name", name, "from", from.String(), "to", to.String()).
                Inc()
        },
    }).
    WithRetry(resilience.RetryConfig{
        OnRetry: func(attempt int, err error) {
            metrics.Counter("retries_total").
                WithLabels("attempt", strconv.Itoa(attempt)).
                Inc()
        },
    }).
    Build()
```

## Error Handling

The module provides specific errors for each pattern:

```go
err := executor.Execute(ctx, myFunc)

switch {
case errors.Is(err, resilience.ErrCircuitOpen):
    // Circuit breaker is open
case errors.Is(err, resilience.ErrMaxRetriesExceeded):
    // All retries failed
case errors.Is(err, resilience.ErrRateLimitExceeded):
    // Rate limit hit
case errors.Is(err, resilience.ErrBulkheadFull):
    // No capacity available
case errors.Is(err, resilience.ErrTimeout):
    // Operation timed out
}
```

## Best Practices

1. **Choose Appropriate Patterns**: Not every operation needs all patterns
2. **Configure Timeouts**: Always set reasonable timeouts
3. **Monitor Circuit States**: Track state changes for visibility
4. **Tune Thresholds**: Adjust based on your SLAs
5. **Test Failure Scenarios**: Validate behavior under load
6. **Use Callbacks**: Hook into pattern events for observability

## Testing

The module includes a no-op provider for testing:

```go
// In tests, patterns can be disabled via configuration
cfg := resilience.Config{
    CircuitBreaker: resilience.CircuitBreakerConfig{Enabled: false},
    Retry:          resilience.RetryConfig{Enabled: false},
}
```

## Architecture

The module follows gostratum patterns:

- ✅ **Fx-first**: Native fx module integration
- ✅ **Configuration**: Uses core/configx with Bind pattern
- ✅ **Logging**: Uses core/logx for structured logging
- ✅ **Interface-driven**: All patterns implement interfaces
- ✅ **Composable**: Builder pattern for combining patterns
- ✅ **Context-aware**: Full context.Context support

## License

MIT License - See LICENSE file for details
