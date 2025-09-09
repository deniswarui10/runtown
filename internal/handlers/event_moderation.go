package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/web/templates/pages"
)

// EventModerationHandler handles event moderation requests
type EventModerationHandler struct {
	moderationService *services.EventModerationService
}

// NewEventModerationHandler creates a new event moderation handler
func NewEventModerationHandler(moderationService *services.EventModerationService) *EventModerationHandler {
	return &EventModerationHandler{
		moderationService: moderationService,
	}
}

// AdminEventModerationPage displays the admin event moderation page
func (h *EventModerationHandler) AdminEventModerationPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get page parameter
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	// Get pending events
	events, totalCount, err := h.moderationService.GetPendingEvents(page, 10)
	if err != nil {
		http.Error(w, "Failed to load pending events", http.StatusInternalServerError)
		return
	}

	// Calculate pagination
	totalPages := (totalCount + 9) / 10
	paginationInfo := map[string]interface{}{
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"TotalCount":  totalCount,
		"HasPrev":     page > 1,
		"HasNext":     page < totalPages,
		"PrevPage":    page - 1,
		"NextPage":    page + 1,
	}

	// Render admin event moderation page
	component := pages.AdminEventModerationPage(user, events, paginationInfo)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// ModerateEvent handles event approval/rejection
func (h *EventModerationHandler) ModerateEvent(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get event ID
	eventID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	action := r.FormValue("action")

	switch action {
	case "approve":
		err = h.moderationService.ApproveEvent(eventID, user.ID, r)
		if err != nil {
			http.Error(w, "Failed to approve event", http.StatusInternalServerError)
			return
		}
	case "reject":
		reason := r.FormValue("rejection_reason")
		if reason == "" {
			http.Error(w, "Rejection reason is required", http.StatusBadRequest)
			return
		}
		err = h.moderationService.RejectEvent(eventID, user.ID, reason, r)
		if err != nil {
			http.Error(w, "Failed to reject event", http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	// Redirect back to moderation page
	http.Redirect(w, r, "/admin/events/moderate", http.StatusSeeOther)
}

// SubmitEventForReview handles organizer submission of events for review
func (h *EventModerationHandler) SubmitEventForReview(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if user.Role != models.UserRoleOrganizer {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get event ID
	eventID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Check if user can submit this event for review
	canSubmit, err := h.moderationService.CanUserSubmitForReview(eventID, user.ID, user.Role)
	if err != nil {
		http.Error(w, "Failed to check permissions", http.StatusInternalServerError)
		return
	}

	if !canSubmit {
		http.Error(w, "You cannot submit this event for review", http.StatusForbidden)
		return
	}

	// Submit for review
	err = h.moderationService.SubmitForReview(eventID, user.ID)
	if err != nil {
		http.Error(w, "Failed to submit event for review", http.StatusInternalServerError)
		return
	}

	// Redirect back to organizer events page
	http.Redirect(w, r, "/organizer/events", http.StatusSeeOther)
}

// EventModerationHistory displays the moderation history for an event
func (h *EventModerationHandler) EventModerationHistory(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Get event ID
	eventID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Get event moderation history
	event, err := h.moderationService.GetEventModerationHistory(eventID)
	if err != nil {
		http.Error(w, "Failed to load event moderation history", http.StatusInternalServerError)
		return
	}

	// Check permissions - admin can see all, organizer can only see their own
	if user.Role != models.UserRoleAdmin && (user.Role != models.UserRoleOrganizer || event.OrganizerID != user.ID) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// For now, just return JSON response - you could create a template for this
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Event moderation history endpoint - template not implemented yet"}`))
}

// ModeratorDashboard displays the moderator dashboard
func (h *EventModerationHandler) ModeratorDashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if user.Role != models.UserRoleModerator && user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get pending events count
	pendingEvents, _, err := h.moderationService.GetPendingEvents(1, 0) // Just get count
	if err != nil {
		http.Error(w, "Failed to load pending events", http.StatusInternalServerError)
		return
	}

	// Create stats for the dashboard
	stats := map[string]interface{}{
		"PendingEvents": len(pendingEvents),
	}

	// Render moderator dashboard
	component := pages.ModeratorDashboard(user, stats)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}