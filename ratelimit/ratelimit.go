package ratelimit

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimitConfig represents the configuration for rate limiting.
type RateLimitConfig struct {
	// The maximum number of requests allowed within the time window.
	MaxRequests int64

	// The time window duration in seconds.
	Window time.Duration
}

// RateLimiter is an interface for rate limiting implementations.
type RateLimiter interface {
	// Allow checks if the request should be allowed based on the rate limit.
	Allow(ctx context.Context, key string) bool
}

// tokenBucketRateLimiter is an implementation of RateLimiter using the token bucket algorithm.
type tokenBucketRateLimiter struct {
	sync.Mutex
	configs         map[string]*RateLimitConfig
	buckets         map[string]*tokenBucket
	cleanupInterval time.Duration
}

type tokenBucket struct {
	tokens     int64
	lastRefill time.Time
	maxTokens  int64
	refillRate float64
	allowBurst bool
	syncMutex  sync.Mutex
}

// NewTokenBucketRateLimiter creates a new instance of tokenBucketRateLimiter.
func NewTokenBucketRateLimiter(cleanupInterval time.Duration) *tokenBucketRateLimiter {
	return &tokenBucketRateLimiter{
		configs:         make(map[string]*RateLimitConfig),
		buckets:         make(map[string]*tokenBucket),
		cleanupInterval: cleanupInterval,
	}
}

// AddConfig adds a new rate limit configuration for the given key.
func (rl *tokenBucketRateLimiter) AddConfig(key string, config *RateLimitConfig) {
	rl.Lock()
	defer rl.Unlock()

	rl.configs[key] = config
	rl.buckets[key] = newTokenBucket(config.MaxRequests, config.Window, true)
}

// RemoveConfig removes the rate limit configuration for the given key.
func (rl *tokenBucketRateLimiter) RemoveConfig(key string) {
	rl.Lock()
	defer rl.Unlock()

	delete(rl.configs, key)
	delete(rl.buckets, key)
}

// Allow checks if the request should be allowed based on the rate limit.
func (rl *tokenBucketRateLimiter) Allow(ctx context.Context, key string) bool {
	rl.Lock()
	config, ok := rl.configs[key]
	fmt.Println(" key is ", config, ok)
	if !ok {
		rl.configs[key] = &RateLimitConfig{
			MaxRequests: 1,
			Window:      1,
		}
		rl.Unlock()
		return true
	}
	if config.MaxRequests == 0 {
		rl.Unlock()
		return true
	}
	bucket, ok := rl.buckets[key]
	rl.Unlock()
	fmt.Println("ok value 2", ok)
	if !ok {
		fmt.Print("going to create new bucket")
		bucket = newTokenBucket(config.MaxRequests, config.Window, true)
		fmt.Println(bucket)
		rl.Lock()
		rl.buckets[key] = bucket
		rl.Unlock()
	}

	return bucket.consume()
}

// newTokenBucket creates a new token bucket with the given configuration.
func newTokenBucket(maxTokens int64, window time.Duration, allowBurst bool) *tokenBucket {
	bucket := &tokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: float64(maxTokens) / window.Seconds(),
		allowBurst: allowBurst,
	}
	bucket.lastRefill = time.Now()
	return bucket
}

// consume attempts to consume a token from the bucket and returns true if successful.
func (tb *tokenBucket) consume() bool {
	tb.syncMutex.Lock()
	defer tb.syncMutex.Unlock()

	tb.refill()
	if tb.tokens > 0 || (tb.allowBurst && tb.tokens == 0) {
		tb.tokens--
		return true
	}
	return false
}

// refill adds tokens to the bucket based on the elapsed time since the last refill.
func (tb *tokenBucket) refill() {
	now := time.Now()
	elapsedTime := now.Sub(tb.lastRefill)
	tb.lastRefill = now
	tb.tokens = int64(float64(tb.tokens) + elapsedTime.Seconds()*tb.refillRate)
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
}

// RateLimitMiddleware is a middleware function that enforces rate limiting on incoming requests.
func RateLimitMiddleware(rl RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Unable to parse client IP address", http.StatusInternalServerError)
			return
		}

		key := r.URL.Path + ":" + ip
		ctx := context.Background()
		fmt.Println(key)
		if !rl.Allow(ctx, key) {
			fmt.Println("RateLimit Exceeded!")
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		fmt.Println("serving the request ")
		next.ServeHTTP(w, r)
	})
}
