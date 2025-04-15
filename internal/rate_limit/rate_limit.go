package ratelimit

import (
	"fmt"
	"sync"
	"time"

	"github.com/guionardo/gs-rproxy/internal/logger"
	"go.uber.org/zap"
)

type RateLimit struct {
	rps     int
	current int64
	count   int
	lock    sync.RWMutex
}

func NewRateLimit(rps int) *RateLimit {
	if rps < 0 {
		rps = 0
	}
	return &RateLimit{
		rps: rps,
	}
}

func (r *RateLimit) CanRequest() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	now := time.Now().Unix()
	log := logger.Log.With(
		zap.Int("current", int(r.current)),
		zap.Int("count", r.count),
		zap.Int64("now", now),
		zap.Int("rps", r.rps))
	log.Debug("Rate Limit")

	if now == r.current {
		// same second
		r.count = r.count + 1
		if r.count > r.rps {
			return fmt.Errorf("rate limit: %d RPS", r.rps)
		}
		return nil
	}

	// next second
	r.count = 1
	r.current = now
	return nil

}

func (r *RateLimit) Release() {
	r.lock.Lock()
	if r.rps > 0 {
		r.count--
	}
	r.lock.Unlock()
}

func (r *RateLimit) String() string {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.rps > 0 {
		return fmt.Sprintf("%d/%d RPS", r.count, r.rps)
	}
	return "unlimited"
}

func (r *RateLimit) RPS() int {
	return r.rps
}
