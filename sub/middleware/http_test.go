package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// mockRateLimiter is a mock implementation of RateLimiter for testing
type mockRateLimiter struct {
	allowReturn bool
	callCount   int32
}

func (m *mockRateLimiter) Allow() bool {
	atomic.AddInt32(&m.callCount, 1)
	return m.allowReturn
}

func (m *mockRateLimiter) getCallCount() int32 {
	return atomic.LoadInt32(&m.callCount)
}

func TestHTTPRateLimiter_Middleware(t *testing.T) {
	tests := []struct {
		name           string
		allowReturn    bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "request allowed",
			allowReturn:    true,
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "request denied",
			allowReturn:    false,
			expectedStatus: http.StatusTooManyRequests,
			expectedBody:   "Too Many Requests",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRateLimiter{allowReturn: tt.allowReturn}
			rl := NewHTTPRateLimiter(mock, nil)
			
			handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			}))
			
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			
			handler.ServeHTTP(rec, req)
			
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
			
			body := strings.TrimSpace(rec.Body.String())
			if !strings.Contains(body, tt.expectedBody) {
				t.Errorf("Expected body to contain %q, got %q", tt.expectedBody, body)
			}
			
			if mock.getCallCount() != 1 {
				t.Errorf("Expected Allow() to be called once, got %d", mock.getCallCount())
			}
		})
	}
}

func TestHTTPRateLimiter_MiddlewareFunc(t *testing.T) {
	mock := &mockRateLimiter{allowReturn: true}
	rl := NewHTTPRateLimiter(mock, nil)
	
	handlerCalled := false
	handler := rl.MiddlewareFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	
	handler(rec, req)
	
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestDefaultKeyFunc(t *testing.T) {
	tests := []struct {
		name        string
		headers     map[string]string
		remoteAddr  string
		expectedKey string
	}{
		{
			name:        "X-Forwarded-For",
			headers:     map[string]string{"X-Forwarded-For": "192.168.1.1"},
			remoteAddr:  "10.0.0.1:1234",
			expectedKey: "192.168.1.1",
		},
		{
			name:        "X-Real-IP",
			headers:     map[string]string{"X-Real-IP": "192.168.1.2"},
			remoteAddr:  "10.0.0.1:1234",
			expectedKey: "192.168.1.2",
		},
		{
			name:        "RemoteAddr fallback",
			headers:     map[string]string{},
			remoteAddr:  "10.0.0.1:1234",
			expectedKey: "10.0.0.1:1234",
		},
		{
			name: "X-Forwarded-For takes precedence",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
				"X-Real-IP":       "192.168.1.2",
			},
			remoteAddr:  "10.0.0.1:1234",
			expectedKey: "192.168.1.1",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			
			key := DefaultKeyFunc(req)
			if key != tt.expectedKey {
				t.Errorf("Expected key %q, got %q", tt.expectedKey, key)
			}
		})
	}
}

func TestCustomErrorHandler(t *testing.T) {
	mock := &mockRateLimiter{allowReturn: false}
	
	customHeaders := map[string]string{
		"X-RateLimit-Limit": "100",
		"Retry-After":       "60",
	}
	
	opts := &Options{
		ErrorHandler: CustomErrorHandler("Rate limit exceeded", customHeaders),
	}
	
	rl := NewHTTPRateLimiter(mock, opts)
	
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	
	handler.ServeHTTP(rec, req)
	
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}
	
	if !strings.Contains(rec.Body.String(), "Rate limit exceeded") {
		t.Errorf("Expected custom error message, got %q", rec.Body.String())
	}
	
	for k, v := range customHeaders {
		if rec.Header().Get(k) != v {
			t.Errorf("Expected header %s: %s, got %s", k, v, rec.Header().Get(k))
		}
	}
}

func TestJSONErrorHandler(t *testing.T) {
	mock := &mockRateLimiter{allowReturn: false}
	
	opts := &Options{
		ErrorHandler: JSONErrorHandler,
	}
	
	rl := NewHTTPRateLimiter(mock, opts)
	
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	
	handler.ServeHTTP(rec, req)
	
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}
	
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}
	
	expectedBody := `{"error":"too many requests","status":429}`
	if strings.TrimSpace(rec.Body.String()) != expectedBody {
		t.Errorf("Expected JSON body %q, got %q", expectedBody, rec.Body.String())
	}
}

func TestPerKeyHTTPRateLimiter(t *testing.T) {
	callCounts := make(map[string]*int32)
	var mu sync.Mutex
	
	factory := func() RateLimiter {
		mock := &mockRateLimiter{allowReturn: true}
		key := "default"
		mu.Lock()
		callCounts[key] = &mock.callCount
		mu.Unlock()
		return mock
	}
	
	keyFunc := func(r *http.Request) string {
		return r.Header.Get("X-User-ID")
	}
	
	opts := &Options{
		KeyFunc: keyFunc,
	}
	
	rl := NewPerKeyHTTPRateLimiter(factory, opts)
	
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	
	// Test different users get different limiters
	users := []string{"user1", "user2", "user1", "user3", "user2"}
	
	for _, user := range users {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", user)
		rec := httptest.NewRecorder()
		
		handler.ServeHTTP(rec, req)
		
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for user %s, got %d", user, rec.Code)
		}
	}
}

func TestKeyFuncs(t *testing.T) {
	t.Run("ByUserID", func(t *testing.T) {
		fn := KeyFuncs.ByUserID("X-User-ID")
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", "user123")
		
		key := fn(req)
		if key != "user123" {
			t.Errorf("Expected key 'user123', got %q", key)
		}
		
		req.Header.Del("X-User-ID")
		key = fn(req)
		if key != "anonymous" {
			t.Errorf("Expected key 'anonymous' for missing header, got %q", key)
		}
	})
	
	t.Run("ByAPIKey", func(t *testing.T) {
		fn := KeyFuncs.ByAPIKey("X-API-Key")
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "secret123")
		
		key := fn(req)
		if key != "secret123" {
			t.Errorf("Expected key 'secret123', got %q", key)
		}
		
		req.Header.Del("X-API-Key")
		key = fn(req)
		if key != "no-api-key" {
			t.Errorf("Expected key 'no-api-key' for missing header, got %q", key)
		}
	})
	
	t.Run("ByPath", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/123", nil)
		
		key := KeyFuncs.ByPath(req)
		if key != "/api/users/123" {
			t.Errorf("Expected key '/api/users/123', got %q", key)
		}
	})
	
	t.Run("Combination", func(t *testing.T) {
		fn := KeyFuncs.Combination(
			KeyFuncs.ByPath,
			KeyFuncs.ByUserID("X-User-ID"),
		)
		
		req := httptest.NewRequest("GET", "/api/data", nil)
		req.Header.Set("X-User-ID", "user456")
		
		key := fn(req)
		if !strings.Contains(key, "/api/data") || !strings.Contains(key, "user456") {
			t.Errorf("Expected key to contain both path and user ID, got %q", key)
		}
	})
}

func TestConcurrentRequests(t *testing.T) {
	mock := &mockRateLimiter{allowReturn: true}
	rl := NewHTTPRateLimiter(mock, nil)
	
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			
			handler.ServeHTTP(rec, req)
			
			if rec.Code != http.StatusOK {
				errors <- nil
			}
		}()
	}
	
	wg.Wait()
	close(errors)
	
	errorCount := 0
	for range errors {
		errorCount++
	}
	
	if errorCount > 0 {
		t.Errorf("Expected no errors in concurrent requests, got %d", errorCount)
	}
	
	if mock.getCallCount() != 100 {
		t.Errorf("Expected 100 calls to Allow(), got %d", mock.getCallCount())
	}
}