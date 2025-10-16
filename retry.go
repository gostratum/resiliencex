package resilience

import (
	"context"
	"math/rand"
	"time"
)

// retry implements the Retry interface
type retry struct {
	config  RetryConfig
	backoff BackoffStrategy
}

// NewRetry creates a new retry instance
func NewRetry(config RetryConfig) Retry {
	if config.MaxAttempts == 0 {
		config.MaxAttempts = DefaultRetryConfig().MaxAttempts
	}
	if config.InitialInterval == 0 {
		config.InitialInterval = DefaultRetryConfig().InitialInterval
	}
	if config.MaxInterval == 0 {
		config.MaxInterval = DefaultRetryConfig().MaxInterval
	}
	if config.Multiplier == 0 {
		config.Multiplier = DefaultRetryConfig().Multiplier
	}
	if config.RandomizationFactor == 0 {
		config.RandomizationFactor = DefaultRetryConfig().RandomizationFactor
	}

	return &retry{
		config: config,
		backoff: &exponentialBackoff{
			initialInterval:     config.InitialInterval,
			maxInterval:         config.MaxInterval,
			multiplier:          config.Multiplier,
			randomizationFactor: config.RandomizationFactor,
		},
	}
}

func (r *retry) Name() string {
	return r.config.Name
}

func (r *retry) Execute(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt < r.config.MaxAttempts; attempt++ {
		// Execute the function
		err := fn(ctx)

		// Success - no retry needed
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry this error
		if r.config.ShouldRetry != nil && !r.config.ShouldRetry(err) {
			return err
		}

		// Check if this was the last attempt
		if attempt == r.config.MaxAttempts-1 {
			break
		}

		// Call retry callback
		if r.config.OnRetry != nil {
			r.config.OnRetry(attempt+1, err)
		}

		// Calculate backoff delay
		delay := r.backoff.Next(attempt)

		// Wait for backoff or context cancellation
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return lastErr
}

// exponentialBackoff implements exponential backoff with jitter
type exponentialBackoff struct {
	initialInterval     time.Duration
	maxInterval         time.Duration
	multiplier          float64
	randomizationFactor float64
}

func (b *exponentialBackoff) Next(attempt int) time.Duration {
	// Calculate exponential backoff
	interval := float64(b.initialInterval)
	for i := 0; i < attempt; i++ {
		interval *= b.multiplier
	}

	// Cap at max interval
	if interval > float64(b.maxInterval) {
		interval = float64(b.maxInterval)
	}

	// Add jitter
	delta := b.randomizationFactor * interval
	minInterval := interval - delta
	maxInterval := interval + delta

	// Random value between min and max
	jitter := minInterval + rand.Float64()*(maxInterval-minInterval)

	return time.Duration(jitter)
}

// constantBackoff implements constant backoff
type constantBackoff struct {
	interval time.Duration
}

func (b *constantBackoff) Next(attempt int) time.Duration {
	return b.interval
}

// linearBackoff implements linear backoff
type linearBackoff struct {
	initialInterval time.Duration
	increment       time.Duration
	maxInterval     time.Duration
}

func (b *linearBackoff) Next(attempt int) time.Duration {
	interval := b.initialInterval + time.Duration(attempt)*b.increment
	if interval > b.maxInterval {
		interval = b.maxInterval
	}
	return interval
}
