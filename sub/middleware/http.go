package middleware

import (
	"fmt"
	"net/http"
	"sync"
)

// RateLimiter interface that the rate limiter should implement
type RateLimiter interface {
	Allow() bool
}

// HTTPRateLimiter provides HTTP middleware for rate limiting
type HTTPRateLimiter struct {
	limiter      RateLimiter
	keyFunc      KeyFunc
	errorHandler ErrorHandler
	limiters     map[string]RateLimiter
	mu           sync.RWMutex
}

// KeyFunc extracts a key from the request for per-key rate limiting
type KeyFunc func(r *http.Request) string

// ErrorHandler handles rate limit errors
type ErrorHandler func(w http.ResponseWriter, r *http.Request)

// Options for configuring the HTTP rate limiter
type Options struct {
	KeyFunc      KeyFunc
	ErrorHandler ErrorHandler
}

// DefaultKeyFunc uses the client IP as the key
func DefaultKeyFunc(r *http.Request) string {
	// Try to get the real IP from X-Forwarded-For or X-Real-IP headers
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// DefaultErrorHandler returns a 429 Too Many Requests response
func DefaultErrorHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
}

// NewHTTPRateLimiter creates a new HTTP rate limiter middleware
func NewHTTPRateLimiter(limiter RateLimiter, opts *Options) *HTTPRateLimiter {
	rl := &HTTPRateLimiter{
		limiter:      limiter,
		keyFunc:      DefaultKeyFunc,
		errorHandler: DefaultErrorHandler,
	}
	
	if opts != nil {
		if opts.KeyFunc != nil {
			rl.keyFunc = opts.KeyFunc
		}
		if opts.ErrorHandler != nil {
			rl.errorHandler = opts.ErrorHandler
		}
	}
	
	return rl
}

// Middleware returns an HTTP middleware function
func (rl *HTTPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.limiter.Allow() {
			rl.errorHandler(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MiddlewareFunc returns an HTTP middleware function for use with http.HandlerFunc
func (rl *HTTPRateLimiter) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.limiter.Allow() {
			rl.errorHandler(w, r)
			return
		}
		next(w, r)
	}
}

// PerKeyHTTPRateLimiter provides per-key HTTP rate limiting
type PerKeyHTTPRateLimiter struct {
	limiterFactory LimiterFactory
	keyFunc        KeyFunc
	errorHandler   ErrorHandler
	limiters       sync.Map
}

// LimiterFactory creates new rate limiters for each key
type LimiterFactory func() RateLimiter

// NewPerKeyHTTPRateLimiter creates a new per-key HTTP rate limiter
func NewPerKeyHTTPRateLimiter(factory LimiterFactory, opts *Options) *PerKeyHTTPRateLimiter {
	rl := &PerKeyHTTPRateLimiter{
		limiterFactory: factory,
		keyFunc:        DefaultKeyFunc,
		errorHandler:   DefaultErrorHandler,
	}
	
	if opts != nil {
		if opts.KeyFunc != nil {
			rl.keyFunc = opts.KeyFunc
		}
		if opts.ErrorHandler != nil {
			rl.errorHandler = opts.ErrorHandler
		}
	}
	
	return rl
}

// Middleware returns an HTTP middleware function with per-key rate limiting
func (rl *PerKeyHTTPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rl.keyFunc(r)
		
		// Get or create limiter for this key
		limiterInterface, _ := rl.limiters.LoadOrStore(key, rl.limiterFactory())
		limiter := limiterInterface.(RateLimiter)
		
		if !limiter.Allow() {
			rl.errorHandler(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MiddlewareFunc returns an HTTP middleware function for use with http.HandlerFunc
func (rl *PerKeyHTTPRateLimiter) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := rl.keyFunc(r)
		
		// Get or create limiter for this key
		limiterInterface, _ := rl.limiters.LoadOrStore(key, rl.limiterFactory())
		limiter := limiterInterface.(RateLimiter)
		
		if !limiter.Allow() {
			rl.errorHandler(w, r)
			return
		}
		next(w, r)
	}
}

// CustomErrorHandler creates an error handler with custom message and headers
func CustomErrorHandler(message string, headers map[string]string) ErrorHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		http.Error(w, message, http.StatusTooManyRequests)
	}
}

// JSONErrorHandler returns a JSON error response
func JSONErrorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	fmt.Fprintf(w, `{"error":"too many requests","status":429}`)
}

// KeyFuncs provides common key extraction functions
var KeyFuncs = struct {
	ByIP        KeyFunc
	ByUserID    func(headerName string) KeyFunc
	ByAPIKey    func(headerName string) KeyFunc
	ByPath      KeyFunc
	Combination func(funcs ...KeyFunc) KeyFunc
}{
	ByIP: DefaultKeyFunc,
	
	ByUserID: func(headerName string) KeyFunc {
		return func(r *http.Request) string {
			if userID := r.Header.Get(headerName); userID != "" {
				return userID
			}
			return "anonymous"
		}
	},
	
	ByAPIKey: func(headerName string) KeyFunc {
		return func(r *http.Request) string {
			if apiKey := r.Header.Get(headerName); apiKey != "" {
				return apiKey
			}
			return "no-api-key"
		}
	},
	
	ByPath: func(r *http.Request) string {
		return r.URL.Path
	},
	
	Combination: func(funcs ...KeyFunc) KeyFunc {
		return func(r *http.Request) string {
			var keys []string
			for _, fn := range funcs {
				keys = append(keys, fn(r))
			}
			return fmt.Sprintf("%v", keys)
		}
	},
}