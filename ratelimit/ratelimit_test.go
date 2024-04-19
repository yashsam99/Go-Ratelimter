package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitMiddleware(t *testing.T) {
	// Create a new token bucket rate limiter
	rateLimiter := NewTokenBucketRateLimiter(time.Second)

	// Add rate limit configurations
	rateLimiter.AddConfig("/api/v1/users", &RateLimitConfig{
		MaxRequests: 1,
		Window:      time.Second,
	})
	rateLimiter.AddConfig("/api/v1/orders", &RateLimitConfig{
		MaxRequests: 1,
		Window:      time.Second,
	})

	// Create a test HTTP handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the handler with the rate limiting middleware
	middlewareHandler := RateLimitMiddleware(rateLimiter, handler)

	// Test case 1: Ensure rate limiting is enforced
	testRequest(t, middlewareHandler, "/api/v1/orders", 2, http.StatusTooManyRequests)

	// Test case 2: Ensure rate limiting is enforced for different endpoints
	testRequest(t, middlewareHandler, "/api/v1/users", 3, http.StatusTooManyRequests)

	// Test case 3: Ensure rate limiting is enforced for different client IPs
	testRequest(t, middlewareHandler, "/api/v1/users", 2, http.StatusOK, "127.0.0.1:8080")
	testRequest(t, middlewareHandler, "/api/v1/users", 2, http.StatusTooManyRequests, "127.0.0.1:8080")

	// Test case 4: Ensure rate limiting is not enforced when there is no configuration
	testRequest(t, middlewareHandler, "/api/v1/users", 10, http.StatusOK)
}

func testRequest(t *testing.T, handler http.Handler, path string, numRequests int, expectedStatus int, remoteAddr ...string) {
	for i := 0; i < numRequests; i++ {
		req, err := http.NewRequest("GET", path, nil)
		if err != nil {
			t.Errorf("Failed to create request: %v", err)
			return
		}

		if len(remoteAddr) > 0 {
			req.RemoteAddr = remoteAddr[0]
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != expectedStatus {
			t.Errorf("Unexpected status code for request %d to %s: got %d, want %d", i+1, path, rr.Code, expectedStatus)
		}
	}
}
