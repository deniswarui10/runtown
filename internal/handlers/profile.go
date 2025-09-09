package handlers

import (
	"net/http"
	"strings"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/web/templates/pages"

	"github.com/gorilla/sessions"
)

// ProfileHandler handles profile-related requests
type ProfileHandler struct {
	authService services.AuthServiceInterface
	userService services.UserServiceInterface
	store       sessions.Store
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(authService services.AuthServiceInterface, userService services.UserServiceInterface, store sessions.Store) *ProfileHandler {
	return &ProfileHandler{
		authService: authService,
		userService: userService,
		store:       store,
	}
}

// ProfilePage renders the profile editing page
func (h *ProfileHandler) ProfilePage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Render profile page
	component := pages.ProfilePage(user, make(map[string][]string), make(map[string]string), false)
	err := component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render profile page", http.StatusInternalServerError)
		return
	}
}

// UpdateProfile handles profile update form submission
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	firstName := strings.TrimSpace(r.FormValue("first_name"))
	lastName := strings.TrimSpace(r.FormValue("last_name"))
	email := strings.TrimSpace(r.FormValue("email"))

	// Validate input
	errors := make(map[string][]string)
	formData := map[string]string{
		"first_name": firstName,
		"last_name":  lastName,
		"email":      email,
	}

	if firstName == "" {
		errors["first_name"] = []string{"First name is required"}
	} else if len(firstName) > 100 {
		errors["first_name"] = []string{"First name must be less than 100 characters"}
	}

	if lastName == "" {
		errors["last_name"] = []string{"Last name is required"}
	} else if len(lastName) > 100 {
		errors["last_name"] = []string{"Last name must be less than 100 characters"}
	}

	if email == "" {
		errors["email"] = []string{"Email is required"}
	} else if !isValidEmail(email) {
		errors["email"] = []string{"Please enter a valid email address"}
	}

	if len(errors) > 0 {
		component := pages.ProfilePage(user, errors, formData, false)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render profile page", http.StatusInternalServerError)
		}
		return
	}

	// Update profile
	updateReq := &services.UpdateProfileRequest{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}

	updatedUser, err := h.userService.UpdateProfile(user.ID, updateReq)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			errors["email"] = []string{"An account with this email already exists"}
		} else {
			errors["general"] = []string{"Failed to update profile. Please try again."}
		}

		component := pages.ProfilePage(user, errors, formData, false)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render profile page", http.StatusInternalServerError)
		}
		return
	}

	// Show success message
	component := pages.ProfilePage(updatedUser, make(map[string][]string), make(map[string]string), true)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render profile page", http.StatusInternalServerError)
		return
	}
}

// SecurityPage renders the security settings page
func (h *ProfileHandler) SecurityPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Render security page
	component := pages.SecurityPage(user, make(map[string][]string), make(map[string]string), false)
	err := component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render security page", http.StatusInternalServerError)
		return
	}
}

// ChangePassword handles password change form submission
func (h *ProfileHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate input
	errors := make(map[string][]string)
	formData := make(map[string]string)

	if currentPassword == "" {
		errors["current_password"] = []string{"Current password is required"}
	}

	if newPassword == "" {
		errors["new_password"] = []string{"New password is required"}
	} else if len(newPassword) < 8 {
		errors["new_password"] = []string{"New password must be at least 8 characters long"}
	}

	if confirmPassword == "" {
		errors["confirm_password"] = []string{"Password confirmation is required"}
	} else if newPassword != confirmPassword {
		errors["confirm_password"] = []string{"Passwords do not match"}
	}

	if len(errors) > 0 {
		component := pages.SecurityPage(user, errors, formData, false)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render security page", http.StatusInternalServerError)
		}
		return
	}

	// Change password
	changeReq := &services.PasswordChangeRequest{
		OldPassword: currentPassword,
		NewPassword: newPassword,
	}

	err := h.authService.ChangePassword(user.ID, changeReq)
	if err != nil {
		if strings.Contains(err.Error(), "current password is incorrect") {
			errors["current_password"] = []string{"Current password is incorrect"}
		} else {
			errors["general"] = []string{"Failed to change password. Please try again."}
		}

		component := pages.SecurityPage(user, errors, formData, false)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render security page", http.StatusInternalServerError)
		}
		return
	}

	// Show success message
	component := pages.SecurityPage(user, make(map[string][]string), make(map[string]string), true)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render security page", http.StatusInternalServerError)
		return
	}
}

// SettingsPage renders the account settings page
func (h *ProfileHandler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Get user preferences (for now, we'll use default values)
	preferences := &services.UserPreferences{
		EmailNotifications:    true,
		EventReminders:        true,
		MarketingEmails:       false,
		SMSNotifications:      false,
		NewsletterSubscription: false,
	}

	// Render settings page
	component := pages.SettingsPage(user, preferences, make(map[string][]string), false)
	err := component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render settings page", http.StatusInternalServerError)
		return
	}
}

// UpdateSettings handles settings update form submission
func (h *ProfileHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Parse preferences from form
	preferences := &services.UserPreferences{
		EmailNotifications:     r.FormValue("email_notifications") == "on",
		EventReminders:         r.FormValue("event_reminders") == "on",
		MarketingEmails:        r.FormValue("marketing_emails") == "on",
		SMSNotifications:       r.FormValue("sms_notifications") == "on",
		NewsletterSubscription: r.FormValue("newsletter_subscription") == "on",
	}

	// Update preferences (for now, we'll just show success)
	// In a real implementation, you would save these to the database
	err := h.userService.UpdatePreferences(user.ID, preferences)
	if err != nil {
		errors := map[string][]string{
			"general": {"Failed to update settings. Please try again."},
		}
		component := pages.SettingsPage(user, preferences, errors, false)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render settings page", http.StatusInternalServerError)
		}
		return
	}

	// Show success message
	component := pages.SettingsPage(user, preferences, make(map[string][]string), true)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render settings page", http.StatusInternalServerError)
		return
	}
}

// DeleteAccountPage renders the account deletion confirmation page
func (h *ProfileHandler) DeleteAccountPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Render delete account page
	component := pages.DeleteAccountPage(user, make(map[string][]string), make(map[string]string))
	err := component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render delete account page", http.StatusInternalServerError)
		return
	}
}

// DeleteAccount handles account deletion form submission
func (h *ProfileHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")
	confirmation := r.FormValue("confirmation")

	// Validate input
	errors := make(map[string][]string)
	formData := map[string]string{
		"confirmation": confirmation,
	}

	if password == "" {
		errors["password"] = []string{"Password is required to delete your account"}
	}

	if confirmation != "DELETE" {
		errors["confirmation"] = []string{"Please type 'DELETE' to confirm account deletion"}
	}

	if len(errors) > 0 {
		component := pages.DeleteAccountPage(user, errors, formData)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render delete account page", http.StatusInternalServerError)
		}
		return
	}

	// Verify password
	loginReq := &services.LoginRequest{
		Email:    user.Email,
		Password: password,
	}

	_, err := h.authService.Login(loginReq)
	if err != nil {
		errors["password"] = []string{"Incorrect password"}
		component := pages.DeleteAccountPage(user, errors, formData)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render delete account page", http.StatusInternalServerError)
		}
		return
	}

	// Delete account
	err = h.userService.DeleteAccount(user.ID)
	if err != nil {
		errors["general"] = []string{"Failed to delete account. Please try again."}
		component := pages.DeleteAccountPage(user, errors, formData)
		w.WriteHeader(http.StatusUnprocessableEntity)
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render delete account page", http.StatusInternalServerError)
		}
		return
	}

	// Clear session
	session, err := h.store.Get(r, "session")
	if err == nil {
		session.Values = make(map[interface{}]interface{})
		session.Options.MaxAge = -1
		session.Save(r, w)
	}

	// Redirect to home page with success message
	http.Redirect(w, r, "/?deleted=1", http.StatusSeeOther)
}

