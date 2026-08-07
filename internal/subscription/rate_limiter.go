package subscription

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type RateLimiter interface {
	Allow(ctx context.Context, userID string, maxAllowed int) (bool, int, error)
	Increment(ctx context.Context, userID string) error
}

type MemoryRateLimiter struct {
	mu       sync.Mutex
	counts   map[string]int
	lastReset time.Time
}

func NewMemoryRateLimiter() *MemoryRateLimiter {
	return &MemoryRateLimiter{
		counts:    make(map[string]int),
		lastReset: time.Now(),
	}
}

func (l *MemoryRateLimiter) key(userID string) string {
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s:%s", userID, today)
}

func (l *MemoryRateLimiter) Allow(ctx context.Context, userID string, maxAllowed int) (bool, int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	k := l.key(userID)
	current := l.counts[k]
	if current >= maxAllowed {
		return false, current, nil
	}
	return true, current, nil
}

func (l *MemoryRateLimiter) Increment(ctx context.Context, userID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	k := l.key(userID)
	l.counts[k]++
	return nil
}
