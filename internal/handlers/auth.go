package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/web/templates/pages"

	"github.com/gorilla/sessions"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	authService       services.AuthServiceInterface
	authFlowService   *services.AuthFlowService
	onboardingService *services.OnboardingService
	store             sessions.Store
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService services.AuthServiceInterface, store sessions.Store) *AuthHandler {
	return &AuthHandler{
		authService:       authService,
		authFlowService:   services.NewAuthFlowService(store),
		onboardingService: services.NewOnboardingService(store),
		store:             store,
	}
}

// getCSRFToken gets or creates a CSRF token for the session
func (h *AuthHandler) getCSRFToken(w http.ResponseWriter, r *http.Request) string {
	session, err := h.store.Get(r, "session")
	if err != nil {
		return ""
	}
	
	token, ok := session.Values["csrf_token"].(string)
	if !ok || token == "" {
		token = middleware.GenerateCSRFToken()
		session.Values["csrf_token"] = token
		session.Save(r, w)
	}
	
	return token
}

// isValidEmail validates email format using regex
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// LoginPage renders the login page
func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	
	// If user is already logged in, redirect to dashboard
	if user != nil {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	// Render login page
	component := pages.LoginPage(nil, make(map[string][]string), make(map[string]string))
	err := component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render login page", http.StatusInternalServerError)
		return
	}
}

// LoginSubmit handles login form submission
func (h *AuthHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Printf("Failed to parse form: %v\n", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	rememberMe := r.FormValue("remember_me") == "on"
	csrfToken := r.FormValue("csrf_token")

	// Debug logging
	fmt.Printf("Login attempt - Email: %s, Password length: %d, CSRF token: %s\n", 
		email, len(password), csrfToken)

	// Validate input
	errors := make(map[string][]string)
	formData := map[string]string{
		"email": email,
	}

	if email == "" {
		errors["email"] = []string{"Email is required"}
		fmt.Printf("Validation error: Email is required\n")
	}
	if password == "" {
		errors["password"] = []string{"Password is required"}
		fmt.Printf("Validation error: Password is required\n")
	}

	if len(errors) > 0 {
		fmt.Printf("Login validation failed with errors: %+v\n", errors)
		component := pages.LoginPage(nil, errors, formData)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render login page", http.StatusInternalServerError)
		}
		return
	}

	// Attempt login
	loginReq := &services.LoginRequest{
		Email:      email,
		Password:   password,
		RememberMe: rememberMe,
	}

	authResponse, err := h.authService.Login(loginReq)
	if err != nil {
		// Check if it's an email verification error
		if strings.Contains(err.Error(), "verify your email") {
			errors["email"] = []string{"Please verify your email address before logging in"}
		} else {
			errors["email"] = []string{"Invalid email or password"}
		}
		
		component := pages.LoginPage(nil, errors, formData)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render login page", http.StatusInternalServerError)
		}
		return
	}

	// Create session
	session, err := h.store.Get(r, "session")
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	session.Values["session_id"] = authResponse.SessionID
	session.Values["user_id"] = authResponse.User.ID
	session.Values["remember_me"] = rememberMe
	
	// Set session duration based on remember me option
	if rememberMe {
		// 30 days for remember me
		session.Options.MaxAge = 30 * 24 * 60 * 60
	} else {
		// 24 hours for regular sessions
		session.Options.MaxAge = 24 * 60 * 60
	}
	
	// Generate CSRF token for the session
	session.Values["csrf_token"] = middleware.GenerateCSRFToken()
	
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Get redirect URL from query parameter or default to dashboard
	redirectURL := r.URL.Query().Get("redirect")
	if redirectURL == "" {
		redirectURL = "/dashboard"
	}

	// Redirect to dashboard or specified URL
	w.Header().Set("HX-Redirect", redirectURL)
	w.WriteHeader(http.StatusOK)
}

// RegisterPage renders the registration page
func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	
	// If user is already logged in, redirect to dashboard
	if user != nil {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	// Render registration page
	component := pages.RegisterPage(nil, make(map[string][]string), make(map[string]string))
	err := component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render registration page", http.StatusInternalServerError)
		return
	}
}

// RegisterSubmit handles registration form submission
func (h *AuthHandler) RegisterSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")
	firstName := strings.TrimSpace(r.FormValue("first_name"))
	lastName := strings.TrimSpace(r.FormValue("last_name"))
	roleValue := r.FormValue("role")

	// Determine user role
	role := models.RoleAttendee
	if roleValue == "organizer" {
		role = models.RoleOrganizer
	}

	// Validate input
	errors := make(map[string][]string)
	formData := map[string]string{
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
	}

	if email == "" {
		errors["email"] = []string{"Email is required"}
	} else if !isValidEmail(email) {
		errors["email"] = []string{"Please enter a valid email address"}
	}
	if firstName == "" {
		errors["first_name"] = []string{"First name is required"}
	}
	if lastName == "" {
		errors["last_name"] = []string{"Last name is required"}
	}
	if password == "" {
		errors["password"] = []string{"Password is required"}
	} else if len(password) < 8 {
		errors["password"] = []string{"Password must be at least 8 characters long"}
	}
	if passwordConfirm == "" {
		errors["password_confirm"] = []string{"Password confirmation is required"}
	} else if password != passwordConfirm {
		errors["password_confirm"] = []string{"Passwords do not match"}
	}
	
	// Check terms acceptance
	terms := r.FormValue("terms")
	if terms != "on" {
		errors["terms"] = []string{"You must agree to the Terms of Service and Privacy Policy"}
	}

	if len(errors) > 0 {
		// Debug: log validation errors
		fmt.Printf("Registration validation failed for %s. Errors: %+v\n", email, errors)
		fmt.Printf("Form data received: email=%s, firstName=%s, lastName=%s, password_len=%d, passwordConfirm_len=%d, terms=%s, role=%s\n", 
			email, firstName, lastName, len(password), len(passwordConfirm), terms, roleValue)
		
		component := pages.RegisterPage(nil, errors, formData)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render registration page", http.StatusInternalServerError)
		}
		return
	}

	// Attempt registration
	registerReq := &services.RegisterRequest{
		Email:     email,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
		Role:      role,
	}

	fmt.Printf("Attempting registration for: %s\n", email)
	authResponse, err := h.authService.Register(registerReq)
	if err != nil {
		fmt.Printf("Registration failed for %s: %v\n", email, err)
		if strings.Contains(err.Error(), "already exists") {
			errors["email"] = []string{"An account with this email already exists"}
		} else {
			errors["general"] = []string{"Registration failed. Please try again."}
		}
		
		component := pages.RegisterPage(nil, errors, formData)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render registration page", http.StatusInternalServerError)
		}
		return
	}

	// For email verification flow, don't create session immediately
	// Instead, show a message asking user to check their email
	if authResponse.SessionID == "" {
		fmt.Printf("Registration successful for %s, showing verification page\n", email)
		// Show email verification required page
		component := pages.RegistrationSuccessPage(email)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render registration success page", http.StatusInternalServerError)
		}
		return
	}

	// If no email verification required, create session and redirect
	session, err := h.store.Get(r, "session")
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	session.Values["session_id"] = authResponse.SessionID
	session.Values["user_id"] = authResponse.User.ID
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Redirect to dashboard
	w.Header().Set("HX-Redirect", "/dashboard")
	w.WriteHeader(http.StatusOK)
}

// ForgotPasswordPage renders the forgot password page
func (h *AuthHandler) ForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	
	// If user is already logged in, redirect to dashboard
	if user != nil {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	// Render forgot password page
	component := pages.ForgotPasswordPage(nil, make(map[string][]string), make(map[string]string), false)
	err := component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render forgot password page", http.StatusInternalServerError)
		return
	}
}

// ForgotPasswordSubmit handles forgot password form submission
func (h *AuthHandler) ForgotPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))

	// Validate input
	errors := make(map[string][]string)
	formData := map[string]string{
		"email": email,
	}

	if email == "" {
		errors["email"] = []string{"Email is required"}
	}

	if len(errors) > 0 {
		component := pages.ForgotPasswordPage(nil, errors, formData, false)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render forgot password page", http.StatusInternalServerError)
		}
		return
	}

	// Request password reset
	resetReq := &services.PasswordResetRequest{
		Email: email,
	}

	err := h.authService.RequestPasswordReset(resetReq)
	if err != nil {
		// Don't reveal whether the email exists or not for security
		// Always show success message
	}

	// Show success message
	component := pages.ForgotPasswordPage(nil, make(map[string][]string), formData, true)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render forgot password page", http.StatusInternalServerError)
		return
	}
}

// VerifyEmail handles email verification
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	// Get token from query string
	token := r.URL.Query().Get("token")
	if token == "" {
		component := pages.VerificationErrorPage("Verification token is required")
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render verification error page", http.StatusInternalServerError)
		}
		return
	}

	// Verify the email
	user, err := h.authService.VerifyEmail(token)
	if err != nil {
		// Show error page
		component := pages.VerificationErrorPage(err.Error())
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render verification error page", http.StatusInternalServerError)
		}
		return
	}

	// Automatically sign in the user after successful verification
	// For now, we'll create the session manually since we don't have a direct CreateSession method
	// This is a simplified approach - in production, you'd want a proper method for this
	sessionID := fmt.Sprintf("session_%d_%d", user.ID, time.Now().Unix())

	// Create session
	session, err := h.store.Get(r, "session")
	if err != nil {
		fmt.Printf("Failed to get session after email verification: %v\n", err)
		component := pages.VerificationSuccessPage(user.FirstName)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render verification success page", http.StatusInternalServerError)
		}
		return
	}

	session.Values["session_id"] = sessionID
	session.Values["user_id"] = user.ID
	err = session.Save(r, w)
	if err != nil {
		fmt.Printf("Failed to save session after email verification: %v\n", err)
	}

	// Initialize onboarding for the user
	onboarding, err := h.onboardingService.InitializeOnboarding(user.ID, user.Role)
	if err != nil {
		fmt.Printf("Failed to initialize onboarding for user %d: %v\n", user.ID, err)
	} else {
		err = h.onboardingService.SaveOnboardingToSession(w, r, onboarding)
		if err != nil {
			fmt.Printf("Failed to save onboarding to session: %v\n", err)
		}
	}

	// Show success page
	component := pages.VerificationSuccessPage(user.FirstName)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render verification success page", http.StatusInternalServerError)
	}
}

// getPostVerificationRedirectURL determines where to redirect user after email verification
func (h *AuthHandler) getPostVerificationRedirectURL(user *models.User, onboarding *models.UserOnboarding) string {
	// Check if user has completed onboarding
	if onboarding != nil && !h.onboardingService.IsOnboardingCompleted(onboarding) {
		return "/onboarding"
	}

	// Redirect based on user role
	switch user.Role {
	case models.RoleAdmin:
		return "/admin/dashboard"
	case models.RoleModerator:
		return "/moderator/dashboard"
	case models.RoleOrganizer:
		return "/organizer/dashboard"
	default:
		return "/dashboard"
	}
}

// ResendVerificationPage shows the resend verification page
func (h *AuthHandler) ResendVerificationPage(w http.ResponseWriter, r *http.Request) {
	component := pages.ResendVerificationPage(nil, make(map[string][]string), make(map[string]string))
	err := component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render resend verification page", http.StatusInternalServerError)
	}
}

// ResendVerificationSubmit resends the verification email
func (h *AuthHandler) ResendVerificationSubmit(w http.ResponseWriter, r *http.Request) {
	// Parse form
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get email from form
	email := strings.TrimSpace(r.FormValue("email"))
	
	// Validate input
	errors := make(map[string][]string)
	formData := map[string]string{
		"email": email,
	}

	if email == "" {
		errors["email"] = []string{"Email is required"}
	} else if !isValidEmail(email) {
		errors["email"] = []string{"Please enter a valid email address"}
	}

	if len(errors) > 0 {
		component := pages.ResendVerificationPage(nil, errors, formData)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render resend verification page", http.StatusInternalServerError)
		}
		return
	}

	// Resend verification email
	err = h.authService.ResendVerificationEmail(email)
	if err != nil {
		// Check if this is a "not found" or "already verified" error
		if strings.Contains(err.Error(), "not found") {
			errors["email"] = []string{"No account found with this email address"}
		} else if strings.Contains(err.Error(), "already verified") {
			errors["email"] = []string{"This email is already verified"}
		} else if strings.Contains(err.Error(), "wait at least") {
			errors["email"] = []string{"Please wait at least 5 minutes before requesting another verification email"}
		} else {
			errors["email"] = []string{"Failed to resend verification email"}
		}

		// Return form with errors
		component := pages.ResendVerificationPage(nil, errors, formData)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render resend verification page", http.StatusInternalServerError)
		}
		return
	}

	// Show success page
	component := pages.ResendVerificationSuccessPage(email)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render resend verification success page", http.StatusInternalServerError)
	}
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	session, err := h.store.Get(r, "session")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get session ID for cleanup
	if sessionID, ok := session.Values["session_id"].(string); ok && sessionID != "" {
		// Clean up session in the database
		err := h.authService.Logout(sessionID)
		if err != nil {
			// Log error but don't fail the logout
		}
	}

	// Clear session
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	err = session.Save(r, w)
	if err != nil {
		// Log error but continue with redirect
	}

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// LogoutAll handles logging out from all sessions
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Logout from all sessions
	if logoutAllService, ok := h.authService.(interface{ LogoutAllSessions(int) error }); ok {
		err := logoutAllService.LogoutAllSessions(user.ID)
		if err != nil {
			// Log error but continue
		}
	}

	// Clear current session
	session, err := h.store.Get(r, "session")
	if err == nil {
		session.Values = make(map[interface{}]interface{})
		session.Options.MaxAge = -1
		session.Save(r, w)
	}

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// GetCSRFToken returns the CSRF token for the current session
func (h *AuthHandler) GetCSRFToken(w http.ResponseWriter, r *http.Request) {
	token := h.getCSRFToken(w, r)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"csrf_token":"%s"}`, token)))
}

// ResetPasswordPage renders the password reset page
func (h *AuthHandler) ResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	// Get token from query string
	token := r.URL.Query().Get("token")
	if token == "" {
		component := pages.PasswordResetErrorPage("Reset token is required")
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render password reset error page", http.StatusInternalServerError)
		}
		return
	}

	// Validate the token
	_, err := h.authService.ValidatePasswordResetToken(token)
	if err != nil {
		component := pages.PasswordResetErrorPage("Invalid or expired reset token")
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render password reset error page", http.StatusInternalServerError)
		}
		return
	}

	// Show password reset form
	formData := map[string]string{"token": token}
	component := pages.ResetPasswordPage(nil, make(map[string][]string), formData)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render password reset page", http.StatusInternalServerError)
	}
}

// ResetPasswordSubmit handles password reset form submission
func (h *AuthHandler) ResetPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	// Parse form
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get form data
	token := r.FormValue("token")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate input
	errors := make(map[string][]string)
	formData := map[string]string{
		"token": token,
	}

	if token == "" {
		errors["token"] = []string{"Reset token is required"}
	}

	if newPassword == "" {
		errors["new_password"] = []string{"New password is required"}
	} else if len(newPassword) < 8 {
		errors["new_password"] = []string{"Password must be at least 8 characters long"}
	} else if len(newPassword) > 128 {
		errors["new_password"] = []string{"Password must be less than 128 characters"}
	}

	if confirmPassword == "" {
		errors["confirm_password"] = []string{"Password confirmation is required"}
	} else if newPassword != confirmPassword {
		errors["confirm_password"] = []string{"Passwords do not match"}
	}

	if len(errors) > 0 {
		component := pages.ResetPasswordPage(nil, errors, formData)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render password reset page", http.StatusInternalServerError)
		}
		return
	}

	// Complete password reset
	resetReq := &services.PasswordResetCompleteRequest{
		Token:       token,
		NewPassword: newPassword,
	}

	err = h.authService.CompletePasswordReset(resetReq)
	if err != nil {
		if strings.Contains(err.Error(), "invalid or expired") {
			errors["token"] = []string{"Invalid or expired reset token"}
		} else if strings.Contains(err.Error(), "password must be") {
			errors["new_password"] = []string{err.Error()}
		} else {
			errors["general"] = []string{"Failed to reset password. Please try again."}
		}

		component := pages.ResetPasswordPage(nil, errors, formData)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render password reset page", http.StatusInternalServerError)
		}
		return
	}

	// Show success page
	component := pages.PasswordResetSuccessPage()
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render password reset success page", http.StatusInternalServerError)
	}
}