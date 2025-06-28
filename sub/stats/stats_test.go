package stats

import (
	"sync"
	"testing"
	"time"
)

// mockRateLimiter is a mock implementation of RateLimiter for testing
type mockRateLimiter struct {
	allowReturn bool
	allowCount  int
	waitCount   int
	mu          sync.Mutex
}

func (m *mockRateLimiter) Allow() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowCount++
	return m.allowReturn
}

func (m *mockRateLimiter) Wait() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.waitCount++
}

func TestNewStats(t *testing.T) {
	stats := NewStats()
	
	if stats.TotalRequests != 0 {
		t.Errorf("Expected TotalRequests to be 0, got %d", stats.TotalRequests)
	}
	if stats.AllowedRequests != 0 {
		t.Errorf("Expected AllowedRequests to be 0, got %d", stats.AllowedRequests)
	}
	if stats.DeniedRequests != 0 {
		t.Errorf("Expected DeniedRequests to be 0, got %d", stats.DeniedRequests)
	}
	if stats.StartTime.IsZero() {
		t.Error("Expected StartTime to be set")
	}
}

func TestRecordAllowed(t *testing.T) {
	stats := NewStats()
	
	stats.RecordAllowed()
	stats.RecordAllowed()
	
	if stats.TotalRequests != 2 {
		t.Errorf("Expected TotalRequests to be 2, got %d", stats.TotalRequests)
	}
	if stats.AllowedRequests != 2 {
		t.Errorf("Expected AllowedRequests to be 2, got %d", stats.AllowedRequests)
	}
	if stats.DeniedRequests != 0 {
		t.Errorf("Expected DeniedRequests to be 0, got %d", stats.DeniedRequests)
	}
	if stats.LastRequestTime.IsZero() {
		t.Error("Expected LastRequestTime to be set")
	}
}

func TestRecordDenied(t *testing.T) {
	stats := NewStats()
	
	stats.RecordDenied()
	stats.RecordDenied()
	stats.RecordDenied()
	
	if stats.TotalRequests != 3 {
		t.Errorf("Expected TotalRequests to be 3, got %d", stats.TotalRequests)
	}
	if stats.AllowedRequests != 0 {
		t.Errorf("Expected AllowedRequests to be 0, got %d", stats.AllowedRequests)
	}
	if stats.DeniedRequests != 3 {
		t.Errorf("Expected DeniedRequests to be 3, got %d", stats.DeniedRequests)
	}
}

func TestGetSnapshot(t *testing.T) {
	stats := NewStats()
	
	// Record some requests
	stats.RecordAllowed()
	stats.RecordAllowed()
	stats.RecordDenied()
	
	// Small delay to ensure duration > 0
	time.Sleep(10 * time.Millisecond)
	
	snapshot := stats.GetSnapshot()
	
	if snapshot.TotalRequests != 3 {
		t.Errorf("Expected TotalRequests to be 3, got %d", snapshot.TotalRequests)
	}
	if snapshot.AllowedRequests != 2 {
		t.Errorf("Expected AllowedRequests to be 2, got %d", snapshot.AllowedRequests)
	}
	if snapshot.DeniedRequests != 1 {
		t.Errorf("Expected DeniedRequests to be 1, got %d", snapshot.DeniedRequests)
	}
	if snapshot.AcceptanceRatio != float64(2)/float64(3) {
		t.Errorf("Expected AcceptanceRatio to be 0.666..., got %f", snapshot.AcceptanceRatio)
	}
	if snapshot.Duration <= 0 {
		t.Error("Expected Duration to be positive")
	}
	if snapshot.Rate <= 0 {
		t.Error("Expected Rate to be positive")
	}
}

func TestReset(t *testing.T) {
	stats := NewStats()
	
	// Record some requests
	stats.RecordAllowed()
	stats.RecordDenied()
	
	// Reset
	stats.Reset()
	
	if stats.TotalRequests != 0 {
		t.Errorf("Expected TotalRequests to be 0 after reset, got %d", stats.TotalRequests)
	}
	if stats.AllowedRequests != 0 {
		t.Errorf("Expected AllowedRequests to be 0 after reset, got %d", stats.AllowedRequests)
	}
	if stats.DeniedRequests != 0 {
		t.Errorf("Expected DeniedRequests to be 0 after reset, got %d", stats.DeniedRequests)
	}
	if stats.LastRequestTime.IsZero() == false {
		t.Error("Expected LastRequestTime to be zero after reset")
	}
}

func TestRateLimiterWithStats(t *testing.T) {
	mock := &mockRateLimiter{allowReturn: true}
	rlWithStats := NewRateLimiterWithStats(mock)
	
	// Test Allow() when allowed
	if !rlWithStats.Allow() {
		t.Error("Expected Allow() to return true")
	}
	
	stats := rlWithStats.GetStats().GetSnapshot()
	if stats.AllowedRequests != 1 {
		t.Errorf("Expected AllowedRequests to be 1, got %d", stats.AllowedRequests)
	}
	if stats.DeniedRequests != 0 {
		t.Errorf("Expected DeniedRequests to be 0, got %d", stats.DeniedRequests)
	}
	
	// Test Allow() when denied
	mock.allowReturn = false
	if rlWithStats.Allow() {
		t.Error("Expected Allow() to return false")
	}
	
	stats = rlWithStats.GetStats().GetSnapshot()
	if stats.AllowedRequests != 1 {
		t.Errorf("Expected AllowedRequests to be 1, got %d", stats.AllowedRequests)
	}
	if stats.DeniedRequests != 1 {
		t.Errorf("Expected DeniedRequests to be 1, got %d", stats.DeniedRequests)
	}
	
	// Test Wait()
	rlWithStats.Wait()
	stats = rlWithStats.GetStats().GetSnapshot()
	if stats.AllowedRequests != 2 {
		t.Errorf("Expected AllowedRequests to be 2 after Wait(), got %d", stats.AllowedRequests)
	}
}

func TestConcurrentAccess(t *testing.T) {
	stats := NewStats()
	
	// Run concurrent operations
	var wg sync.WaitGroup
	workers := 10
	requestsPerWorker := 100
	
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < requestsPerWorker; j++ {
				if j%3 == 0 {
					stats.RecordDenied()
				} else {
					stats.RecordAllowed()
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	snapshot := stats.GetSnapshot()
	expectedTotal := int64(workers * requestsPerWorker)
	if snapshot.TotalRequests != expectedTotal {
		t.Errorf("Expected TotalRequests to be %d, got %d", expectedTotal, snapshot.TotalRequests)
	}
}

func TestAcceptanceRatioEdgeCases(t *testing.T) {
	stats := NewStats()
	
	// Test with no requests
	snapshot := stats.GetSnapshot()
	if snapshot.AcceptanceRatio != 0 {
		t.Errorf("Expected AcceptanceRatio to be 0 with no requests, got %f", snapshot.AcceptanceRatio)
	}
	
	// Test with only allowed requests
	stats.RecordAllowed()
	stats.RecordAllowed()
	snapshot = stats.GetSnapshot()
	if snapshot.AcceptanceRatio != 1.0 {
		t.Errorf("Expected AcceptanceRatio to be 1.0 with only allowed requests, got %f", snapshot.AcceptanceRatio)
	}
	
	// Test with only denied requests
	stats.Reset()
	stats.RecordDenied()
	stats.RecordDenied()
	snapshot = stats.GetSnapshot()
	if snapshot.AcceptanceRatio != 0 {
		t.Errorf("Expected AcceptanceRatio to be 0 with only denied requests, got %f", snapshot.AcceptanceRatio)
	}
}