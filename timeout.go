package resilience

import (
	"context"
	"time"
)

// timeout implements the Timeout interface
type timeout struct {
	duration time.Duration
	name     string
}

// NewTimeout creates a new timeout
func NewTimeout(duration time.Duration, name string) Timeout {
	if duration == 0 {
		duration = DefaultTimeoutConfig().Duration
	}
	if name == "" {
		name = "default"
	}

	return &timeout{
		duration: duration,
		name:     name,
	}
}

func (t *timeout) Name() string {
	return t.name
}

func (t *timeout) Execute(ctx context.Context, fn func(context.Context) error) error {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, t.duration)
	defer cancel()

	// Execute with timeout
	errChan := make(chan error, 1)
	go func() {
		errChan <- fn(timeoutCtx)
	}()

	select {
	case err := <-errChan:
		return err
	case <-timeoutCtx.Done():
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return ErrTimeout
		}
		return timeoutCtx.Err()
	}
}
