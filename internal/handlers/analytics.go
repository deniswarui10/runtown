package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/web/templates/pages"
)

// AnalyticsHandler handles analytics-related HTTP requests
type AnalyticsHandler struct {
	analyticsService services.AnalyticsServiceInterface
	authService      services.AuthServiceInterface
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(analyticsService services.AnalyticsServiceInterface, authService services.AuthServiceInterface) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
		authService:      authService,
	}
}

// OrganizerDashboard handles GET /organizer/dashboard
func (h *AnalyticsHandler) OrganizerDashboard(w http.ResponseWriter, r *http.Request) {
	// Get user from context (middleware should have loaded it)
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get dashboard data
	dashboard, err := h.analyticsService.GetOrganizerDashboard(user.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get dashboard data: %v", err), http.StatusInternalServerError)
		return
	}

	// Render dashboard template
	component := pages.OrganizerDashboard(user, dashboard)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to render template: %v", err), http.StatusInternalServerError)
		return
	}
}

// EventAnalytics handles GET /organizer/events/{id}/analytics
func (h *AnalyticsHandler) EventAnalytics(w http.ResponseWriter, r *http.Request) {
	// Get user from context (middleware should have loaded it)
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Get event analytics
	analytics, err := h.analyticsService.GetEventAnalytics(eventID, user.ID)
	if err != nil {
		if err.Error() == "organizer does not have access to this event" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get event analytics: %v", err), http.StatusInternalServerError)
		return
	}

	// Render analytics template
	component := pages.EventAnalytics(user, analytics)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to render template: %v", err), http.StatusInternalServerError)
		return
	}
}

// ExportAttendees handles GET /organizer/events/{id}/export-attendees
func (h *AnalyticsHandler) ExportAttendees(w http.ResponseWriter, r *http.Request) {
	// Get user from context (middleware should have loaded it)
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Export attendee data
	csvData, err := h.analyticsService.ExportAttendeeData(eventID, user.ID)
	if err != nil {
		if err.Error() == "organizer does not have access to this event" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to export attendee data: %v", err), http.StatusInternalServerError)
		return
	}

	// Set headers for CSV download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"event_%d_attendees.csv\"", eventID))
	w.Header().Set("Content-Length", strconv.Itoa(len(csvData)))

	// Write CSV data
	w.Write(csvData)
}

// DashboardAPI handles GET /api/organizer/dashboard (for HTMX requests)
func (h *AnalyticsHandler) DashboardAPI(w http.ResponseWriter, r *http.Request) {
	// Get user from context (middleware should have loaded it)
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get dashboard data
	dashboard, err := h.analyticsService.GetOrganizerDashboard(user.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get dashboard data: %v", err), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	if err := writeJSON(w, dashboard); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write JSON response: %v", err), http.StatusInternalServerError)
		return
	}
}

// EventAnalyticsAPI handles GET /api/organizer/events/{id}/analytics (for HTMX requests)
func (h *AnalyticsHandler) EventAnalyticsAPI(w http.ResponseWriter, r *http.Request) {
	// Get user from context (middleware should have loaded it)
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Get event analytics
	analytics, err := h.analyticsService.GetEventAnalytics(eventID, user.ID)
	if err != nil {
		if err.Error() == "organizer does not have access to this event" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get event analytics: %v", err), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	if err := writeJSON(w, analytics); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write JSON response: %v", err), http.StatusInternalServerError)
		return
	}
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}