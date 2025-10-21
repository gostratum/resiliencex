package resilience

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCircuitStateString(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestBulkheadAvailable(t *testing.T) {
	config := BulkheadConfig{
		Name:          "test",
		MaxConcurrent: 5,
		MaxQueueSize:  2,
	}
	bulkhead := NewBulkhead(config)

	// Initially should have all slots available
	available := bulkhead.Available()
	assert.Equal(t, 5, available)
}
