package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_IsAllowed(t *testing.T) {
	// Create rate limiter with 3 attempts per minute
	rl := NewRateLimiter(3, time.Minute)
	
	ip := "192.168.1.1"
	
	// First 3 attempts should be allowed
	for i := 0; i < 3; i++ {
		if !rl.IsAllowed(ip) {
			t.Errorf("Attempt %d should be allowed", i+1)
		}
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

func TestRateLimiter_Cleanup(t *testing.T) {
	// Create rate limiter with short window for testing
	rl := NewRateLimiter(2, 100*time.Millisecond)
	
	ip := "192.168.1.1"
	
	// Use up the limit
	rl.IsAllowed(ip)
	rl.IsAllowed(ip)
	
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

func TestGenerateCSRFToken(t *testing.T) {
	token1 := GenerateCSRFToken()
	token2 := GenerateCSRFToken()
	
	// Tokens should not be empty
	if token1 == "" || token2 == "" {
		t.Error("CSRF tokens should not be empty")
	}
	
	// Tokens should be different
	if token1 == token2 {
		t.Error("CSRF tokens should be unique")
	}
	
	// Tokens should be hex encoded (64 characters for 32 bytes)
	if len(token1) != 64 || len(token2) != 64 {
		t.Error("CSRF tokens should be 64 characters long")
	}
}

func TestRateLimitLogin(t *testing.T) {
	// Create rate limiter
	rl := NewRateLimiter(2, time.Minute)
	
	// Create middleware
	middleware := RateLimitLogin(rl)
	
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
}

func TestSecureHeaders(t *testing.T) {
	handler := SecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	expectedHeaders := map[string]string{
		"X-Frame-Options":           "DENY",
		"X-Content-Type-Options":    "nosniff",
		"X-XSS-Protection":          "1; mode=block",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Content-Security-Policy":   "default-src 'self'; script-src 'self' 'unsafe-inline' https://unpkg.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' https:;",
	}
	
	for header, expectedValue := range expectedHeaders {
		actualValue := w.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Expected header %s: %s, got: %s", header, expectedValue, actualValue)
		}
	}
}