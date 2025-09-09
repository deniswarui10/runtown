package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
)

func TestCSRFProtection_GET(t *testing.T) {
	// Create in-memory session store
	store := sessions.NewCookieStore([]byte("test-secret-key"))
	
	// Create CSRF middleware
	csrfMiddleware := NewCSRFMiddleware(store)
	
	// Create test handler
	handler := csrfMiddleware.CSRFProtection(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	
	// Test GET request (should pass through)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("GET request should pass through, got status %d", w.Code)
	}
}

func TestCSRFProtection_POST_NoToken(t *testing.T) {
	// Create in-memory session store
	store := sessions.NewCookieStore([]byte("test-secret-key"))
	
	// Create CSRF middleware
	csrfMiddleware := NewCSRFMiddleware(store)
	
	// Create test handler
	handler := csrfMiddleware.CSRFProtection(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	
	// Test POST request without CSRF token (should be blocked)
	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusForbidden {
		t.Errorf("POST request without CSRF token should be blocked, got status %d", w.Code)
	}
}

func TestCSRFProtection_HTMX(t *testing.T) {
	// Create in-memory session store
	store := sessions.NewCookieStore([]byte("test-secret-key"))
	
	// Create CSRF middleware
	csrfMiddleware := NewCSRFMiddleware(store)
	
	// Create test handler
	handler := csrfMiddleware.CSRFProtection(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	
	// Test POST request with HTMX header but no CSRF token
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusForbidden {
		t.Errorf("HTMX POST request without CSRF token should be blocked, got status %d", w.Code)
	}
	
	// Check that HTML error message is returned
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Expected HTML error message for HTMX request")
	}
}

func TestEnsureCSRFToken(t *testing.T) {
	// Create in-memory session store
	store := sessions.NewCookieStore([]byte("test-secret-key"))
	
	// Create CSRF middleware
	csrfMiddleware := NewCSRFMiddleware(store)
	
	// Create test handler that checks for CSRF token
	handler := csrfMiddleware.EnsureCSRFToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	
	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Request should succeed, got status %d", w.Code)
	}
}