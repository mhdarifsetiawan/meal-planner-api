package subscription_test

import (
	"context"
	"testing"

	"meal-planner-api/internal/subscription"
)

func TestMemoryRateLimiter(t *testing.T) {
	limiter := subscription.NewMemoryRateLimiter()
	ctx := context.Background()
	userID := "test-user-123"

	// 1. Initial check (limit 2)
	allowed, count, err := limiter.Allow(ctx, userID, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed || count != 0 {
		t.Fatalf("expected allowed=true, count=0, got allowed=%v, count=%d", allowed, count)
	}

	// 2. Increment 1
	if err := limiter.Increment(ctx, userID); err != nil {
		t.Fatalf("failed to increment: %v", err)
	}

	// 3. Second check (limit 2)
	allowed, count, err = limiter.Allow(ctx, userID, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed || count != 1 {
		t.Fatalf("expected allowed=true, count=1, got allowed=%v, count=%d", allowed, count)
	}

	// 4. Increment 2
	if err := limiter.Increment(ctx, userID); err != nil {
		t.Fatalf("failed to increment: %v", err)
	}

	// 5. Third check (limit 2) - should be blocked
	allowed, count, err = limiter.Allow(ctx, userID, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed || count != 2 {
		t.Fatalf("expected allowed=false, count=2, got allowed=%v, count=%d", allowed, count)
	}
}

func TestNilPostgresRateLimiter(t *testing.T) {
	limiter := subscription.NewPostgresRateLimiter(nil)
	ctx := context.Background()
	userID := "test-user-nil"

	allowed, count, err := limiter.Allow(ctx, userID, 3)
	if err != nil {
		t.Fatalf("unexpected error on nil db pool: %v", err)
	}
	if !allowed || count != 0 {
		t.Fatalf("expected allowed=true, count=0 for nil pool, got allowed=%v, count=%d", allowed, count)
	}

	err = limiter.Increment(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error on nil db pool increment: %v", err)
	}
}
