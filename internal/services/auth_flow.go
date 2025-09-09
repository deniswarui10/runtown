package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"event-ticketing-platform/internal/models"

	"github.com/gorilla/sessions"
)

// AuthFlowService handles authentication flow management
type AuthFlowService struct {
	store sessions.Store
}

// NewAuthFlowService creates a new authentication flow service
func NewAuthFlowService(store sessions.Store) *AuthFlowService {
	return &AuthFlowService{
		store: store,
	}
}

// InitializeSignupFlow initializes a new signup flow
func (s *AuthFlowService) InitializeSignupFlow(sessionID string, redirectURL string) (*models.AuthFlow, error) {
	flow := &models.AuthFlow{
		CurrentStep: "basic_info",
		NextStep:    "personal_details",
		RedirectURL: redirectURL,
		Steps:       models.GetDefaultSignupSteps(),
		Data:        make(map[string]interface{}),
		ExpiresAt:   time.Now().Add(24 * time.Hour), // Flow expires in 24 hours
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return flow, nil
}

// UpdateFlowStep updates a step in the authentication flow
func (s *AuthFlowService) UpdateFlowStep(flow *models.AuthFlow, stepID string, data map[string]interface{}) error {
	// Update step data
	for key, value := range data {
		flow.Data[key] = value
	}

	// Mark current step as completed
	for i, step := range flow.Steps {
		if step.ID == stepID {
			flow.Steps[i].Completed = true
			break
		}
	}

	// Determine next step
	nextStep := s.getNextStep(flow)
	flow.CurrentStep = nextStep
	flow.NextStep = s.getStepAfter(flow, nextStep)
	flow.UpdatedAt = time.Now()

	return nil
}

// GetNextStep determines the next step in the flow
func (s *AuthFlowService) getNextStep(flow *models.AuthFlow) string {
	for _, step := range flow.Steps {
		if !step.Completed && step.Required {
			return step.ID
		}
	}

	// All required steps completed, find first optional step
	for _, step := range flow.Steps {
		if !step.Completed {
			return step.ID
		}
	}

	return "completed"
}

// GetStepAfter gets the step that comes after the given step
func (s *AuthFlowService) getStepAfter(flow *models.AuthFlow, currentStepID string) string {
	foundCurrent := false
	for _, step := range flow.Steps {
		if foundCurrent && !step.Completed {
			return step.ID
		}
		if step.ID == currentStepID {
			foundCurrent = true
		}
	}
	return "completed"
}

// IsFlowCompleted checks if all required steps are completed
func (s *AuthFlowService) IsFlowCompleted(flow *models.AuthFlow) bool {
	for _, step := range flow.Steps {
		if step.Required && !step.Completed {
			return false
		}
	}
	return true
}

// SaveFlowToSession saves the authentication flow to session
func (s *AuthFlowService) SaveFlowToSession(w http.ResponseWriter, r *http.Request, flow *models.AuthFlow) error {
	session, err := s.store.Get(r, "session")
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	flowJSON, err := json.Marshal(flow)
	if err != nil {
		return fmt.Errorf("failed to marshal flow: %w", err)
	}

	session.Values["auth_flow"] = string(flowJSON)
	return session.Save(r, w)
}

// GetFlowFromSession retrieves the authentication flow from session
func (s *AuthFlowService) GetFlowFromSession(r *http.Request) (*models.AuthFlow, error) {
	session, err := s.store.Get(r, "session")
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	flowJSON, ok := session.Values["auth_flow"].(string)
	if !ok || flowJSON == "" {
		return nil, fmt.Errorf("no auth flow found in session")
	}

	var flow models.AuthFlow
	err = json.Unmarshal([]byte(flowJSON), &flow)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal flow: %w", err)
	}

	return &flow, nil
}

// ClearFlowFromSession removes the authentication flow from session
func (s *AuthFlowService) ClearFlowFromSession(w http.ResponseWriter, r *http.Request) error {
	session, err := s.store.Get(r, "session")
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	delete(session.Values, "auth_flow")
	return session.Save(r, w)
}

// OnboardingService handles user onboarding flow
type OnboardingService struct {
	store sessions.Store
}

// NewOnboardingService creates a new onboarding service
func NewOnboardingService(store sessions.Store) *OnboardingService {
	return &OnboardingService{
		store: store,
	}
}

// InitializeOnboarding initializes onboarding for a user
func (s *OnboardingService) InitializeOnboarding(userID int, role models.UserRole) (*models.UserOnboarding, error) {
	steps := models.GetDefaultOnboardingSteps(role)
	
	onboarding := &models.UserOnboarding{
		UserID:      userID,
		CurrentStep: 1,
		TotalSteps:  len(steps),
		Steps:       steps,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return onboarding, nil
}

// CompleteOnboardingStep marks a step as completed and advances to next
func (s *OnboardingService) CompleteOnboardingStep(onboarding *models.UserOnboarding, stepID string, data map[string]interface{}) error {
	// Find and complete the step
	for i, step := range onboarding.Steps {
		if step.ID == stepID {
			onboarding.Steps[i].Completed = true
			onboarding.Steps[i].Data = data
			break
		}
	}

	// Advance to next incomplete step
	for i, step := range onboarding.Steps {
		if !step.Completed {
			onboarding.CurrentStep = i + 1
			onboarding.UpdatedAt = time.Now()
			return nil
		}
	}

	// All steps completed
	now := time.Now()
	onboarding.CompletedAt = &now
	onboarding.CurrentStep = onboarding.TotalSteps
	onboarding.UpdatedAt = now

	return nil
}

// IsOnboardingCompleted checks if onboarding is completed
func (s *OnboardingService) IsOnboardingCompleted(onboarding *models.UserOnboarding) bool {
	return onboarding.CompletedAt != nil
}

// GetCurrentStep returns the current onboarding step
func (s *OnboardingService) GetCurrentStep(onboarding *models.UserOnboarding) *models.OnboardingStep {
	if onboarding.CurrentStep <= 0 || onboarding.CurrentStep > len(onboarding.Steps) {
		return nil
	}
	return &onboarding.Steps[onboarding.CurrentStep-1]
}

// SaveOnboardingToSession saves onboarding state to session
func (s *OnboardingService) SaveOnboardingToSession(w http.ResponseWriter, r *http.Request, onboarding *models.UserOnboarding) error {
	session, err := s.store.Get(r, "session")
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	onboardingJSON, err := json.Marshal(onboarding)
	if err != nil {
		return fmt.Errorf("failed to marshal onboarding: %w", err)
	}

	session.Values["user_onboarding"] = string(onboardingJSON)
	return session.Save(r, w)
}

// GetOnboardingFromSession retrieves onboarding state from session
func (s *OnboardingService) GetOnboardingFromSession(r *http.Request) (*models.UserOnboarding, error) {
	session, err := s.store.Get(r, "session")
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	onboardingJSON, ok := session.Values["user_onboarding"].(string)
	if !ok || onboardingJSON == "" {
		return nil, fmt.Errorf("no onboarding found in session")
	}

	var onboarding models.UserOnboarding
	err = json.Unmarshal([]byte(onboardingJSON), &onboarding)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal onboarding: %w", err)
	}

	return &onboarding, nil
}