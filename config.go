package resilience

import (
	"time"

	"github.com/gostratum/core/configx"
)

// Config represents the configuration for the resilience module
type Config struct {
	// CircuitBreaker configuration
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`

	// Retry configuration
	Retry RetryConfig `mapstructure:"retry"`

	// RateLimiter configuration
	RateLimiter RateLimiterConfig `mapstructure:"rate_limiter"`

	// Bulkhead configuration
	Bulkhead BulkheadConfig `mapstructure:"bulkhead"`

	// Timeout configuration
	Timeout TimeoutConfig `mapstructure:"timeout"`
}

// Prefix returns the configuration prefix for resilience
func (Config) Prefix() string {
	return "resilience"
}

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	// Enabled determines if circuit breaker is enabled
	Enabled bool `mapstructure:"enabled"`

	// Name is the circuit breaker identifier
	Name string `mapstructure:"name"`

	// MaxRequests is the max requests allowed in half-open state
	MaxRequests uint32 `mapstructure:"max_requests"`

	// Interval is the cyclic period in closed state for resetting counters
	Interval time.Duration `mapstructure:"interval"`

	// Timeout is the period of open state before transitioning to half-open
	Timeout time.Duration `mapstructure:"timeout"`

	// ReadyToTrip determines when to trip the circuit to open state
	// Circuit trips when failure ratio > threshold and request count > min requests
	FailureThreshold float64 `mapstructure:"failure_threshold"`

	// MinRequests is the minimum requests needed before checking failure ratio
	MinRequests uint32 `mapstructure:"min_requests"`

	// OnStateChange is called when state changes
	OnStateChange OnStateChange `mapstructure:"-"`
}

// DefaultCircuitBreakerConfig returns default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Enabled:          true,
		Name:             "default",
		MaxRequests:      5,
		Interval:         60 * time.Second,
		Timeout:          30 * time.Second,
		FailureThreshold: 0.6, // 60% failure rate
		MinRequests:      10,
	}
}

// RetryConfig configures retry behavior
type RetryConfig struct {
	// Enabled determines if retry is enabled
	Enabled bool `mapstructure:"enabled"`

	// Name is the retry identifier
	Name string `mapstructure:"name"`

	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int `mapstructure:"max_attempts"`

	// InitialInterval is the initial backoff interval
	InitialInterval time.Duration `mapstructure:"initial_interval"`

	// MaxInterval is the maximum backoff interval
	MaxInterval time.Duration `mapstructure:"max_interval"`

	// Multiplier is the backoff multiplier
	Multiplier float64 `mapstructure:"multiplier"`

	// RandomizationFactor adds jitter to prevent thundering herd
	RandomizationFactor float64 `mapstructure:"randomization_factor"`

	// ShouldRetry determines if an error should trigger a retry
	ShouldRetry ShouldRetry `mapstructure:"-"`

	// OnRetry is called before each retry attempt
	OnRetry OnRetry `mapstructure:"-"`
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Enabled:             true,
		Name:                "default",
		MaxAttempts:         3,
		InitialInterval:     100 * time.Millisecond,
		MaxInterval:         10 * time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0.5,
	}
}

// RateLimiterConfig configures rate limiter behavior
type RateLimiterConfig struct {
	// Enabled determines if rate limiter is enabled
	Enabled bool `mapstructure:"enabled"`

	// Name is the rate limiter identifier
	Name string `mapstructure:"name"`

	// Rate is the number of requests per second
	Rate float64 `mapstructure:"rate"`

	// Burst is the maximum burst size
	Burst int `mapstructure:"burst"`

	// OnRateLimit is called when rate limit is exceeded
	OnRateLimit OnRateLimit `mapstructure:"-"`
}

// DefaultRateLimiterConfig returns default rate limiter configuration
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		Enabled: true,
		Name:    "default",
		Rate:    100.0, // 100 requests per second
		Burst:   200,   // Allow burst of 200
	}
}

// BulkheadConfig configures bulkhead behavior
type BulkheadConfig struct {
	// Enabled determines if bulkhead is enabled
	Enabled bool `mapstructure:"enabled"`

	// Name is the bulkhead identifier
	Name string `mapstructure:"name"`

	// MaxConcurrent is the maximum number of concurrent operations
	MaxConcurrent int `mapstructure:"max_concurrent"`

	// MaxQueueSize is the maximum queue size for waiting operations
	MaxQueueSize int `mapstructure:"max_queue_size"`

	// OnBulkheadFull is called when bulkhead is at capacity
	OnBulkheadFull OnBulkheadFull `mapstructure:"-"`
}

// DefaultBulkheadConfig returns default bulkhead configuration
func DefaultBulkheadConfig() BulkheadConfig {
	return BulkheadConfig{
		Enabled:       true,
		Name:          "default",
		MaxConcurrent: 10,
		MaxQueueSize:  100,
	}
}

// TimeoutConfig configures timeout behavior
type TimeoutConfig struct {
	// Enabled determines if timeout is enabled
	Enabled bool `mapstructure:"enabled"`

	// Duration is the timeout duration
	Duration time.Duration `mapstructure:"duration"`
}

// DefaultTimeoutConfig returns default timeout configuration
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Enabled:  true,
		Duration: 30 * time.Second,
	}
}

// NewConfig creates a new Config from the configuration loader
func NewConfig(loader configx.Loader) (Config, error) {
	var cfg Config

	// Set defaults
	cfg.CircuitBreaker = DefaultCircuitBreakerConfig()
	cfg.Retry = DefaultRetryConfig()
	cfg.RateLimiter = DefaultRateLimiterConfig()
	cfg.Bulkhead = DefaultBulkheadConfig()
	cfg.Timeout = DefaultTimeoutConfig()

	// Bind configuration
	if err := loader.Bind(&cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// Sanitize returns a copy of the resilience Config. There are typically no
// secrets here, but we provide the method for consistency across modules.
func (c *Config) Sanitize() *Config {
	out := *c
	// Shallow copy of nested structs is sufficient as they do not hold secrets
	out.CircuitBreaker = c.CircuitBreaker
	out.Retry = c.Retry
	out.RateLimiter = c.RateLimiter
	out.Bulkhead = c.Bulkhead
	out.Timeout = c.Timeout
	return &out
}

// ConfigSummary returns a compact diagnostic map safe for logging.
func (c *Config) ConfigSummary() map[string]any {
	return map[string]any{
		"circuit_breaker_enabled": c.CircuitBreaker.Enabled,
		"retry_enabled":           c.Retry.Enabled,
		"rate_limiter_enabled":    c.RateLimiter.Enabled,
		"bulkhead_enabled":        c.Bulkhead.Enabled,
	}
}
