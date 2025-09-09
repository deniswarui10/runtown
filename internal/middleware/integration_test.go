package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionManagementIntegration(t *testing.T) {
	// Test rate limiter functionality
	rl := NewRateLimiter(2, time.Minute)
	
	// Test that rate limiter allows initial requests
	if !rl.IsAllowed("192.168.1.1") {
		t.Error("First request should be allowed")
	}
	
	if !rl.IsAllowed("192.168.1.1") {
		t.Error("Second request should be allowed")
	}
	
	// Third request should be blocked
	if rl.IsAllowed("192.168.1.1") {
		t.Error("Third request should be blocked")
	}
	
	// Different IP should still work
	if !rl.IsAllowed("192.168.1.2") {
		t.Error("Different IP should be allowed")
	}
}

func TestLoginRateLimiterIntegration(t *testing.T) {
	// Test login rate limiter
	rl := NewLoginRateLimiter(2, time.Minute, 5*time.Minute)
	
	ip := "192.168.1.1"
	
	// First two attempts should be allowed
	if !rl.IsAllowed(ip) {
		t.Error("First attempt should be allowed")
	}
	rl.RecordAttempt(ip)
	
	if !rl.IsAllowed(ip) {
		t.Error("Second attempt should be allowed")
	}
	rl.RecordAttempt(ip)
	
	// Third attempt should be blocked
	if rl.IsAllowed(ip) {
		t.Error("Third attempt should be blocked")
	}
	
	// Check time until allowed
	timeUntil := rl.GetTimeUntilAllowed(ip)
	if timeUntil <= 0 {
		t.Error("Should have time until allowed")
	}
}

func TestSecureHeadersIntegration(t *testing.T) {
	handler := SecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	// Check that security headers are set
	headers := []string{
		"X-Frame-Options",
		"X-Content-Type-Options", 
		"X-XSS-Protection",
		"Referrer-Policy",
		"Content-Security-Policy",
	}
	
	for _, header := range headers {
		if w.Header().Get(header) == "" {
			t.Errorf("Security header %s should be set", header)
		}
	}
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}