package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/sessions"
)

// SessionMiddleware provides session management functionality
type SessionMiddleware struct {
	store sessions.Store
}

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	attempts map[string][]time.Time
	mutex    sync.RWMutex
	maxAttempts int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxAttempts int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string][]time.Time),
		maxAttempts: maxAttempts,
		window:   window,
	}
	
	// Start cleanup goroutine
	go rl.cleanup()
	
	return rl
}

// IsAllowed checks if a request from the given IP is allowed
func (rl *RateLimiter) IsAllowed(ip string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-rl.window)
	
	// Get attempts for this IP
	attempts := rl.attempts[ip]
	
	// Remove old attempts
	var validAttempts []time.Time
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			validAttempts = append(validAttempts, attempt)
		}
	}
	
	// Check if under limit
	if len(validAttempts) >= rl.maxAttempts {
		return false
	}
	
	// Add current attempt
	validAttempts = append(validAttempts, now)
	rl.attempts[ip] = validAttempts
	
	return true
}

// cleanup removes old entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window)
		
		for ip, attempts := range rl.attempts {
			var validAttempts []time.Time
			for _, attempt := range attempts {
				if attempt.After(cutoff) {
					validAttempts = append(validAttempts, attempt)
				}
			}
			
			if len(validAttempts) == 0 {
				delete(rl.attempts, ip)
			} else {
				rl.attempts[ip] = validAttempts
			}
		}
		rl.mutex.Unlock()
	}
}

// NewSessionMiddleware creates a new session middleware
func NewSessionMiddleware(store sessions.Store) *SessionMiddleware {
	return &SessionMiddleware{
		store: store,
	}
}

// SessionConfig configures session middleware
func (m *SessionMiddleware) SessionConfig(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set secure session headers
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		
		// Add CSRF protection headers for non-GET requests
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
			// Skip CSRF protection for logout endpoint
			if r.URL.Path == "/auth/logout" {
				// Allow logout without CSRF token for now
			} else {
				// Basic CSRF protection - in production, use a proper CSRF library
				session, err := m.store.Get(r, "session")
				if err == nil {
					if token, ok := session.Values["csrf_token"].(string); ok {
						if r.Header.Get("X-CSRF-Token") != token && r.FormValue("csrf_token") != token {
							http.Error(w, "CSRF token mismatch", http.StatusForbidden)
							return
						}
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// CleanupExpiredSessions middleware periodically cleans up expired sessions
func (m *SessionMiddleware) CleanupExpiredSessions(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This would typically be handled by a background job
		// For now, we'll just ensure the current session is valid
		session, err := m.store.Get(r, "session")
		if err != nil {
			// Clear invalid session
			session.Options.MaxAge = -1
			session.Save(r, w)
		} else {
			// Check if session has expired
			if expiry, ok := session.Values["expires_at"].(time.Time); ok {
				if time.Now().After(expiry) {
					// Session expired, clear it
					session.Options.MaxAge = -1
					session.Save(r, w)
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// SetSessionTimeout sets session timeout
func (m *SessionMiddleware) SetSessionTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := m.store.Get(r, "session")
			if err == nil {
				// Update session expiry
				session.Values["expires_at"] = time.Now().Add(timeout)
				session.Save(r, w)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireCSRF middleware ensures CSRF token is present for state-changing requests
func (m *SessionMiddleware) RequireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF check for safe methods
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		session, err := m.store.Get(r, "session")
		if err != nil {
			http.Error(w, "Session error", http.StatusInternalServerError)
			return
		}

		// Get CSRF token from session
		sessionToken, ok := session.Values["csrf_token"].(string)
		if !ok || sessionToken == "" {
			http.Error(w, "CSRF token not found in session", http.StatusForbidden)
			return
		}

		// Get CSRF token from request
		requestToken := r.Header.Get("X-CSRF-Token")
		if requestToken == "" {
			requestToken = r.FormValue("csrf_token")
		}

		// Validate CSRF token
		if requestToken != sessionToken {
			if IsHTMXRequest(r) {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`
					<div class="bg-red-50 border border-red-200 text-red-800 p-4 rounded-lg">
						<div class="flex">
							<div class="flex-shrink-0">
								<svg class="h-5 w-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
								</svg>
							</div>
							<div class="ml-3">
								<p class="text-sm">Security token mismatch. Please refresh the page and try again.</p>
							</div>
						</div>
					</div>
				`))
			} else {
				http.Error(w, "CSRF token mismatch", http.StatusForbidden)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}



// RateLimitLogin provides rate limiting for login attempts
func RateLimitLogin(rateLimiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply rate limiting to POST requests (login attempts)
			if r.Method != "POST" {
				next.ServeHTTP(w, r)
				return
			}
			
			// Get client IP
			ip := getClientIP(r)
			
			// Check rate limit
			if !rateLimiter.IsAllowed(ip) {
				if IsHTMXRequest(r) {
					w.WriteHeader(http.StatusTooManyRequests)
					w.Write([]byte(`
						<div class="bg-red-50 border border-red-200 text-red-800 p-4 rounded-lg">
							<div class="flex">
								<div class="flex-shrink-0">
									<svg class="h-5 w-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
									</svg>
								</div>
								<div class="ml-3">
									<p class="text-sm">Too many login attempts. Please try again later.</p>
								</div>
							</div>
						</div>
					`))
				} else {
					http.Error(w, "Too many login attempts. Please try again later.", http.StatusTooManyRequests)
				}
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}



// ExtendSession extends the session duration for "Remember Me" functionality
func (m *SessionMiddleware) ExtendSession(duration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := m.store.Get(r, "session")
			if err == nil {
				// Check if "remember me" is enabled
				if rememberMe, ok := session.Values["remember_me"].(bool); ok && rememberMe {
					// Extend session expiry
					session.Values["expires_at"] = time.Now().Add(duration)
					session.Options.MaxAge = int(duration.Seconds())
					session.Save(r, w)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecureHeaders adds security headers to responses
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://unpkg.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' https:;")
		
		// Only set HSTS for HTTPS
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		
		next.ServeHTTP(w, r)
	})
}

