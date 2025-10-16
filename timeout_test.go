package resilience

import (
"context"
"testing"
"time"

"github.com/stretchr/testify/assert"
)

func TestNewTimeout(t *testing.T) {
	timeout := NewTimeout(1*time.Second, "test")
	assert.NotNil(t, timeout)
	assert.Equal(t, "test", timeout.Name())
}

func TestTimeoutExecution(t *testing.T) {
	t.Run("completes within timeout", func(t *testing.T) {
timeout := NewTimeout(100*time.Millisecond, "test")
ctx := context.Background()
		
		err := timeout.Execute(ctx, func(ctx context.Context) error {
time.Sleep(10 * time.Millisecond)
return nil
})
		
		assert.NoError(t, err)
	})
	
	t.Run("fails when timeout exceeded", func(t *testing.T) {
timeout := NewTimeout(50*time.Millisecond, "test")
ctx := context.Background()
		
		err := timeout.Execute(ctx, func(ctx context.Context) error {
time.Sleep(100 * time.Millisecond)
return nil
})
		
		assert.Error(t, err)
		assert.Equal(t, ErrTimeout, err)
	})
}
