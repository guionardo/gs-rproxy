package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	// Wait to run in a round time
	wait := time.Now().UnixMilli() % 1000
	time.Sleep(time.Millisecond * time.Duration(1000-wait))

	rl := NewRateLimit(10)
	assert.NotNil(t, rl)
	wg := sync.WaitGroup{}
	var succ, errors int64
	for _ = range 25 {
		wg.Add(1)
		go func() {
			err := rl.CanRequest()
			if err == nil {
				atomic.AddInt64(&succ, 1)
				// rl.Release()
			} else {
				atomic.AddInt64(&errors, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Equal(t, int64(10), succ)
	assert.Equal(t, int64(15), errors)
}
