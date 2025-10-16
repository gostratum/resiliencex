package resilience

import (
	"context"
	"sync"
	"time"
)

// rateLimiter implements the RateLimiter interface using token bucket algorithm
type rateLimiter struct {
	config   RateLimiterConfig
	mu       sync.Mutex
	tokens   float64
	lastTime time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) RateLimiter {
	if config.Rate == 0 {
		config.Rate = DefaultRateLimiterConfig().Rate
	}
	if config.Burst == 0 {
		config.Burst = DefaultRateLimiterConfig().Burst
	}

	return &rateLimiter{
		config:   config,
		tokens:   float64(config.Burst),
		lastTime: time.Now(),
	}
}

func (rl *rateLimiter) Name() string {
	return rl.config.Name
}

func (rl *rateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.refillTokens(now)

	if rl.tokens >= 1.0 {
		rl.tokens--
		return true
	}

	// Call rate limit callback
	if rl.config.OnRateLimit != nil {
		rl.config.OnRateLimit(rl.config.Name)
	}

	return false
}

func (rl *rateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow() {
			return nil
		}

		// Calculate wait time for next token
		waitTime := rl.nextTokenDuration()

		// Wait or context cancellation
		select {
		case <-time.After(waitTime):
			// Try again
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (rl *rateLimiter) refillTokens(now time.Time) {
	elapsed := now.Sub(rl.lastTime)
	rl.lastTime = now

	// Add tokens based on elapsed time and rate
	tokensToAdd := rl.config.Rate * elapsed.Seconds()
	rl.tokens += tokensToAdd

	// Cap at burst limit
	if rl.tokens > float64(rl.config.Burst) {
		rl.tokens = float64(rl.config.Burst)
	}
}

func (rl *rateLimiter) nextTokenDuration() time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Calculate time until next token is available
	tokensNeeded := 1.0 - rl.tokens
	if tokensNeeded <= 0 {
		return 0
	}

	// Time = tokens / rate
	seconds := tokensNeeded / rl.config.Rate
	return time.Duration(seconds * float64(time.Second))
}
