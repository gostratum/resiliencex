package resilience

import (
	"context"
	"sync"
	"time"
)

// circuitBreaker implements the CircuitBreaker interface
type circuitBreaker struct {
	config    CircuitBreakerConfig
	mu        sync.RWMutex
	state     CircuitState
	counts    *counts
	stateTime time.Time
}

// counts tracks circuit breaker statistics
type counts struct {
	requests       uint32
	totalSuccesses uint32
	totalFailures  uint32
	consecSuccess  uint32
	consecFailures uint32
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) CircuitBreaker {
	if config.MaxRequests == 0 {
		config.MaxRequests = DefaultCircuitBreakerConfig().MaxRequests
	}
	if config.Interval == 0 {
		config.Interval = DefaultCircuitBreakerConfig().Interval
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultCircuitBreakerConfig().Timeout
	}
	if config.FailureThreshold == 0 {
		config.FailureThreshold = DefaultCircuitBreakerConfig().FailureThreshold
	}
	if config.MinRequests == 0 {
		config.MinRequests = DefaultCircuitBreakerConfig().MinRequests
	}

	return &circuitBreaker{
		config:    config,
		state:     StateClosed,
		counts:    &counts{},
		stateTime: time.Now(),
	}
}

func (cb *circuitBreaker) Name() string {
	return cb.config.Name
}

func (cb *circuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *circuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	// Check if we can proceed
	generation, err := cb.beforeRequest()
	if err != nil {
		return err
	}

	// Execute the function
	err = fn(ctx)

	// Record the result
	cb.afterRequest(generation, err == nil)

	return err
}

func (cb *circuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.toNewGeneration(time.Now())
	cb.setState(StateClosed, time.Now())
}

func (cb *circuitBreaker) beforeRequest() (uint64, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state := cb.state

	switch state {
	case StateClosed:
		// Reset counts if interval has passed
		if now.Sub(cb.stateTime) > cb.config.Interval {
			cb.toNewGeneration(now)
		}

	case StateOpen:
		// Check if timeout has passed to move to half-open
		if now.Sub(cb.stateTime) > cb.config.Timeout {
			cb.setState(StateHalfOpen, now)
			return 0, nil
		}
		return 0, ErrCircuitOpen

	case StateHalfOpen:
		// Limit requests in half-open state
		if cb.counts.requests >= cb.config.MaxRequests {
			return 0, ErrCircuitOpen
		}
	}

	cb.counts.requests++
	return cb.currentGeneration(), nil
}

func (cb *circuitBreaker) afterRequest(generation uint64, success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	// Ignore if generation has changed
	if generation != cb.currentGeneration() {
		return
	}

	if success {
		cb.onSuccess(now)
	} else {
		cb.onFailure(now)
	}
}

func (cb *circuitBreaker) onSuccess(now time.Time) {
	cb.counts.totalSuccesses++
	cb.counts.consecSuccess++
	cb.counts.consecFailures = 0

	if cb.state == StateHalfOpen {
		// Transition to closed after consecutive successes
		if cb.counts.consecSuccess >= cb.config.MaxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

func (cb *circuitBreaker) onFailure(now time.Time) {
	cb.counts.totalFailures++
	cb.counts.consecFailures++
	cb.counts.consecSuccess = 0

	if cb.state == StateHalfOpen {
		// Transition back to open on any failure in half-open
		cb.setState(StateOpen, now)
		return
	}

	// Check if we should trip the circuit
	if cb.readyToTrip() {
		cb.setState(StateOpen, now)
	}
}

func (cb *circuitBreaker) readyToTrip() bool {
	// Need minimum requests before checking failure ratio
	if cb.counts.requests < cb.config.MinRequests {
		return false
	}

	failureRatio := float64(cb.counts.totalFailures) / float64(cb.counts.requests)
	return failureRatio >= cb.config.FailureThreshold
}

func (cb *circuitBreaker) setState(state CircuitState, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state
	cb.stateTime = now

	if cb.state == StateClosed {
		cb.toNewGeneration(now)
	}

	// Call state change callback
	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(cb.config.Name, prev, state)
	}
}

func (cb *circuitBreaker) toNewGeneration(now time.Time) {
	cb.counts = &counts{}
	cb.stateTime = now
}

func (cb *circuitBreaker) currentGeneration() uint64 {
	return uint64(cb.stateTime.UnixNano())
}
