package server

import (
	"database/sql"
	"fmt"
	"net/http"

	"event-ticketing-platform/internal/auth"
	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
)

// AuthbossIntegration handles the integration of Authboss with the existing server
type AuthbossIntegration struct {
	authbossConfig *auth.AuthbossConfig
	middleware     *middleware.AuthbossMiddleware
}

// NewAuthbossIntegration creates a new Authboss integration
func NewAuthbossIntegration(db *sql.DB, sessionStore sessions.Store, emailService services.EmailService, baseURL string, isDevelopment bool) (*AuthbossIntegration, error) {
	// Create Authboss configuration
	authbossConfig, err := auth.NewAuthbossConfig(db, sessionStore, emailService, baseURL, isDevelopment)
	if err != nil {
		return nil, err
	}

	// Create Authboss middleware
	authbossMiddleware := middleware.NewAuthbossMiddleware(authbossConfig)

	return &AuthbossIntegration{
		authbossConfig: authbossConfig,
		middleware:     authbossMiddleware,
	}, nil
}

// SetupAuthRoutes sets up the Authboss authentication routes
func (ai *AuthbossIntegration) SetupAuthRoutes(r chi.Router) {
	// Set up auth routes directly without mounting
	r.Route("/auth", func(r chi.Router) {
		// Mount the Authboss handler for specific routes
		r.Handle("/login", ai.authbossConfig.GetAuthbossHandler())
		r.Handle("/register", ai.authbossConfig.GetAuthbossHandler())
		r.Handle("/recover", ai.authbossConfig.GetAuthbossHandler())
		r.Handle("/confirm", ai.authbossConfig.GetAuthbossHandler())
		r.Handle("/forgot-password", ai.authbossConfig.GetAuthbossHandler())
		
		// Add CSRF token endpoint for AJAX requests
		r.Get("/csrf-token", ai.handleGetCSRFToken)
		
		// Add custom logout route that works with existing frontend
		r.Get("/logout", ai.handleLogout)
		r.Post("/logout", ai.handleLogout)
	})
}

// GetLoadUserMiddleware returns the Authboss load user middleware
func (ai *AuthbossIntegration) GetLoadUserMiddleware() func(http.Handler) http.Handler {
	return ai.middleware.LoadUser
}

// GetRequireAuthMiddleware returns the Authboss require auth middleware
func (ai *AuthbossIntegration) GetRequireAuthMiddleware() func(http.Handler) http.Handler {
	return ai.middleware.RequireAuth
}

// GetRequireRoleMiddleware returns the Authboss require role middleware
func (ai *AuthbossIntegration) GetRequireRoleMiddleware(role string) func(http.Handler) http.Handler {
	return ai.authbossConfig.RequireRole(role)
}

// handleGetCSRFToken returns a CSRF token for AJAX requests
func (ai *AuthbossIntegration) handleGetCSRFToken(w http.ResponseWriter, r *http.Request) {
	// Generate new CSRF token using the middleware function
	token := middleware.GenerateCSRFToken()
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"csrf_token":"%s"}`, token)))
}

// handleLogout handles the logout request and redirects appropriately
func (ai *AuthbossIntegration) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Clear the user session manually since Authboss doesn't have a direct logout method
	// We'll use the storage interface to clear the session
	err := ai.authbossConfig.Storage.ClearSession(w, r)
	if err != nil {
		http.Error(w, "Logout failed", http.StatusInternalServerError)
		return
	}
	
	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// GetAuthbossConfig returns the Authboss configuration for direct access if needed
func (ai *AuthbossIntegration) GetAuthbossConfig() *auth.AuthbossConfig {
	return ai.authbossConfig
}

// Close closes the Authboss integration and releases resources
func (ai *AuthbossIntegration) Close() error {
	return ai.authbossConfig.Close()
}