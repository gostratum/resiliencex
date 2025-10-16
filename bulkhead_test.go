package resilience

import (
"context"
"sync"
"testing"
"time"

"github.com/stretchr/testify/assert"
)

func TestNewBulkhead(t *testing.T) {
	config := DefaultBulkheadConfig()
	bulkhead := NewBulkhead(config)
	assert.NotNil(t, bulkhead)
	assert.Equal(t, "default", bulkhead.Name())
}

func TestBulkheadConcurrency(t *testing.T) {
	t.Run("enforces max concurrency", func(t *testing.T) {
config := BulkheadConfig{
Name:          "test",
MaxConcurrent: 2,
MaxQueueSize:  0,
}
bulkhead := NewBulkhead(config)
ctx := context.Background()
		
		concurrent := 0
		maxConcurrent := 0
		mu := sync.Mutex{}
		
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				bulkhead.Execute(ctx, func(ctx context.Context) error {
mu.Lock()
					concurrent++
					if concurrent > maxConcurrent {
						maxConcurrent = concurrent
					}
					mu.Unlock()
					
					time.Sleep(10 * time.Millisecond)
					
					mu.Lock()
					concurrent--
					mu.Unlock()
					return nil
				})
			}()
		}
		
		wg.Wait()
		assert.LessOrEqual(t, maxConcurrent, 2)
	})
}

func TestBulkheadQueue(t *testing.T) {
	t.Run("queues requests when at capacity", func(t *testing.T) {
config := BulkheadConfig{
Name:          "test",
MaxConcurrent: 1,
MaxQueueSize:  2,
}
bulkhead := NewBulkhead(config)
ctx := context.Background()
		
		completed := 0
		mu := sync.Mutex{}
		
		var wg sync.WaitGroup
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				bulkhead.Execute(ctx, func(ctx context.Context) error {
time.Sleep(10 * time.Millisecond)
mu.Lock()
					completed++
					mu.Unlock()
					return nil
				})
			}()
		}
		
		wg.Wait()
		assert.Equal(t, 3, completed)
	})
}

func TestBulkheadFull(t *testing.T) {
	t.Run("rejects when bulkhead full", func(t *testing.T) {
config := BulkheadConfig{
Name:          "test",
MaxConcurrent: 1,
MaxQueueSize:  0,
}
bulkhead := NewBulkhead(config)

blockCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		
		go bulkhead.Execute(blockCtx, func(ctx context.Context) error {
<-ctx.Done()
			return nil
		})
		
		time.Sleep(10 * time.Millisecond)
		
		err := bulkhead.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
		
		assert.Error(t, err)
		assert.Equal(t, ErrBulkheadFull, err)
	})
}
