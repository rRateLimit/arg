package stats

import (
	"sync"
	"time"
)

// Stats holds rate limiter statistics
type Stats struct {
	TotalRequests    int64
	AllowedRequests  int64
	DeniedRequests   int64
	StartTime        time.Time
	LastRequestTime  time.Time
	mu               sync.RWMutex
}

// NewStats creates a new Stats instance
func NewStats() *Stats {
	return &Stats{
		StartTime: time.Now(),
	}
}

// RecordAllowed records an allowed request
func (s *Stats) RecordAllowed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.TotalRequests++
	s.AllowedRequests++
	s.LastRequestTime = time.Now()
}

// RecordDenied records a denied request
func (s *Stats) RecordDenied() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.TotalRequests++
	s.DeniedRequests++
	s.LastRequestTime = time.Now()
}

// GetSnapshot returns a copy of current statistics
func (s *Stats) GetSnapshot() StatsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	duration := time.Since(s.StartTime)
	if s.LastRequestTime.After(s.StartTime) {
		duration = s.LastRequestTime.Sub(s.StartTime)
	}
	
	var rate float64
	if duration.Seconds() > 0 {
		rate = float64(s.AllowedRequests) / duration.Seconds()
	}
	
	return StatsSnapshot{
		TotalRequests:   s.TotalRequests,
		AllowedRequests: s.AllowedRequests,
		DeniedRequests:  s.DeniedRequests,
		StartTime:       s.StartTime,
		LastRequestTime: s.LastRequestTime,
		Duration:        duration,
		Rate:            rate,
		AcceptanceRatio: s.calculateAcceptanceRatio(),
	}
}

// Reset resets all statistics
func (s *Stats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.TotalRequests = 0
	s.AllowedRequests = 0
	s.DeniedRequests = 0
	s.StartTime = time.Now()
	s.LastRequestTime = time.Time{}
}

func (s *Stats) calculateAcceptanceRatio() float64 {
	if s.TotalRequests == 0 {
		return 0
	}
	return float64(s.AllowedRequests) / float64(s.TotalRequests)
}

// StatsSnapshot represents a point-in-time snapshot of statistics
type StatsSnapshot struct {
	TotalRequests   int64
	AllowedRequests int64
	DeniedRequests  int64
	StartTime       time.Time
	LastRequestTime time.Time
	Duration        time.Duration
	Rate            float64
	AcceptanceRatio float64
}

// Collector interface for collecting rate limiter statistics
type Collector interface {
	RecordAllowed()
	RecordDenied()
	GetSnapshot() StatsSnapshot
	Reset()
}

// RateLimiterWithStats wraps a rate limiter with statistics collection
type RateLimiterWithStats struct {
	limiter   RateLimiter
	stats     *Stats
}

// RateLimiter interface that the main rate limiter should implement
type RateLimiter interface {
	Allow() bool
	Wait()
}

// NewRateLimiterWithStats creates a new rate limiter with statistics
func NewRateLimiterWithStats(limiter RateLimiter) *RateLimiterWithStats {
	return &RateLimiterWithStats{
		limiter: limiter,
		stats:   NewStats(),
	}
}

// Allow checks if a request can be processed and records statistics
func (r *RateLimiterWithStats) Allow() bool {
	allowed := r.limiter.Allow()
	if allowed {
		r.stats.RecordAllowed()
	} else {
		r.stats.RecordDenied()
	}
	return allowed
}

// Wait blocks until a token is available and records statistics
func (r *RateLimiterWithStats) Wait() {
	r.limiter.Wait()
	r.stats.RecordAllowed()
}

// GetStats returns the statistics collector
func (r *RateLimiterWithStats) GetStats() Collector {
	return r.stats
}