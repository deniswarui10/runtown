package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
			"HX-Request",
			"HX-Trigger",
			"HX-Target",
			"HX-Current-URL",
		},
		ExposedHeaders: []string{
			"HX-Location",
			"HX-Push-Url",
			"HX-Redirect",
			"HX-Refresh",
			"HX-Replace-Url",
		},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// CORSMiddleware creates a CORS middleware with the given configuration
func CORSMiddleware(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			// Check if origin is allowed
			if isOriginAllowed(origin, config.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			
			// Set other CORS headers
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			
			if len(config.AllowedMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			}
			
			if len(config.AllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			}
			
			if len(config.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
			}
			
			if config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
			}
			
			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if the origin is in the allowed list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}
	
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		// Support wildcard subdomains (e.g., *.example.com)
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	
	return false
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", 
			"default-src 'self'; "+
			"script-src 'self' 'unsafe-inline' https://unpkg.com; "+
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
			"font-src 'self' https://fonts.gstatic.com; "+
			"img-src 'self' data: https:; "+
			"connect-src 'self';")
		
		// HSTS header for HTTPS
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		
		next.ServeHTTP(w, r)
	})
}