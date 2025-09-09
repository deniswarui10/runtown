package models

import "time"

// AuthFlowStep represents a step in the authentication flow
type AuthFlowStep struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Completed   bool   `json:"completed"`
	Component   string `json:"component"`
	Order       int    `json:"order"`
}

// AuthFlow represents the current state of a user's authentication flow
type AuthFlow struct {
	UserID        int                    `json:"user_id"`
	CurrentStep   string                 `json:"current_step"`
	NextStep      string                 `json:"next_step"`
	RedirectURL   string                 `json:"redirect_url"`
	Steps         []AuthFlowStep         `json:"steps"`
	Data          map[string]interface{} `json:"data"`
	ExpiresAt     time.Time              `json:"expires_at"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// OnboardingStep represents a step in the user onboarding process
type OnboardingStep struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Required    bool                   `json:"required"`
	Completed   bool                   `json:"completed"`
	Component   string                 `json:"component"`
	Order       int                    `json:"order"`
	Data        map[string]interface{} `json:"data"`
}

// UserOnboarding represents the current state of a user's onboarding
type UserOnboarding struct {
	UserID        int              `json:"user_id"`
	CurrentStep   int              `json:"current_step"`
	TotalSteps    int              `json:"total_steps"`
	Steps         []OnboardingStep `json:"steps"`
	CompletedAt   *time.Time       `json:"completed_at"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

// AuthFlowType represents different types of authentication flows
type AuthFlowType string

const (
	AuthFlowSignup         AuthFlowType = "signup"
	AuthFlowSignin         AuthFlowType = "signin"
	AuthFlowEmailVerify    AuthFlowType = "email_verify"
	AuthFlowPasswordReset  AuthFlowType = "password_reset"
	AuthFlowProfileSetup   AuthFlowType = "profile_setup"
)

// GetDefaultSignupSteps returns the default steps for signup flow
func GetDefaultSignupSteps() []AuthFlowStep {
	return []AuthFlowStep{
		{
			ID:          "basic_info",
			Title:       "Basic Information",
			Description: "Enter your email and create a password",
			Required:    true,
			Completed:   false,
			Component:   "BasicInfoForm",
			Order:       1,
		},
		{
			ID:          "personal_details",
			Title:       "Personal Details",
			Description: "Tell us your name and role",
			Required:    true,
			Completed:   false,
			Component:   "PersonalDetailsForm",
			Order:       2,
		},
		{
			ID:          "email_verification",
			Title:       "Email Verification",
			Description: "Verify your email address",
			Required:    true,
			Completed:   false,
			Component:   "EmailVerificationForm",
			Order:       3,
		},
		{
			ID:          "profile_setup",
			Title:       "Profile Setup",
			Description: "Complete your profile (optional)",
			Required:    false,
			Completed:   false,
			Component:   "ProfileSetupForm",
			Order:       4,
		},
	}
}

// GetDefaultOnboardingSteps returns the default onboarding steps based on user role
func GetDefaultOnboardingSteps(role UserRole) []OnboardingStep {
	baseSteps := []OnboardingStep{
		{
			ID:          "welcome",
			Title:       "Welcome",
			Description: "Welcome to the platform",
			Required:    true,
			Completed:   false,
			Component:   "WelcomeStep",
			Order:       1,
		},
		{
			ID:          "profile_completion",
			Title:       "Complete Profile",
			Description: "Add additional profile information",
			Required:    false,
			Completed:   false,
			Component:   "ProfileCompletionStep",
			Order:       2,
		},
	}

	// Add role-specific steps
	switch role {
	case RoleOrganizer:
		baseSteps = append(baseSteps, []OnboardingStep{
			{
				ID:          "organizer_verification",
				Title:       "Organizer Verification",
				Description: "Verify your identity as an event organizer",
				Required:    true,
				Completed:   false,
				Component:   "OrganizerVerificationStep",
				Order:       3,
			},
			{
				ID:          "payment_setup",
				Title:       "Payment Setup",
				Description: "Set up your payment methods for withdrawals",
				Required:    false,
				Completed:   false,
				Component:   "PaymentSetupStep",
				Order:       4,
			},
			{
				ID:          "first_event",
				Title:       "Create Your First Event",
				Description: "Get started by creating your first event",
				Required:    false,
				Completed:   false,
				Component:   "FirstEventStep",
				Order:       5,
			},
		}...)
	case RoleAttendee:
		baseSteps = append(baseSteps, []OnboardingStep{
			{
				ID:          "preferences",
				Title:       "Event Preferences",
				Description: "Tell us what events you're interested in",
				Required:    false,
				Completed:   false,
				Component:   "PreferencesStep",
				Order:       3,
			},
			{
				ID:          "explore_events",
				Title:       "Explore Events",
				Description: "Discover events happening near you",
				Required:    false,
				Completed:   false,
				Component:   "ExploreEventsStep",
				Order:       4,
			},
		}...)
	}

	return baseSteps
}