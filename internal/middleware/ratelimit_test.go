package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginRateLimiter_IsAllowed(t *testing.T) {
	// Create rate limiter with 3 attempts per minute, 5 minute block
	rl := NewLoginRateLimiter(3, time.Minute, 5*time.Minute)
	
	ip := "192.168.1.1"
	
	// First 3 attempts should be allowed
	for i := 0; i < 3; i++ {
		if !rl.IsAllowed(ip) {
			t.Errorf("Attempt %d should be allowed", i+1)
		}
		rl.RecordAttempt(ip) // Record the attempt
	}
	
	// 4th attempt should be blocked
	if rl.IsAllowed(ip) {
		t.Error("4th attempt should be blocked")
	}
	
	// Different IP should still be allowed
	if !rl.IsAllowed("192.168.1.2") {
		t.Error("Different IP should be allowed")
	}
}

func TestLoginRateLimiter_RecordAttempt(t *testing.T) {
	rl := NewLoginRateLimiter(2, time.Minute, 5*time.Minute)
	
	ip := "192.168.1.1"
	
	// Record attempts
	rl.RecordAttempt(ip)
	rl.RecordAttempt(ip)
	
	// Should be blocked after recording attempts
	if rl.IsAllowed(ip) {
		t.Error("Should be blocked after recording max attempts")
	}
}

func TestLoginRateLimiter_GetTimeUntilAllowed(t *testing.T) {
	rl := NewLoginRateLimiter(1, time.Minute, 5*time.Minute)
	
	ip := "192.168.1.1"
	
	// Initially should be 0
	if duration := rl.GetTimeUntilAllowed(ip); duration != 0 {
		t.Errorf("Expected 0 duration, got %v", duration)
	}
	
	// Record attempt
	rl.RecordAttempt(ip)
	
	// Should have time until allowed
	duration := rl.GetTimeUntilAllowed(ip)
	if duration <= 0 || duration > time.Minute {
		t.Errorf("Expected duration between 0 and 1 minute, got %v", duration)
	}
}

func TestLoginRateLimiter_Cleanup(t *testing.T) {
	// Create rate limiter with short window for testing
	rl := NewLoginRateLimiter(1, 100*time.Millisecond, time.Minute)
	
	ip := "192.168.1.1"
	
	// Record attempt
	rl.RecordAttempt(ip)
	
	// Should be blocked
	if rl.IsAllowed(ip) {
		t.Error("Should be blocked")
	}
	
	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)
	
	// Should be allowed again
	if !rl.IsAllowed(ip) {
		t.Error("Should be allowed after window expires")
	}
}

func TestLoginRateLimit_Middleware(t *testing.T) {
	// Create rate limiter
	rl := NewLoginRateLimiter(2, time.Minute, 5*time.Minute)
	
	// Create middleware
	middleware := LoginRateLimit(rl)
	
	// Create test handler
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	
	// Test GET request (should pass through)
	req := httptest.NewRequest("GET", "/auth/login", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("GET request should pass through, got status %d", w.Code)
	}
	
	// Test POST requests (should be rate limited)
	for i := 0; i < 2; i++ {
		req = httptest.NewRequest("POST", "/auth/login", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("POST request %d should be allowed, got status %d", i+1, w.Code)
		}
	}
	
	// Third POST request should be blocked
	req = httptest.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Third POST request should be blocked, got status %d", w.Code)
	}
	
	// Check that error message is returned
	body := w.Body.String()
	if body == "" {
		t.Error("Expected error message in response body")
	}
}

func TestLoginRateLimit_HTMX(t *testing.T) {
	// Create rate limiter with 1 attempt for easy testing
	rl := NewLoginRateLimiter(1, time.Minute, 5*time.Minute)
	
	// Create middleware
	middleware := LoginRateLimit(rl)
	
	// Create test handler
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	
	// First request should succeed
	req := httptest.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("First POST request should be allowed, got status %d", w.Code)
	}
	
	// Second request with HTMX header should be blocked with HTML response
	req = httptest.NewRequest("POST", "/auth/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("HX-Request", "true")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Second POST request should be blocked, got status %d", w.Code)
	}
	
	// Check that HTML error message is returned for HTMX
	body := w.Body.String()
	if !contains(body, "Too many login attempts") {
		t.Error("Expected HTML error message for HTMX request")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsHelper(s, substr))))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}