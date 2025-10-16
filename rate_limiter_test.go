package resilience

import (
"context"
"testing"
"time"

"github.com/stretchr/testify/assert"
)

func TestRateLimiterBasics(t *testing.T) {
	t.Run("creates rate limiter", func(t *testing.T) {
config := DefaultRateLimiterConfig()
		rl := NewRateLimiter(config)
		assert.NotNil(t, rl)
		assert.Equal(t, "default", rl.Name())
	})

	t.Run("allows requests within burst", func(t *testing.T) {
config := RateLimiterConfig{Name: "test", Rate: 10.0, Burst: 5}
rl := NewRateLimiter(config)

for i := 0; i < 5; i++ {
			assert.True(t, rl.Allow())
		}
		assert.False(t, rl.Allow())
	})

	t.Run("wait blocks until allowed", func(t *testing.T) {
config := RateLimiterConfig{Name: "test", Rate: 100.0, Burst: 1}
rl := NewRateLimiter(config)
ctx := context.Background()
		
		assert.True(t, rl.Allow())
		
		start := time.Now()
		err := rl.Wait(ctx)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Greater(t, duration, 5*time.Millisecond)
	})
}
