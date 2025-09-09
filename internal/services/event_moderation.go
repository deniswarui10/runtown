package services

import (
	"fmt"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
)

// EventModerationService handles event moderation operations
type EventModerationService struct {
	eventRepo *repositories.EventRepository
	auditService *AuditService
}

// NewEventModerationService creates a new event moderation service
func NewEventModerationService(eventRepo *repositories.EventRepository, auditService *AuditService) *EventModerationService {
	return &EventModerationService{
		eventRepo: eventRepo,
		auditService: auditService,
	}
}

// GetPendingEvents retrieves events that are pending review
func (s *EventModerationService) GetPendingEvents(page, limit int) ([]*models.Event, int, error) {
	offset := (page - 1) * limit
	return s.eventRepo.GetPendingEvents(limit, offset)
}

// ApproveEvent approves an event for publication
func (s *EventModerationService) ApproveEvent(eventID int, reviewerID int, r interface{}) error {
	// Get event details for audit log
	event, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	// Approve the event
	err = s.eventRepo.ApproveEvent(eventID, reviewerID)
	if err != nil {
		return err
	}

	// Log the action
	auditDetails := map[string]interface{}{
		"event_id": eventID,
		"event_title": event.Title,
		"organizer_id": event.OrganizerID,
	}

	if httpReq, ok := r.(*interface{}); ok && httpReq != nil {
		// Type assertion failed, but we'll continue without HTTP request context
		_ = s.auditService.LogAction(reviewerID, models.AuditActionEventApprove, models.AuditTargetEvent, eventID, auditDetails, nil)
	}

	return nil
}

// RejectEvent rejects an event with a reason
func (s *EventModerationService) RejectEvent(eventID int, reviewerID int, reason string, r interface{}) error {
	// Get event details for audit log
	event, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	// Reject the event
	err = s.eventRepo.RejectEvent(eventID, reviewerID, reason)
	if err != nil {
		return err
	}

	// Log the action
	auditDetails := map[string]interface{}{
		"event_id": eventID,
		"event_title": event.Title,
		"organizer_id": event.OrganizerID,
		"rejection_reason": reason,
	}

	if httpReq, ok := r.(*interface{}); ok && httpReq != nil {
		// Type assertion failed, but we'll continue without HTTP request context
		_ = s.auditService.LogAction(reviewerID, models.AuditActionEventReject, models.AuditTargetEvent, eventID, auditDetails, nil)
	}

	return nil
}

// SubmitForReview submits an event for admin review
func (s *EventModerationService) SubmitForReview(eventID int, organizerID int) error {
	return s.eventRepo.SubmitForReview(eventID, organizerID)
}

// GetEventModerationHistory retrieves the moderation history for an event
func (s *EventModerationService) GetEventModerationHistory(eventID int) (*models.Event, error) {
	return s.eventRepo.GetEventModerationHistory(eventID)
}

// CanUserModerateEvent checks if a user can moderate a specific event
func (s *EventModerationService) CanUserModerateEvent(userID int, userRole models.UserRole) bool {
	return userRole == models.UserRoleAdmin
}

// CanUserSubmitForReview checks if a user can submit an event for review
func (s *EventModerationService) CanUserSubmitForReview(eventID int, userID int, userRole models.UserRole) (bool, error) {
	if userRole != models.UserRoleOrganizer {
		return false, nil
	}

	// Check if the user owns the event
	event, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return false, err
	}

	return event.OrganizerID == userID && event.Status == models.StatusDraft, nil
}