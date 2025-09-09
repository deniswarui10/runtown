package middleware

import (
	"context"
	"net/http"

	"event-ticketing-platform/internal/auth"
	"event-ticketing-platform/internal/models"
)

// AuthbossCompatMiddleware provides compatibility with the existing middleware interface
type AuthbossCompatMiddleware struct {
	authboss *auth.AuthbossConfig
}

// NewAuthbossCompatMiddleware creates a new compatibility middleware
func NewAuthbossCompatMiddleware(authbossConfig *auth.AuthbossConfig) *AuthbossCompatMiddleware {
	return &AuthbossCompatMiddleware{
		authboss: authbossConfig,
	}
}

// LoadUser loads the current user and adds it to context in the expected format
func (m *AuthbossCompatMiddleware) LoadUser(next http.Handler) http.Handler {
	// Create a wrapper that converts Authboss user to models.User
	wrappedNext := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get the user and convert to models.User
		authUser := m.authboss.LoadUser(r)
		if authUser != nil {
			// Convert AuthbossUser to models.User for compatibility
			user := &models.User{
				ID:        int(authUser.ID),
				Email:     authUser.Email,
				FirstName: authUser.FirstName,
				LastName:  authUser.LastName,
				Role:      authUser.Role,
				CreatedAt: authUser.CreatedAt,
				UpdatedAt: authUser.UpdatedAt,
			}
			
			// Add to context using the expected key
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			r = r.WithContext(ctx)
		}
		
		next.ServeHTTP(w, r)
	})
	
	// Use Authboss LoadClientStateMiddleware which handles the ResponseWriter wrapping
	return m.authboss.Authboss.LoadClientStateMiddleware(wrappedNext)
}

// RequireAuth ensures user is authenticated (compatible with existing interface)
func (m *AuthbossCompatMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil {
			// Redirect to login page
			if IsHTMXRequest(r) {
				// For HTMX requests, return a redirect header
				w.Header().Set("HX-Redirect", "/auth/login")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/auth/login?redirect="+r.URL.Path, http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireRole ensures user has the required role (compatible with existing interface)
func (m *AuthbossCompatMiddleware) RequireRole(role models.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				if IsHTMXRequest(r) {
					w.Header().Set("HX-Redirect", "/auth/login")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, "/auth/login?redirect="+r.URL.Path, http.StatusSeeOther)
				return
			}

			// Check if user has required role or is admin
			if user.Role != role && user.Role != models.UserRoleAdmin {
				if IsHTMXRequest(r) {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("Access denied"))
					return
				}
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireOwnership ensures user owns the resource or is admin (compatible with existing interface)
func (m *AuthbossCompatMiddleware) RequireOwnership(getOwnerID func(*http.Request) (int, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				if IsHTMXRequest(r) {
					w.Header().Set("HX-Redirect", "/auth/login")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, "/auth/login?redirect="+r.URL.Path, http.StatusSeeOther)
				return
			}

			// Admin can access everything
			if user.Role == models.UserRoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			// Get the owner ID of the resource
			ownerID, err := getOwnerID(r)
			if err != nil {
				http.Error(w, "Resource not found", http.StatusNotFound)
				return
			}

			// Check if user owns the resource
			if user.ID != ownerID {
				if IsHTMXRequest(r) {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("Access denied"))
					return
				}
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SessionCleanup provides session cleanup (Authboss handles this internally)
func (m *AuthbossCompatMiddleware) SessionCleanup(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Authboss handles session cleanup internally
		next.ServeHTTP(w, r)
	})
}