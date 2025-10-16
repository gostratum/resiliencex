package resilience

import (
"testing"
"time"

"github.com/stretchr/testify/assert"
)

func TestConfigPrefix(t *testing.T) {
	cfg := Config{}
	assert.Equal(t, "resilience", cfg.Prefix())
}

func TestDefaultConfigs(t *testing.T) {
	t.Run("DefaultBulkheadConfig", func(t *testing.T) {
config := DefaultBulkheadConfig()
		assert.True(t, config.Enabled)
		assert.Equal(t, "default", config.Name)
		assert.Greater(t, config.MaxConcurrent, 0)
		assert.GreaterOrEqual(t, config.MaxQueueSize, 0)
	})
	
	t.Run("DefaultTimeoutConfig", func(t *testing.T) {
config := DefaultTimeoutConfig()
		assert.True(t, config.Enabled)
		assert.Greater(t, config.Duration, time.Duration(0))
	})
}

func TestResilienceErrors(t *testing.T) {
	t.Run("error messages", func(t *testing.T) {
assert.Equal(t, "resilience: circuit breaker is open", ErrCircuitOpen.Error())
		assert.Equal(t, "resilience: max retries exceeded", ErrMaxRetriesExceeded.Error())
		assert.Equal(t, "resilience: rate limit exceeded", ErrRateLimitExceeded.Error())
		assert.Equal(t, "resilience: bulkhead at capacity", ErrBulkheadFull.Error())
		assert.Equal(t, "resilience: operation timed out", ErrTimeout.Error())
	})
}
