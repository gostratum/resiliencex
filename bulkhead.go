package resilience

import (
	"context"
)

// bulkhead implements the Bulkhead interface using semaphore pattern
type bulkhead struct {
	config BulkheadConfig
	sem    chan struct{}
	queue  chan struct{}
}

// NewBulkhead creates a new bulkhead
func NewBulkhead(config BulkheadConfig) Bulkhead {
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = DefaultBulkheadConfig().MaxConcurrent
	}
	if config.MaxQueueSize == 0 {
		config.MaxQueueSize = DefaultBulkheadConfig().MaxQueueSize
	}

	return &bulkhead{
		config: config,
		sem:    make(chan struct{}, config.MaxConcurrent),
		queue:  make(chan struct{}, config.MaxQueueSize),
	}
}

func (b *bulkhead) Name() string {
	return b.config.Name
}

func (b *bulkhead) Execute(ctx context.Context, fn func(context.Context) error) error {
	// Try to acquire a slot
	select {
	case b.sem <- struct{}{}:
		// Got a slot, execute immediately
		defer func() { <-b.sem }()
		return fn(ctx)

	default:
		// No slot available, try to queue
		select {
		case b.queue <- struct{}{}:
			// Queued successfully
			defer func() { <-b.queue }()

			// Wait for a slot
			select {
			case b.sem <- struct{}{}:
				defer func() { <-b.sem }()
				return fn(ctx)
			case <-ctx.Done():
				return ctx.Err()
			}

		default:
			// Queue is full
			if b.config.OnBulkheadFull != nil {
				b.config.OnBulkheadFull(b.config.Name)
			}
			return ErrBulkheadFull
		}
	}
}

func (b *bulkhead) Available() int {
	return b.config.MaxConcurrent - len(b.sem)
}
