package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"event-ticketing-platform/internal/auth"
	"event-ticketing-platform/internal/models"

	"github.com/aarondl/authboss/v3"
	"github.com/gorilla/sessions"
)

// AuthbossMiddleware provides Authboss-based authentication functionality
type AuthbossMiddleware struct {
	authboss *auth.AuthbossConfig
}

// NewAuthbossMiddleware creates a new Authboss middleware
func NewAuthbossMiddleware(authbossConfig *auth.AuthbossConfig) *AuthbossMiddleware {
	return &AuthbossMiddleware{
		authboss: authbossConfig,
	}
}

// LoadUser middleware loads the current user using Authboss
func (m *AuthbossMiddleware) LoadUser(next http.Handler) http.Handler {
	// Use Authboss's built-in LoadClientStateMiddleware to properly load session state
	return m.authboss.Authboss.LoadClientStateMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[DEBUG] LoadUser middleware called for: %s\n", r.URL.Path)

		// Now check if user was loaded and set in context
		user, err := m.authboss.Authboss.CurrentUser(r)
		if err != nil {
			fmt.Printf("[DEBUG] Error loading current user: %v\n", err)
		} else if user != nil {
			if authUser, ok := user.(*auth.AuthbossUser); ok {
				fmt.Printf("[DEBUG] User loaded successfully: ID=%d, Email=%s\n", authUser.ID, authUser.Email)

				// Convert to models.User and set in context for compatibility with existing handlers
				modelsUser := &models.User{
					ID:        int(authUser.ID),
					Email:     authUser.Email,
					FirstName: authUser.FirstName,
					LastName:  authUser.LastName,
					Role:      authUser.Role,
					CreatedAt: authUser.CreatedAt,
					UpdatedAt: authUser.UpdatedAt,
				}

				// Set user in context using the old context key for compatibility
				ctx := context.WithValue(r.Context(), UserContextKey, modelsUser)
				r = r.WithContext(ctx)
				fmt.Printf("[DEBUG] User set in context: ID=%d, Email=%s\n", modelsUser.ID, modelsUser.Email)
			}
		} else {
			fmt.Printf("[DEBUG] No user found in session\n")
		}

		// Continue with the next handler
		next.ServeHTTP(w, r)
	}))
}

// RequireAuth middleware ensures user is authenticated using Authboss
func (m *AuthbossMiddleware) RequireAuth(next http.Handler) http.Handler {
	// Use Authboss's built-in LoadClientStateMiddleware to properly load session state first
	return m.authboss.Authboss.LoadClientStateMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[DEBUG] RequireAuth middleware called for: %s\n", r.URL.Path)

		user, err := m.authboss.Authboss.CurrentUser(r)
		if err != nil {
			fmt.Printf("[DEBUG] RequireAuth - Error getting current user: %v\n", err)
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		} else if user == nil {
			fmt.Printf("[DEBUG] RequireAuth - No authenticated user found, redirecting to login\n")
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		if authUser, ok := user.(*auth.AuthbossUser); ok {
			fmt.Printf("[DEBUG] RequireAuth - User authenticated: ID=%d, Email=%s\n", authUser.ID, authUser.Email)

			// Convert to models.User and set in context for compatibility with existing handlers
			modelsUser := &models.User{
				ID:        int(authUser.ID),
				Email:     authUser.Email,
				FirstName: authUser.FirstName,
				LastName:  authUser.LastName,
				Role:      authUser.Role,
				CreatedAt: authUser.CreatedAt,
				UpdatedAt: authUser.UpdatedAt,
			}

			// Set user in context using the old context key for compatibility
			ctx := context.WithValue(r.Context(), "user", modelsUser)
			r = r.WithContext(ctx)
			fmt.Printf("[DEBUG] RequireAuth - User set in context: ID=%d, Email=%s\n", modelsUser.ID, modelsUser.Email)
		}

		// Continue with the actual handler
		next.ServeHTTP(w, r)
	}))
}

// RequireRole middleware ensures user has the required role
func (m *AuthbossMiddleware) RequireRole(role models.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := m.authboss.LoadUser(r)
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

// RequireOwnership middleware ensures user owns the resource or is admin
func (m *AuthbossMiddleware) RequireOwnership(getOwnerID func(*http.Request) (int, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := m.authboss.LoadUser(r)
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
			if int(user.ID) != ownerID {
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

// GetUserFromContext retrieves the Authboss user from request context
func GetAuthbossUserFromContext(authboss *auth.AuthbossConfig, r *http.Request) *auth.AuthbossUser {
	user, err := authboss.Authboss.CurrentUser(r)
	if err != nil || user == nil {
		return nil
	}

	authUser, ok := user.(*auth.AuthbossUser)
	if !ok {
		return nil
	}

	return authUser
}

// GetUserFromContext retrieves the user from request context (compatibility with existing code)
func GetUserFromContextAuthboss(authboss *auth.AuthbossConfig, r *http.Request) *models.User {
	authUser := GetAuthbossUserFromContext(authboss, r)
	if authUser == nil {
		return nil
	}

	// Convert AuthbossUser back to models.User for compatibility
	return &models.User{
		ID:        int(authUser.ID),
		Email:     authUser.Email,
		FirstName: authUser.FirstName,
		LastName:  authUser.LastName,
		Role:      authUser.Role,
		CreatedAt: authUser.CreatedAt,
		UpdatedAt: authUser.UpdatedAt,
	}
}

// RequireAuthGlobal is a global middleware function that requires authentication using Authboss
func RequireAuthGlobal(authbossConfig *auth.AuthbossConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return authboss.Middleware(authbossConfig.Authboss, true, false, false)(next)
	}
}

// SessionCleanup middleware that periodically cleans up expired sessions
func (m *AuthbossMiddleware) SessionCleanup(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Authboss handles session cleanup internally
		// We can add additional cleanup logic here if needed
		next.ServeHTTP(w, r)
	})
}

// SecurityValidation middleware that validates session security
func (m *AuthbossMiddleware) SecurityValidation(sessionStore sessions.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation for public routes
			if !m.isProtectedRoute(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Validate session security
			if !m.authboss.ValidateSessionSecurity(r, sessionStore) {
				// Clear invalid session and redirect to login
				session, err := sessionStore.Get(r, "session")
				if err == nil {
					session.Values = make(map[interface{}]interface{})
					session.Options.MaxAge = -1
					session.Save(r, w)
				}

				http.Redirect(w, r, "/auth/login?session_expired=1", http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isProtectedRoute checks if a route requires authentication
func (m *AuthbossMiddleware) isProtectedRoute(path string) bool {
	protectedPrefixes := []string{
		"/dashboard",
		"/organizer",
		"/admin",
		"/moderator",
		"/cart",
		"/checkout",
		"/api",
	}

	for _, prefix := range protectedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// SetUserContext sets the user in the context (for testing) - Authboss compatible
func SetAuthbossUserContext(ctx context.Context, user *auth.AuthbossUser) context.Context {
	return context.WithValue(ctx, authboss.CTXKeyUser, user)
}
