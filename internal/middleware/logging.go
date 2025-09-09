package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log request details
		duration := time.Since(start)
		log.Printf(
			"%s %s %d %v %s %s",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
			r.RemoteAddr,
			r.UserAgent(),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// DetailedLoggingMiddleware provides more detailed logging including request body size
func DetailedLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code and response size
		wrapped := &detailedResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			size:           0,
		}

		// Get user info if available
		var userInfo string
		if user := GetUserFromContext(r.Context()); user != nil {
			userInfo = user.Email
		} else {
			userInfo = "anonymous"
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log detailed request information
		duration := time.Since(start)
		log.Printf(
			"[%s] %s %s %d %d bytes %v - User: %s - IP: %s - UA: %s",
			r.Method,
			r.URL.Path,
			r.URL.RawQuery,
			wrapped.statusCode,
			wrapped.size,
			duration,
			userInfo,
			getClientIP(r),
			r.UserAgent(),
		)
	})
}

// detailedResponseWriter wraps http.ResponseWriter to capture status code and response size
type detailedResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *detailedResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *detailedResponseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// getClientIP gets the real client IP address
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := generateRequestID()
		
		// Add request ID to response headers
		w.Header().Set("X-Request-ID", requestID)
		
		// Add request ID to context for use in handlers
		ctx := r.Context()
		ctx = context.WithValue(ctx, "request_id", requestID)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Simple implementation - in production, use a proper UUID library
	return fmt.Sprintf("%d", time.Now().UnixNano())
}