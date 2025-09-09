package handlers

import (
	"fmt"
	"net/http"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"

	"github.com/gorilla/sessions"
)

// OnboardingHandler handles user onboarding flow
type OnboardingHandler struct {
	onboardingService *services.OnboardingService
	userService       services.UserServiceInterface
	store             sessions.Store
}

// NewOnboardingHandler creates a new onboarding handler
func NewOnboardingHandler(userService services.UserServiceInterface, store sessions.Store) *OnboardingHandler {
	return &OnboardingHandler{
		onboardingService: services.NewOnboardingService(store),
		userService:       userService,
		store:             store,
	}
}

// OnboardingPage displays the current onboarding step
func (h *OnboardingHandler) OnboardingPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		if middleware.IsHTMXRequest(r) {
			w.Header().Set("HX-Redirect", "/auth/login")
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		}
		return
	}

	// Get onboarding state from session
	onboarding, err := h.onboardingService.GetOnboardingFromSession(r)
	if err != nil {
		// Initialize onboarding if not found
		onboarding, err = h.onboardingService.InitializeOnboarding(user.ID, user.Role)
		if err != nil {
			http.Error(w, "Failed to initialize onboarding", http.StatusInternalServerError)
			return
		}
		
		err = h.onboardingService.SaveOnboardingToSession(w, r, onboarding)
		if err != nil {
			http.Error(w, "Failed to save onboarding state", http.StatusInternalServerError)
			return
		}
	}

	// Check if onboarding is already completed
	if h.onboardingService.IsOnboardingCompleted(onboarding) {
		redirectURL := h.getPostOnboardingRedirectURL(user)
		if middleware.IsHTMXRequest(r) {
			w.Header().Set("HX-Redirect", redirectURL)
			w.WriteHeader(http.StatusOK)
		} else {
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		}
		return
	}

	// Get current step
	currentStep := h.onboardingService.GetCurrentStep(onboarding)
	if currentStep == nil {
		http.Error(w, "Invalid onboarding state", http.StatusInternalServerError)
		return
	}

	// Render onboarding page
	// TODO: Implement OnboardingPage template
	w.Write([]byte(fmt.Sprintf("Onboarding Step: %s for user %s", currentStep.Title, user.FirstName)))
}

// OnboardingStepSubmit handles submission of an onboarding step
func (h *OnboardingHandler) OnboardingStepSubmit(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	stepID := r.FormValue("step_id")
	if stepID == "" {
		http.Error(w, "Step ID is required", http.StatusBadRequest)
		return
	}

	// Get onboarding state
	onboarding, err := h.onboardingService.GetOnboardingFromSession(r)
	if err != nil {
		http.Error(w, "Onboarding state not found", http.StatusBadRequest)
		return
	}

	// Process step data based on step type
	stepData := make(map[string]interface{})
	switch stepID {
	case "welcome":
		// Welcome step - just mark as completed
		stepData["completed_at"] = fmt.Sprintf("%d", r.Header.Get("X-Request-Time"))
	case "profile_completion":
		stepData["bio"] = r.FormValue("bio")
		stepData["phone"] = r.FormValue("phone")
		stepData["location"] = r.FormValue("location")
	case "organizer_verification":
		stepData["organization_name"] = r.FormValue("organization_name")
		stepData["organization_type"] = r.FormValue("organization_type")
		stepData["website"] = r.FormValue("website")
		stepData["description"] = r.FormValue("description")
	case "payment_setup":
		stepData["bank_name"] = r.FormValue("bank_name")
		stepData["account_number"] = r.FormValue("account_number")
		stepData["account_name"] = r.FormValue("account_name")
	case "preferences":
		stepData["event_categories"] = r.Form["event_categories"]
		stepData["notification_preferences"] = r.Form["notification_preferences"]
	}

	// Complete the step
	err = h.onboardingService.CompleteOnboardingStep(onboarding, stepID, stepData)
	if err != nil {
		http.Error(w, "Failed to complete step", http.StatusInternalServerError)
		return
	}

	// Save updated onboarding state
	err = h.onboardingService.SaveOnboardingToSession(w, r, onboarding)
	if err != nil {
		http.Error(w, "Failed to save onboarding state", http.StatusInternalServerError)
		return
	}

	// Check if onboarding is completed
	if h.onboardingService.IsOnboardingCompleted(onboarding) {
		// Update user profile completion status
		err = h.updateUserProfileCompletion(user.ID, onboarding)
		if err != nil {
			fmt.Printf("Failed to update user profile completion: %v\n", err)
		}

		// Redirect to appropriate dashboard
		redirectURL := h.getPostOnboardingRedirectURL(user)
		if middleware.IsHTMXRequest(r) {
			w.Header().Set("HX-Redirect", redirectURL)
			w.WriteHeader(http.StatusOK)
		} else {
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		}
		return
	}

	// Continue to next step
	if middleware.IsHTMXRequest(r) {
		// Return the next step content
		currentStep := h.onboardingService.GetCurrentStep(onboarding)
		// TODO: Implement OnboardingStepContent template
		w.Write([]byte(fmt.Sprintf("Next Step: %s", currentStep.Title)))
	} else {
		http.Redirect(w, r, "/onboarding", http.StatusSeeOther)
	}
}

// SkipOnboardingStep skips an optional onboarding step
func (h *OnboardingHandler) SkipOnboardingStep(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	stepID := r.URL.Query().Get("step_id")
	if stepID == "" {
		http.Error(w, "Step ID is required", http.StatusBadRequest)
		return
	}

	// Get onboarding state
	onboarding, err := h.onboardingService.GetOnboardingFromSession(r)
	if err != nil {
		http.Error(w, "Onboarding state not found", http.StatusBadRequest)
		return
	}

	// Mark step as completed with skip flag
	stepData := map[string]interface{}{
		"skipped": true,
	}
	
	err = h.onboardingService.CompleteOnboardingStep(onboarding, stepID, stepData)
	if err != nil {
		http.Error(w, "Failed to skip step", http.StatusInternalServerError)
		return
	}

	// Save updated state
	err = h.onboardingService.SaveOnboardingToSession(w, r, onboarding)
	if err != nil {
		http.Error(w, "Failed to save onboarding state", http.StatusInternalServerError)
		return
	}

	// Redirect to continue onboarding
	if middleware.IsHTMXRequest(r) {
		w.Header().Set("HX-Redirect", "/onboarding")
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, "/onboarding", http.StatusSeeOther)
	}
}

// getPostOnboardingRedirectURL determines where to redirect after onboarding completion
func (h *OnboardingHandler) getPostOnboardingRedirectURL(user *models.User) string {
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

// updateUserProfileCompletion updates the user's profile completion status
func (h *OnboardingHandler) updateUserProfileCompletion(userID int, onboarding *models.UserOnboarding) error {
	// Calculate completion percentage based on completed steps
	completedSteps := 0
	for _, step := range onboarding.Steps {
		if step.Completed {
			completedSteps++
		}
	}

	// Update user profile completion
	// This would typically involve calling a user service method
	// For now, we'll just log the completion
	fmt.Printf("User %d completed onboarding: %d/%d steps completed\n", 
		userID, completedSteps, len(onboarding.Steps))

	return nil
}