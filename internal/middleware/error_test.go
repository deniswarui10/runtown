package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorHandlingMiddleware_NoPanic(t *testing.T) {
	// Create test handler that doesn't panic
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Execute middleware
	ErrorHandlingMiddleware(handler).ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "success", rr.Body.String())
}

func TestErrorHandlingMiddleware_PanicRegularRequest(t *testing.T) {
	// Create test handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Execute middleware
	ErrorHandlingMiddleware(handler).ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "Internal Server Error\n", rr.Body.String())
}

func TestErrorHandlingMiddleware_PanicHTMXRequest(t *testing.T) {
	// Create test handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	// Execute middleware
	ErrorHandlingMiddleware(handler).ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "text/html", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), "Something went wrong")
	assert.Contains(t, rr.Body.String(), "bg-red-50")
}

func TestNotFoundHandler_RegularRequest(t *testing.T) {
	handler := NotFoundHandler()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "<!DOCTYPE html>")
	assert.Contains(t, rr.Body.String(), "404")
	assert.Contains(t, rr.Body.String(), "Page Not Found")
}

func TestNotFoundHandler_HTMXRequest(t *testing.T) {
	handler := NotFoundHandler()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "text/html", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), "Page Not Found")
	assert.Contains(t, rr.Body.String(), "text-center")
	assert.NotContains(t, rr.Body.String(), "<!DOCTYPE html>")
}

func TestMethodNotAllowedHandler_RegularRequest(t *testing.T) {
	handler := MethodNotAllowedHandler()

	req := httptest.NewRequest("PATCH", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", rr.Header().Get("Allow"))
	assert.Equal(t, "Method Not Allowed\n", rr.Body.String())
}

func TestMethodNotAllowedHandler_HTMXRequest(t *testing.T) {
	handler := MethodNotAllowedHandler()

	req := httptest.NewRequest("PATCH", "/test", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", rr.Header().Get("Allow"))
	assert.Equal(t, "text/html", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), "Method not allowed")
	assert.Contains(t, rr.Body.String(), "bg-yellow-50")
}

func TestHandleValidationErrors(t *testing.T) {
	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Execute middleware
	HandleValidationErrors(handler).ServeHTTP(rr, req)

	// Assertions - this middleware currently just passes through
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "success", rr.Body.String())
}

// Test helper to verify panic recovery doesn't interfere with normal operation
func TestErrorHandlingMiddleware_MultipleRequests(t *testing.T) {
	normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("normal"))
	})

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	middleware := ErrorHandlingMiddleware

	// Test normal request
	req1 := httptest.NewRequest("GET", "/normal", nil)
	rr1 := httptest.NewRecorder()
	middleware(normalHandler).ServeHTTP(rr1, req1)

	assert.Equal(t, http.StatusOK, rr1.Code)
	assert.Equal(t, "normal", rr1.Body.String())

	// Test panic request
	req2 := httptest.NewRequest("GET", "/panic", nil)
	rr2 := httptest.NewRecorder()
	middleware(panicHandler).ServeHTTP(rr2, req2)

	assert.Equal(t, http.StatusInternalServerError, rr2.Code)

	// Test normal request again to ensure middleware still works
	req3 := httptest.NewRequest("GET", "/normal", nil)
	rr3 := httptest.NewRecorder()
	middleware(normalHandler).ServeHTTP(rr3, req3)

	assert.Equal(t, http.StatusOK, rr3.Code)
	assert.Equal(t, "normal", rr3.Body.String())
}