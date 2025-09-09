package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
)

// CSRFMiddleware provides CSRF protection functionality
type CSRFMiddleware struct {
	store sessions.Store
}

// NewCSRFMiddleware creates a new CSRF middleware
func NewCSRFMiddleware(store sessions.Store) *CSRFMiddleware {
	return &CSRFMiddleware{
		store: store,
	}
}

// CSRFProtection middleware provides CSRF protection for state-changing requests
func (m *CSRFMiddleware) CSRFProtection(next http.Handler) http.Handler {
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
			// Generate new CSRF token if not present
			sessionToken = GenerateCSRFToken()
			session.Values["csrf_token"] = sessionToken
			session.Save(r, w)
		}

		// Get CSRF token from request
		requestToken := r.Header.Get("X-CSRF-Token")
		if requestToken == "" {
			requestToken = r.FormValue("csrf_token")
		}

		// Debug logging for CSRF validation
		fmt.Printf("CSRF validation - Session token: %s, Request token: %s, Match: %t\n", 
			sessionToken, requestToken, requestToken == sessionToken)

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

// EnsureCSRFToken middleware ensures a CSRF token is present in the session and context
func (m *CSRFMiddleware) EnsureCSRFToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := m.store.Get(r, "session")
		if err == nil {
			// Generate CSRF token if not present
			token, ok := session.Values["csrf_token"].(string)
			if !ok || token == "" {
				token = GenerateCSRFToken()
				session.Values["csrf_token"] = token
				session.Save(r, w)
				fmt.Printf("Generated new CSRF token: %s\n", token)
			} else {
				fmt.Printf("Using existing CSRF token: %s\n", token)
			}
			
			// Add CSRF token to request context for templates
			ctx := r.Context()
			ctx = context.WithValue(ctx, "csrf_token", token)
			r = r.WithContext(ctx)
		} else {
			fmt.Printf("Failed to get session for CSRF token: %v\n", err)
		}

		next.ServeHTTP(w, r)
	})
}