package resilience

import (
	"context"

	"github.com/gostratum/core/configx"
	"github.com/gostratum/core/logx"
	"go.uber.org/fx"
)

// Module provides the resilience module for fx
func Module() fx.Option {
	return fx.Module("resiliencex",
		fx.Provide(
			NewConfig,
			NewProvider,
		),
	)
}

// Params contains dependencies for the resilience provider
type Params struct {
	fx.In

	Config configx.Loader
	Logger logx.Logger
}

// Result contains the resilience provider outputs
type Result struct {
	fx.Out

	Builder Builder
}

// NewProvider creates a new resilience provider
func NewProvider(params Params) (Result, error) {
	cfg, err := NewConfig(params.Config)
	if err != nil {
		return Result{}, err
	}

	params.Logger.Info("Initializing resilience module",
		logx.String("circuit_breaker", cfg.CircuitBreaker.Name),
		logx.String("retry", cfg.Retry.Name),
		logx.String("rate_limiter", cfg.RateLimiter.Name),
		logx.String("bulkhead", cfg.Bulkhead.Name),
	)

	// Create builder with default configuration
	builder := NewBuilder().WithName("default-executor")

	// Add circuit breaker if enabled
	if cfg.CircuitBreaker.Enabled {
		builder = builder.WithCircuitBreaker(cfg.CircuitBreaker)
		params.Logger.Info("Circuit breaker enabled",
			logx.String("name", cfg.CircuitBreaker.Name),
			logx.Float64("failure_threshold", cfg.CircuitBreaker.FailureThreshold),
		)
	}

	// Add retry if enabled
	if cfg.Retry.Enabled {
		builder = builder.WithRetry(cfg.Retry)
		params.Logger.Info("Retry enabled",
			logx.String("name", cfg.Retry.Name),
			logx.Int("max_attempts", cfg.Retry.MaxAttempts),
		)
	}

	// Add rate limiter if enabled
	if cfg.RateLimiter.Enabled {
		builder = builder.WithRateLimiter(cfg.RateLimiter)
		params.Logger.Info("Rate limiter enabled",
			logx.String("name", cfg.RateLimiter.Name),
			logx.Float64("rate", cfg.RateLimiter.Rate),
		)
	}

	// Add bulkhead if enabled
	if cfg.Bulkhead.Enabled {
		builder = builder.WithBulkhead(cfg.Bulkhead)
		params.Logger.Info("Bulkhead enabled",
			logx.String("name", cfg.Bulkhead.Name),
			logx.Int("max_concurrent", cfg.Bulkhead.MaxConcurrent),
		)
	}

	// Add timeout if enabled
	if cfg.Timeout.Enabled {
		builder = builder.WithTimeout(cfg.Timeout.Duration)
		params.Logger.Info("Timeout enabled",
			logx.Duration("duration", cfg.Timeout.Duration),
		)
	}

	return Result{
		Builder: builder,
	}, nil
}

// LifecycleHooks adds lifecycle hooks for the resilience module
func LifecycleHooks(lc fx.Lifecycle, logger logx.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Resilience module started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Resilience module stopped")
			return nil
		},
	})
}
