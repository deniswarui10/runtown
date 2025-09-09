package middleware

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestLoggingMiddleware(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Create request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "test-agent")

	rr := httptest.NewRecorder()

	// Execute middleware
	LoggingMiddleware(handler).ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "test response", rr.Body.String())

	// Check log output
	logOutput := buf.String()
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "/test")
	assert.Contains(t, logOutput, "200")
	assert.Contains(t, logOutput, "127.0.0.1:12345")
	assert.Contains(t, logOutput, "test-agent")
}

func TestDetailedLoggingMiddleware(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})

	// Create request with user context
	req := httptest.NewRequest("POST", "/test?param=value", nil)
	req.RemoteAddr = "192.168.1.1:54321"
	req.Header.Set("User-Agent", "detailed-test-agent")

	user := &models.User{
		ID:    1,
		Email: "test@example.com",
	}
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Execute middleware
	DetailedLoggingMiddleware(handler).ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, "created", rr.Body.String())

	// Check log output
	logOutput := buf.String()
	assert.Contains(t, logOutput, "[POST]")
	assert.Contains(t, logOutput, "/test")
	assert.Contains(t, logOutput, "param=value")
	assert.Contains(t, logOutput, "201")
	assert.Contains(t, logOutput, "7 bytes") // "created" is 7 bytes
	assert.Contains(t, logOutput, "test@example.com")
	assert.Contains(t, logOutput, "192.168.1.1:54321")
	assert.Contains(t, logOutput, "detailed-test-agent")
}

func TestDetailedLoggingMiddleware_AnonymousUser(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create request without user context
	req := httptest.NewRequest("GET", "/public", nil)
	rr := httptest.NewRecorder()

	// Execute middleware
	DetailedLoggingMiddleware(handler).ServeHTTP(rr, req)

	// Check log output
	logOutput := buf.String()
	assert.Contains(t, logOutput, "anonymous")
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rw := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		statusCode:     http.StatusOK,
	}

	rw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, rw.statusCode)
}

func TestDetailedResponseWriter_Write(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &detailedResponseWriter{
		ResponseWriter: recorder,
		statusCode:     http.StatusOK,
		size:           0,
	}

	data := []byte("test data")
	n, err := rw.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, len(data), rw.size)
	assert.Equal(t, "test data", recorder.Body.String())
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remoteAddr string
		expected string
	}{
		{
			name:       "X-Forwarded-For header",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "203.0.113.1",
		},
		{
			name:       "X-Real-IP header",
			headers:    map[string]string{"X-Real-IP": "203.0.113.2"},
			remoteAddr: "192.168.1.1:12345",
			expected:   "203.0.113.2",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1:12345",
		},
		{
			name: "X-Forwarded-For takes precedence",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1",
				"X-Real-IP":       "203.0.113.2",
			},
			remoteAddr: "192.168.1.1:12345",
			expected:   "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			result := getClientIP(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	// Create test handler that captures request ID
	var capturedRequestID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = r.Context().Value("request_id").(string)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Execute middleware
	RequestIDMiddleware(handler).ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NotEmpty(t, capturedRequestID)
	assert.Equal(t, capturedRequestID, rr.Header().Get("X-Request-ID"))
}

func TestGenerateRequestID(t *testing.T) {
	// Generate multiple request IDs
	id1 := generateRequestID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	id2 := generateRequestID()

	// Assertions
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	
	// Should be numeric (timestamp)
	assert.True(t, strings.ContainsAny(id1, "0123456789"))
}