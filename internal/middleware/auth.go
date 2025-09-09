package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"

	"github.com/gorilla/sessions"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

// AuthMiddleware provides authentication functionality
type AuthMiddleware struct {
	authService services.AuthServiceInterface
	store       sessions.Store
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService services.AuthServiceInterface, store sessions.Store) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		store:       store,
	}
}

// LoadUser middleware loads the current user from session and adds to context
func (m *AuthMiddleware) LoadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := m.store.Get(r, "session")
		if err != nil {
			// Continue without user if session is invalid
			next.ServeHTTP(w, r)
			return
		}

		userID, ok := session.Values["user_id"].(int)
		if !ok || userID == 0 {
			// Try to convert from other types (session storage might convert types)
			if userIDValue, exists := session.Values["user_id"]; exists {
				switch v := userIDValue.(type) {
				case float64:
					userID = int(v)
					ok = userID != 0
				case string:
					if parsedID, err := strconv.Atoi(v); err == nil {
						userID = parsedID
						ok = userID != 0
					}
				}
			}
			
			if !ok || userID == 0 {
				// No valid user in session
				next.ServeHTTP(w, r)
				return
			}
		}

		// Validate session and get user
		sessionID, ok := session.Values["session_id"].(string)
		if !ok {
			// Invalid session
			next.ServeHTTP(w, r)
			return
		}

		user, err := m.authService.ValidateSession(sessionID)
		if err != nil {
			// Invalid or expired session, clear it
			session.Values["user_id"] = nil
			session.Values["session_id"] = nil
			session.Values["csrf_token"] = nil
			session.Values["remember_me"] = nil
			session.Options.MaxAge = -1
			session.Save(r, w)
			next.ServeHTTP(w, r)
			return
		}

		// Generate CSRF token if not present
		if _, ok := session.Values["csrf_token"].(string); !ok {
			session.Values["csrf_token"] = GenerateCSRFToken()
			session.Save(r, w)
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth middleware ensures user is authenticated
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
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

// RequireRole middleware ensures user has the required role
func (m *AuthMiddleware) RequireRole(role models.UserRole) func(http.Handler) http.Handler {
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

// RequireOwnership middleware ensures user owns the resource or is admin
func (m *AuthMiddleware) RequireOwnership(getOwnerID func(*http.Request) (int, error)) func(http.Handler) http.Handler {
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
			if user.Role == models.RoleAdmin {
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

// GetUserFromContext retrieves the user from request context
func GetUserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// SetUserContext sets the user in the context (for testing)
func SetUserContext(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, UserContextKey, user)
}

// IsHTMXRequest checks if the request is from HTMX
func IsHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// RequireAuth is a global middleware function that requires authentication
func RequireAuth(next http.Handler) http.Handler {
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

// GenerateCSRFToken generates a CSRF token for the session
func GenerateCSRFToken() string {
	// Generate a secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		// Fallback to timestamp-based token if crypto/rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(tokenBytes)
}

// GetCSRFToken retrieves the CSRF token from the session
func GetCSRFToken(r *http.Request, store sessions.Store) string {
	session, err := store.Get(r, "session")
	if err != nil {
		return ""
	}
	
	token, ok := session.Values["csrf_token"].(string)
	if !ok {
		// Generate a new token if none exists
		token = GenerateCSRFToken()
		session.Values["csrf_token"] = token
		// Note: We can't save the session here without the ResponseWriter
	}
	
	return token
}

// SessionCleanup middleware that periodically cleans up expired sessions
func (m *AuthMiddleware) SessionCleanup(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Run cleanup occasionally (1% chance per request)
		// In production, this should be a separate background job
		if time.Now().UnixNano()%100 == 0 {
			go func() {
				if cleanupService, ok := m.authService.(interface{ CleanupExpiredSessions() error }); ok {
					cleanupService.CleanupExpiredSessions()
				}
			}()
		}
		
		next.ServeHTTP(w, r)
	})
}