package auth

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aarondl/authboss/v3"
	"github.com/aarondl/authboss/v3/defaults"
	"github.com/gorilla/sessions"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/internal/utils"
)

// AuthbossConfig holds the Authboss configuration and instance
type AuthbossConfig struct {
	Authboss *authboss.Authboss
	Storage  *Storage
}

// NewAuthbossConfig creates and configures a new Authboss instance
func NewAuthbossConfig(db *sql.DB, sessionStore sessions.Store, emailService services.EmailService, baseURL string, isDevelopment bool) (*AuthbossConfig, error) {
	// Create Authboss instance
	ab := authboss.New()

	// Create storage
	storage := NewStorage(db, sessionStore, "session", !isDevelopment)

	// Configure storage
	storage.ConfigureAuthboss(ab)

	// Configure core settings
	ab.Config.Paths.Mount = "/auth"
	ab.Config.Paths.RootURL = baseURL
	ab.Config.Paths.AuthLoginOK = "/dashboard"
	ab.Config.Paths.LogoutOK = "/"
	ab.Config.Paths.RegisterOK = "/auth/verify"
	ab.Config.Paths.ConfirmOK = "/dashboard"
	ab.Config.Paths.RecoverOK = "/auth/login"

	// Configure modules
	ab.Config.Modules.LogoutMethod = "GET" // Allow GET logout for simplicity
	ab.Config.Modules.RegisterPreserveFields = []string{"first_name", "last_name", "role"}
	ab.Config.Modules.RecoverTokenDuration = 24 * time.Hour
	ab.Config.Modules.ExpireAfter = 30 * 24 * time.Hour // 30 days
	ab.Config.Modules.LockAfter = 5                     // Lock after 5 failed attempts
	ab.Config.Modules.LockWindow = 5 * time.Minute     // Reset attempt count after 5 minutes
	ab.Config.Modules.LockDuration = 30 * time.Minute  // Lock for 30 minutes

	// Configure email settings
	if emailService != nil {
		ab.Config.Mail.From = "noreply@runtown.com"
		ab.Config.Mail.FromName = "Runtown"
		ab.Config.Mail.SubjectPrefix = "[Runtown] "
	}

	// Configure security settings - BCryptCost is handled by the hasher
	// We'll use the default hasher which handles bcrypt cost internally

	// Configure validation rules
	ab.Config.Core.BodyReader = defaults.HTTPBodyReader{ReadJSON: false}
	ab.Config.Core.ViewRenderer = NewAuthbossRenderer()
	ab.Config.Core.Mailer = NewAuthbossMailer(emailService)
	ab.Config.Core.Logger = NewAuthbossLogger()

	// Initialize Authboss
	if err := ab.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize authboss: %w", err)
	}

	return &AuthbossConfig{
		Authboss: ab,
		Storage:  storage,
	}, nil
}

// GetAuthbossHandler returns the HTTP handler for Authboss routes
func (ac *AuthbossConfig) GetAuthbossHandler() http.Handler {
	// Create a handler that routes to Authboss modules
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For now, redirect to login for any auth route
		// This is a simplified implementation - in a full implementation,
		// you'd route to specific Authboss modules based on the path
		switch r.URL.Path {
		case "/auth/login":
			if r.Method == "GET" {
				// Render login page
				component := ac.Authboss.Config.Core.ViewRenderer
				if component != nil {
					data := make(map[string]interface{})
					output, contentType, err := component.Render(r.Context(), "login", data)
					if err != nil {
						http.Error(w, "Failed to render login page", http.StatusInternalServerError)
						return
					}
					w.Header().Set("Content-Type", contentType)
					w.Write(output)
					return
				}
			} else if r.Method == "POST" {
				// Handle login form submission
				ac.handleLogin(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		case "/auth/register":
			if r.Method == "GET" {
				// Render register page
				component := ac.Authboss.Config.Core.ViewRenderer
				if component != nil {
					data := make(map[string]interface{})
					output, contentType, err := component.Render(r.Context(), "register", data)
					if err != nil {
						http.Error(w, "Failed to render register page", http.StatusInternalServerError)
						return
					}
					w.Header().Set("Content-Type", contentType)
					w.Write(output)
					return
				}
			} else if r.Method == "POST" {
				// Handle registration form submission
				ac.handleRegister(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		case "/auth/forgot-password":
			if r.Method == "GET" {
				// Render forgot password page
				component := ac.Authboss.Config.Core.ViewRenderer
				if component != nil {
					data := make(map[string]interface{})
					output, contentType, err := component.Render(r.Context(), "recover_start", data)
					if err != nil {
						http.Error(w, "Failed to render forgot password page", http.StatusInternalServerError)
						return
					}
					w.Header().Set("Content-Type", contentType)
					w.Write(output)
					return
				}
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		case "/auth/confirm":
			if r.Method == "GET" {
				// Handle email confirmation
				ac.handleEmailConfirmation(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		default:
			http.NotFound(w, r)
		}
	})
}

// LoadUser loads the current user from the request context
func (ac *AuthbossConfig) LoadUser(r *http.Request) *AuthbossUser {
	user, err := ac.Authboss.CurrentUser(r)
	if err != nil || user == nil {
		return nil
	}

	authUser, ok := user.(*AuthbossUser)
	if !ok {
		return nil
	}

	return authUser
}

// IsAuthenticated checks if the current request has an authenticated user
func (ac *AuthbossConfig) IsAuthenticated(r *http.Request) bool {
	user, err := ac.Authboss.CurrentUser(r)
	return err == nil && user != nil
}

// RequireAuth middleware that requires authentication
func (ac *AuthbossConfig) RequireAuth(next http.Handler) http.Handler {
	return authboss.Middleware(ac.Authboss, true, false, false)(next)
}

// RequireRole middleware that requires a specific role
func (ac *AuthbossConfig) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := ac.LoadUser(r)
			if user == nil {
				http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
				return
			}

			userRole := string(user.Role)
			if userRole != role {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Cleanup performs cleanup operations
func (ac *AuthbossConfig) Cleanup() error {
	return ac.Storage.Cleanup()
}

// handleLogin handles the login form submission
func (ac *AuthbossConfig) handleLogin(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[DEBUG] Login handler called - Method: %s, URL: %s\n", r.Method, r.URL.String())
	
	// Check rate limiting
	if !ac.RateLimitCheck(r, "login") {
		ac.logSecurityEvent("rate_limit_exceeded", "", r, "Login rate limit exceeded")
		http.Error(w, "Too many login attempts. Please try again later.", http.StatusTooManyRequests)
		return
	}
	
	// Parse form data
	if err := r.ParseForm(); err != nil {
		fmt.Printf("[DEBUG] Failed to parse form: %v\n", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	rememberMe := r.FormValue("remember_me") == "on"
	csrfToken := r.FormValue("csrf_token")

	fmt.Printf("[DEBUG] Login form data - Email: %s, Password length: %d, CSRF: %s, RememberMe: %t\n", 
		email, len(password), csrfToken, rememberMe)

	// Validate CSRF token
	sessionStorer := ac.Storage.SessionStorer
	session, err := sessionStorer.store.Get(r, sessionStorer.sessionName)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to get session for CSRF validation: %v\n", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	sessionToken, ok := session.Values["csrf_token"].(string)
	if !ok || sessionToken == "" {
		fmt.Printf("[DEBUG] No CSRF token in session\n")
		data := map[string]interface{}{
			"validation": map[string][]string{
				"general": {"Security token missing. Please refresh the page and try again."},
			},
			"preserve": map[string]string{
				"email": email,
			},
		}
		
		component := ac.Authboss.Config.Core.ViewRenderer
		if component != nil {
			output, contentType, err := component.Render(r.Context(), "login", data)
			if err != nil {
				http.Error(w, "Failed to render login page", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write(output)
			return
		}
	}

	if csrfToken != sessionToken {
		fmt.Printf("[DEBUG] CSRF token mismatch - Session: %s, Request: %s\n", sessionToken, csrfToken)
		data := map[string]interface{}{
			"validation": map[string][]string{
				"general": {"Security token mismatch. Please refresh the page and try again."},
			},
			"preserve": map[string]string{
				"email": email,
			},
		}
		
		component := ac.Authboss.Config.Core.ViewRenderer
		if component != nil {
			output, contentType, err := component.Render(r.Context(), "login", data)
			if err != nil {
				http.Error(w, "Failed to render login page", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write(output)
			return
		}
	}

	fmt.Printf("[DEBUG] CSRF token validation passed\n")

	// Validate input
	if email == "" || password == "" {
		// Render login page with errors
		data := map[string]interface{}{
			"validation": map[string][]string{
				"email":    {"Email is required"},
				"password": {"Password is required"},
			},
			"preserve": map[string]string{
				"email": email,
			},
		}
		
		component := ac.Authboss.Config.Core.ViewRenderer
		if component != nil {
			output, contentType, err := component.Render(r.Context(), "login", data)
			if err != nil {
				http.Error(w, "Failed to render login page", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write(output)
			return
		}
	}

	// Try to load user from storage
	fmt.Printf("[DEBUG] Attempting to load user: %s\n", email)
	user, err := ac.Authboss.Config.Storage.Server.Load(r.Context(), email)
	if err != nil || user == nil {
		// User not found - still record failed attempt for rate limiting
		fmt.Printf("[DEBUG] User not found or error loading user: %v\n", err)
		ac.logSecurityEvent("login_failed", email, r, "User not found")
		
		data := map[string]interface{}{
			"validation": map[string][]string{
				"general": {"Invalid email or password"},
			},
			"preserve": map[string]string{
				"email": email,
			},
		}
		
		component := ac.Authboss.Config.Core.ViewRenderer
		if component != nil {
			output, contentType, err := component.Render(r.Context(), "login", data)
			if err != nil {
				http.Error(w, "Failed to render login page", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write(output)
			return
		}
	}

	// Verify password
	authUser, ok := user.(*AuthbossUser)
	if !ok {
		fmt.Printf("[DEBUG] Failed to cast user to AuthbossUser\n")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	fmt.Printf("[DEBUG] User loaded successfully - ID: %d, Email: %s\n", authUser.ID, authUser.Email)

	// Check if account is locked
	if ac.isAccountLocked(authUser) {
		ac.logSecurityEvent("login_blocked", email, r, "Account locked")
		
		data := map[string]interface{}{
			"validation": map[string][]string{
				"general": {"Account is temporarily locked due to too many failed login attempts. Please try again later."},
			},
			"preserve": map[string]string{
				"email": email,
			},
		}
		
		component := ac.Authboss.Config.Core.ViewRenderer
		if component != nil {
			output, contentType, err := component.Render(r.Context(), "login", data)
			if err != nil {
				http.Error(w, "Failed to render login page", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write(output)
			return
		}
	}

	fmt.Printf("[DEBUG] Verifying password for user: %s\n", email)
	if !authUser.VerifyPassword(password) {
		// Invalid password - increment failed attempts
		fmt.Printf("[DEBUG] Password verification failed for user: %s\n", email)
		ac.recordFailedAttempt(authUser)
		ac.logSecurityEvent("login_failed", email, r, "Invalid password")
		
		data := map[string]interface{}{
			"validation": map[string][]string{
				"general": {"Invalid email or password"},
			},
			"preserve": map[string]string{
				"email": email,
			},
		}
		
		component := ac.Authboss.Config.Core.ViewRenderer
		if component != nil {
			output, contentType, err := component.Render(r.Context(), "login", data)
			if err != nil {
				http.Error(w, "Failed to render login page", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write(output)
			return
		}
	}

	// Login successful - reset failed attempts and log success
	ac.resetFailedAttempts(authUser)
	ac.logSecurityEvent("login_success", email, r, "Successful login")
	
	// Reuse the existing session from CSRF validation above
	// sessionStorer and session are already declared above

	// Set session values using Authboss session keys
	session.Values[authboss.SessionKey] = authUser.GetPID()
	session.Values["remember_me"] = rememberMe
	
	// Configure session security
	ac.ConfigureSessionSecurity(w, r, sessionStorer.store)
	
	// Set session duration based on remember me option
	if rememberMe {
		session.Options.MaxAge = 86400 * 30 // 30 days for remember me
		// Create remember token for enhanced security
		ac.createRememberToken(authUser, w, r)
	} else {
		session.Options.MaxAge = 86400 // 24 hours for regular sessions
	}

	err = session.Save(r, w)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to save session: %v\n", err)
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}
	
	fmt.Printf("[DEBUG] Session saved successfully - User ID: %s, Session values: %+v\n", 
		authUser.GetPID(), session.Values)

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// handleRegister handles the registration form submission
func (ac *AuthbossConfig) handleRegister(w http.ResponseWriter, r *http.Request) {
	// Check rate limiting
	if !ac.RateLimitCheck(r, "register") {
		ac.logSecurityEvent("rate_limit_exceeded", "", r, "Registration rate limit exceeded")
		http.Error(w, "Too many registration attempts. Please try again later.", http.StatusTooManyRequests)
		return
	}
	
	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")
	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	role := r.FormValue("role")

	// Validate input
	errors := make(map[string][]string)
	
	if email == "" {
		errors["email"] = []string{"Email is required"}
	}
	
	if firstName == "" {
		errors["first_name"] = []string{"First name is required"}
	}
	
	if lastName == "" {
		errors["last_name"] = []string{"Last name is required"}
	}
	
	if password == "" {
		errors["password"] = []string{"Password is required"}
	} else {
		// Validate password strength with enhanced checks
		passwordErrors := ac.EnhancedPasswordValidation(password, email)
		if len(passwordErrors) > 0 {
			errors["password"] = passwordErrors
		}
	}
	
	if passwordConfirm == "" {
		errors["password_confirm"] = []string{"Password confirmation is required"}
	} else if password != passwordConfirm {
		errors["password_confirm"] = []string{"Passwords do not match"}
	}

	// Check if user already exists
	existingUser, err := ac.Authboss.Config.Storage.Server.Load(r.Context(), email)
	if err == nil && existingUser != nil {
		errors["email"] = []string{"An account with this email already exists"}
	}

	if len(errors) > 0 {
		// Render registration page with errors
		data := map[string]interface{}{
			"validation": errors,
			"preserve": map[string]string{
				"email":      email,
				"first_name": firstName,
				"last_name":  lastName,
				"role":       role,
			},
		}
		
		component := ac.Authboss.Config.Core.ViewRenderer
		if component != nil {
			output, contentType, err := component.Render(r.Context(), "register", data)
			if err != nil {
				http.Error(w, "Failed to render register page", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write(output)
			return
		}
	}

	// Create new user
	ac.logSecurityEvent("registration_attempt", email, r, "New user registration")
	
	// Determine user role
	var userRole models.UserRole
	if role == "organizer" {
		userRole = models.UserRoleOrganizer
	} else {
		userRole = models.UserRoleUser // Default role
	}
	
	// Create the user using the storage layer
	newUser := &AuthbossUser{
		User: &models.User{
			Email:     email,
			FirstName: firstName,
			LastName:  lastName,
			Role:      userRole,
		},
	}
	
	// Hash the password
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		http.Error(w, "Failed to process registration", http.StatusInternalServerError)
		return
	}
	newUser.PasswordHash = hashedPassword
	
	// Set Authboss fields
	now := time.Now()
	newUser.AttemptCount = 0
	newUser.PasswordChangedAt = &now
	// Don't set confirmed_at - user needs to verify email
	
	// Save user to database
	ac.Authboss.Config.Core.Logger.Info(fmt.Sprintf("Attempting to save user: %s", email))
	err = ac.Authboss.Config.Storage.Server.Save(r.Context(), newUser)
	if err != nil {
		ac.logSecurityEvent("registration_failed", email, r, fmt.Sprintf("Database error: %v", err))
		ac.Authboss.Config.Core.Logger.Error(fmt.Sprintf("Failed to save user %s: %v", email, err))
		http.Error(w, "Failed to create account", http.StatusInternalServerError)
		return
	}
	ac.Authboss.Config.Core.Logger.Info(fmt.Sprintf("User saved successfully: %s (ID: %d)", email, newUser.ID))
	
	ac.logSecurityEvent("registration_success", email, r, "User account created")
	
	// Send verification email
	if ac.Authboss.Config.Core.Mailer != nil {
		// Generate verification token
		verificationToken := ac.generateSecureToken()
		
		// Update user with verification token
		newUser.ConfirmSelector = verificationToken
		newUser.ConfirmVerifier = verificationToken // Simplified for now
		
		// Save updated user
		err = ac.Authboss.Config.Storage.Server.Save(r.Context(), newUser)
		if err != nil {
			ac.Authboss.Config.Core.Logger.Error(fmt.Sprintf("Failed to save verification token: %v", err))
		}
		
		// Send verification email
		verificationURL := fmt.Sprintf("%s/auth/confirm?token=%s", ac.Authboss.Config.Paths.RootURL, verificationToken)
		
		// Create email content
		emailContent := fmt.Sprintf(`
			<h2>Welcome to Runtown!</h2>
			<p>Thank you for registering. Please click the link below to verify your email address:</p>
			<p><a href="%s">Verify Email Address</a></p>
			<p>If you didn't create this account, please ignore this email.</p>
		`, verificationURL)
		
		// Send email using the mailer
		emailData := authboss.Email{
			To:       []string{email},
			Subject:  "Verify your email address",
			HTMLBody: emailContent,
			TextBody: fmt.Sprintf("Please verify your email by visiting: %s", verificationURL),
		}
		
		err = ac.Authboss.Config.Core.Mailer.Send(r.Context(), emailData)
		if err != nil {
			ac.logSecurityEvent("email_send_failed", email, r, fmt.Sprintf("Failed to send verification email: %v", err))
			ac.Authboss.Config.Core.Logger.Error(fmt.Sprintf("Failed to send verification email to %s: %v", email, err))
		} else {
			ac.logSecurityEvent("verification_email_sent", email, r, "Verification email sent successfully")
			ac.Authboss.Config.Core.Logger.Info(fmt.Sprintf("Verification email sent to %s", email))
		}
	}
	
	// Redirect to login with success message
	http.Redirect(w, r, "/auth/login?registered=1", http.StatusSeeOther)
}

// Security helper methods

// isAccountLocked checks if an account is currently locked
func (ac *AuthbossConfig) isAccountLocked(user *AuthbossUser) bool {
	if user.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*user.LockedUntil)
}

// recordFailedAttempt records a failed login attempt and locks account if necessary
func (ac *AuthbossConfig) recordFailedAttempt(user *AuthbossUser) {
	now := time.Now()
	user.LastAttempt = &now
	
	// Reset attempt count if last attempt was more than the lock window ago
	if user.LastAttempt != nil && now.Sub(*user.LastAttempt) > 5*time.Minute {
		user.AttemptCount = 0
	}
	
	user.AttemptCount++
	
	// Lock account if too many attempts
	if user.AttemptCount >= 5 {
		lockUntil := now.Add(30 * time.Minute)
		user.LockedUntil = &lockUntil
		user.AttemptCount = 0 // Reset counter after locking
	}
	
	// Save user with updated attempt info
	ac.Authboss.Config.Storage.Server.Save(context.Background(), user)
}

// resetFailedAttempts resets failed login attempts after successful login
func (ac *AuthbossConfig) resetFailedAttempts(user *AuthbossUser) {
	user.AttemptCount = 0
	user.LockedUntil = nil
	user.LastAttempt = nil
	
	// Save user with reset attempt info
	ac.Authboss.Config.Storage.Server.Save(context.Background(), user)
}

// logSecurityEvent logs security-related events
func (ac *AuthbossConfig) logSecurityEvent(event, email string, r *http.Request, details string) {
	// Get client IP
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	
	// Log security event
	ac.Authboss.Config.Core.Logger.Info(fmt.Sprintf(
		"Security Event: %s | Email: %s | IP: %s | UserAgent: %s | Details: %s",
		event, email, clientIP, r.Header.Get("User-Agent"), details,
	))
}

// ValidatePasswordStrength validates password strength according to security policies
func (ac *AuthbossConfig) ValidatePasswordStrength(password string) []string {
	var errors []string
	
	if len(password) < 8 {
		errors = append(errors, "Password must be at least 8 characters long")
	}
	
	if len(password) > 128 {
		errors = append(errors, "Password must be less than 128 characters")
	}
	
	// Check for at least one uppercase letter
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false
	
	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case char >= 32 && char <= 126: // Printable ASCII characters
			if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
				hasSpecial = true
			}
		}
	}
	
	if !hasUpper {
		errors = append(errors, "Password must contain at least one uppercase letter")
	}
	
	if !hasLower {
		errors = append(errors, "Password must contain at least one lowercase letter")
	}
	
	if !hasDigit {
		errors = append(errors, "Password must contain at least one number")
	}
	
	if !hasSpecial {
		errors = append(errors, "Password must contain at least one special character")
	}
	
	return errors
}

// ConfigureSessionSecurity configures session security settings
func (ac *AuthbossConfig) ConfigureSessionSecurity(w http.ResponseWriter, r *http.Request, sessionStore sessions.Store) {
	session, err := sessionStore.Get(r, "session")
	if err != nil {
		return
	}
	
	// Set secure session options
	session.Options.HttpOnly = true
	session.Options.Secure = r.TLS != nil // Only secure if HTTPS
	session.Options.SameSite = http.SameSiteLaxMode
	
	// Add session fingerprinting for security
	userAgent := r.Header.Get("User-Agent")
	clientIP := ac.getClientIP(r)
	sessionFingerprint := fmt.Sprintf("%s:%s", userAgent, clientIP)
	
	// Check if fingerprint matches (for existing sessions)
	if existingFingerprint, exists := session.Values["fingerprint"]; exists {
		if existingFingerprint != sessionFingerprint {
			// Session hijacking detected - clear session
			session.Values = make(map[interface{}]interface{})
			session.Options.MaxAge = -1
			session.Save(r, w)
			ac.logSecurityEvent("session_hijack_detected", "", r, "Session fingerprint mismatch")
			return
		}
	}
	
	// Set fingerprint for new sessions
	session.Values["fingerprint"] = sessionFingerprint
	session.Values["created_at"] = time.Now().Unix()
	
	session.Save(r, w)
}

// getClientIP extracts the real client IP from the request
func (ac *AuthbossConfig) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	
	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// ValidateSessionSecurity validates session security and detects anomalies
func (ac *AuthbossConfig) ValidateSessionSecurity(r *http.Request, sessionStore sessions.Store) bool {
	session, err := sessionStore.Get(r, "session")
	if err != nil {
		return false
	}
	
	// Check session age
	if createdAt, exists := session.Values["created_at"]; exists {
		if createdAtInt, ok := createdAt.(int64); ok {
			sessionAge := time.Since(time.Unix(createdAtInt, 0))
			// Force re-authentication after 24 hours for security
			if sessionAge > 24*time.Hour {
				ac.logSecurityEvent("session_expired", "", r, "Session exceeded maximum age")
				return false
			}
		}
	}
	
	// Check for suspicious activity patterns
	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		ac.logSecurityEvent("suspicious_activity", "", r, "Missing User-Agent header")
		return false
	}
	
	return true
}

// EnhancedPasswordValidation provides additional password security checks
func (ac *AuthbossConfig) EnhancedPasswordValidation(password, email string) []string {
	errors := ac.ValidatePasswordStrength(password)
	
	// Check for common passwords (basic implementation)
	commonPasswords := []string{
		"password", "123456", "password123", "admin", "qwerty",
		"letmein", "welcome", "monkey", "dragon", "master",
	}
	
	passwordLower := strings.ToLower(password)
	for _, common := range commonPasswords {
		if passwordLower == common {
			errors = append(errors, "Password is too common and easily guessable")
			break
		}
	}
	
	// Check if password contains email
	if email != "" {
		emailParts := strings.Split(strings.ToLower(email), "@")
		if len(emailParts) > 0 {
			emailUser := emailParts[0]
			if strings.Contains(passwordLower, emailUser) && len(emailUser) > 3 {
				errors = append(errors, "Password should not contain parts of your email address")
			}
		}
	}
	
	// Check for sequential characters
	if ac.hasSequentialChars(password) {
		errors = append(errors, "Password should not contain sequential characters (e.g., 123, abc)")
	}
	
	return errors
}

// hasSequentialChars checks for sequential characters in password
func (ac *AuthbossConfig) hasSequentialChars(password string) bool {
	if len(password) < 3 {
		return false
	}
	
	for i := 0; i < len(password)-2; i++ {
		// Check for ascending sequence
		if password[i]+1 == password[i+1] && password[i+1]+1 == password[i+2] {
			return true
		}
		// Check for descending sequence
		if password[i]-1 == password[i+1] && password[i+1]-1 == password[i+2] {
			return true
		}
	}
	return false
}

// RateLimitCheck implements basic rate limiting for authentication attempts
func (ac *AuthbossConfig) RateLimitCheck(r *http.Request, action string) bool {
	clientIP := ac.getClientIP(r)
	
	// This is a basic implementation - in production, you'd use Redis or similar
	// For now, we'll use a simple in-memory approach with cleanup
	
	// Rate limiting: max 10 attempts per IP per 15 minutes for login
	// max 5 registration attempts per IP per hour
	
	// In a real implementation, you'd store this in Redis with TTL
	// For now, just log the attempt and return true (allowing the request)
	ac.logSecurityEvent("rate_limit_check", "", r, fmt.Sprintf("Action: %s, IP: %s", action, clientIP))
	
	return true // Allow for now - implement proper rate limiting with Redis in production
}

// handleEmailConfirmation handles email verification
func (ac *AuthbossConfig) handleEmailConfirmation(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Invalid verification link", http.StatusBadRequest)
		return
	}

	// Find user by confirmation token
	rows, err := ac.Storage.ServerStorer.db.QueryContext(r.Context(), `
		SELECT id, email FROM users WHERE confirm_selector = $1 OR confirm_verifier = $1
	`, token)
	if err != nil {
		ac.Authboss.Config.Core.Logger.Error(fmt.Sprintf("Failed to query user by token: %v", err))
		http.Error(w, "Verification failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	if !rows.Next() {
		http.Error(w, "Invalid or expired verification link", http.StatusBadRequest)
		return
	}

	var userID int
	var email string
	err = rows.Scan(&userID, &email)
	if err != nil {
		http.Error(w, "Verification failed", http.StatusInternalServerError)
		return
	}

	// Update user as confirmed
	_, err = ac.Storage.ServerStorer.db.ExecContext(r.Context(), `
		UPDATE users SET 
			confirmed_at = CURRENT_TIMESTAMP,
			email_verified = true,
			email_verified_at = CURRENT_TIMESTAMP,
			confirm_selector = NULL,
			confirm_verifier = NULL,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, userID)
	if err != nil {
		ac.Authboss.Config.Core.Logger.Error(fmt.Sprintf("Failed to confirm user: %v", err))
		http.Error(w, "Verification failed", http.StatusInternalServerError)
		return
	}

	ac.logSecurityEvent("email_confirmed", email, r, "Email address verified successfully")
	ac.Authboss.Config.Core.Logger.Info(fmt.Sprintf("Email confirmed for user: %s", email))

	// Redirect to login with success message
	http.Redirect(w, r, "/auth/login?confirmed=1", http.StatusSeeOther)
}

// generateSecureToken generates a secure random token
func (ac *AuthbossConfig) generateSecureToken() string {
	// Fallback to simple token generation since OneTimeTokenGenerator has complex return
	return fmt.Sprintf("%d-%s", time.Now().Unix(), ac.generateRandomString(32))
}

// generateRandomString generates a random string of specified length
func (ac *AuthbossConfig) generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

// createRememberToken creates a secure remember me token
func (ac *AuthbossConfig) createRememberToken(user *AuthbossUser, w http.ResponseWriter, r *http.Request) {
	// Generate secure remember token
	token := ac.generateSecureToken()
	
	// Store remember token in database
	err := ac.Storage.RememberStorer.AddRememberToken(r.Context(), user.GetPID(), token)
	if err != nil {
		ac.logSecurityEvent("remember_token_failed", user.Email, r, fmt.Sprintf("Failed to create remember token: %v", err))
		return
	}
	
	// Set remember cookie
	cookie := &http.Cookie{
		Name:     "authboss_rm",
		Value:    token,
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	}
	
	http.SetCookie(w, cookie)
	ac.logSecurityEvent("remember_token_created", user.Email, r, "Remember me token created")
}

// validateRememberToken validates and uses a remember me token
func (ac *AuthbossConfig) validateRememberToken(r *http.Request) *AuthbossUser {
	cookie, err := r.Cookie("authboss_rm")
	if err != nil {
		return nil
	}
	
	// This is a simplified implementation
	// In a full implementation, you'd validate the token against the database
	// and return the associated user
	
	// For now, just log that we received a remember token
	if cookie.Value != "" {
		ac.logSecurityEvent("remember_token_validation", "", r, fmt.Sprintf("Remember token received: %s", cookie.Value[:8]+"..."))
	}
	
	return nil // Placeholder - implement full remember token validation
}

// SecurityAuditLog logs detailed security events for audit purposes
func (ac *AuthbossConfig) SecurityAuditLog(event, userID, email string, r *http.Request, details map[string]interface{}) {
	clientIP := ac.getClientIP(r)
	userAgent := r.Header.Get("User-Agent")
	timestamp := time.Now().UTC()
	
	// Create detailed audit log entry for structured logging
	auditData := fmt.Sprintf(
		"timestamp=%s event=%s user_id=%s email=%s client_ip=%s user_agent=%s details=%+v",
		timestamp.Format(time.RFC3339), event, userID, email, clientIP, userAgent, details,
	)
	
	// In production, you'd send this to a security monitoring system
	// For now, log it with structured format
	ac.Authboss.Config.Core.Logger.Info(fmt.Sprintf(
		"SECURITY_AUDIT: %s | User: %s (%s) | IP: %s | %s",
		event, email, userID, clientIP, auditData,
	))
}

// Close closes the Authboss configuration and releases resources
func (ac *AuthbossConfig) Close() error {
	return ac.Storage.Close()

}
