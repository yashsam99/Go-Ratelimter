package main

import (
	"fmt"
	"net/http"
	"time"

	ratelimit "goratelimiter/ratelimit"
)

func handleUsers(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello from users endpoint")
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello from orders endpoint")
}

func main() {
	// Create a new token bucket rate limiter
	rateLimiter := ratelimit.NewTokenBucketRateLimiter(time.Second)

	// Add rate limit configurations
	rateLimiter.AddConfig("/api/v1/users", &ratelimit.RateLimitConfig{
		MaxRequests: 0,
		Window:      time.Second,
	})
	rateLimiter.AddConfig("/api/v1/orders", &ratelimit.RateLimitConfig{
		MaxRequests: 0,
		Window:      time.Second,
	})

	// Create a new HTTP server with the rate limiting middleware
	mux := http.NewServeMux()
	mux.Handle("/api/v1/users", ratelimit.RateLimitMiddleware(rateLimiter, http.HandlerFunc(handleUsers)))
	mux.Handle("/api/v1/orders", ratelimit.RateLimitMiddleware(rateLimiter, http.HandlerFunc(handleOrders)))

	// Start the HTTP server
	server := &http.Server{
		Addr:           ":8080",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	fmt.Println("Starting server at :8080")
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
