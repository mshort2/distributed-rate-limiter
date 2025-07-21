package test

import (
	"context"
	"testing"
	"time"

	"github.com/mshort2/distributed-rate-limiter/internal/config"
	"github.com/mshort2/distributed-rate-limiter/internal/middleware"
	limiter "github.com/mshort2/distributed-rate-limiter/pkg/ratelimiter"
)

func TestSlidingWindowLimiter(t *testing.T) {
	cfg := config.Load()
	limiter, err := limiter.NewRateLimiter(cfg, 5, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}

	clientID := "test-client"
	_, _ = limiter.RedisDB.ZRemRangeByScore(context.Background(), clientID, "0", "+inf")
	t.Run("allows requests within limit", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			counter, err := limiter.RedisDB.ZCard(context.Background(), clientID)
			if err != nil {
				t.Fatalf("Failed to get ZCard: %v", err)
			}
			t.Logf("Current count: %d", counter)
			requestID := middleware.GenerateRequestID()
			resp, err := limiter.Allow(context.Background(), clientID, requestID)
			t.Logf("i: %d, remaining: %d", i, resp.Remaining)
			t.Logf("Request %s: allowed=%v, remaining=%d", requestID, resp.Allowed, resp.Remaining)
			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}
			if !resp.Allowed {
				t.Fatalf("Request %d should be allowed", i)
			}
			expectedRemaining := 4 - i
			if resp.Remaining != expectedRemaining {
				t.Errorf("Expected remaining %d, got %d", expectedRemaining, resp.Remaining)
			}
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		resp, err := limiter.Allow(context.Background(), clientID, middleware.GenerateRequestID())
		if err != nil {
			t.Fatalf("Request over limit failed: %v", err)
		}
		if resp.Allowed {
			t.Error("Request should be blocked")
		}
		if resp.Remaining != 0 {
			t.Errorf("Expected remaining 0, got %d", resp.Remaining)
		}
	})

	t.Run("sliding window resets after expiry", func(t *testing.T) {
		time.Sleep(2 * time.Second)
		resp, err := limiter.Allow(context.Background(), clientID, middleware.GenerateRequestID())
		if err != nil {
			t.Fatalf("Request after window failed: %v", err)
		}
		if !resp.Allowed {
			t.Error("Request after window should be allowed")
		}
		if resp.Remaining != 4 {
			t.Errorf("Expected remaining 4, got %d", resp.Remaining)
		}
	})
}

func TestMultipleClients(t *testing.T) {
	cfg := config.Load()
	limiter, err := limiter.NewRateLimiter(cfg, 3, 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}

	clientA := "clientA"
	clientB := "clientB"

	t.Run("separate limits per client", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			respA, err := limiter.Allow(context.Background(), clientA, middleware.GenerateRequestID())
			t.Logf("clientA request %d: allowed=%v, remaining=%d", i, respA.Allowed, respA.Remaining)
			if err != nil {
				t.Fatalf("clientA request %d failed: %v", i, err)
			}
			if !respA.Allowed {
				t.Fatalf("clientA request %d should be allowed", i)
			}

			respB, err := limiter.Allow(context.Background(), clientB, middleware.GenerateRequestID())
			t.Logf("clientB request %d: allowed=%v, remaining=%d", i, respB.Allowed, respB.Remaining)
			if err != nil {
				t.Fatalf("clientB request %d failed: %v", i, err)
			}
			if !respB.Allowed {
				t.Fatalf("clientB request %d should be allowed", i)
			}
		}

		respA, err := limiter.Allow(context.Background(), clientA, middleware.GenerateRequestID())
		t.Logf("clientA over limit: allowed=%v, remaining=%d", respA.Allowed, respA.Remaining)
		if err != nil {
			t.Fatalf("clientA over limit failed: %v", err)
		}
		if respA.Allowed {
			t.Error("clientA request over limit should be blocked")
		}

		respB, err := limiter.Allow(context.Background(), clientB, middleware.GenerateRequestID())
		t.Logf("clientB over limit: allowed=%v, remaining=%d", respB.Allowed, respB.Remaining)
		if err != nil {
			t.Fatalf("clientB over limit failed: %v", err)
		}
		if respB.Allowed {
			t.Error("clientB request over limit should be blocked")
		}
	})
}
