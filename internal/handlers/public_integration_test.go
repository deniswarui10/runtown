package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"event-ticketing-platform/internal/services"

	"github.com/go-chi/chi/v5"
)

func TestPublicHandler_HomePage_Integration(t *testing.T) {
	// Setup
	eventService := &services.MockEventService{}
	ticketService := &services.MockTicketService{}
	handler := NewPublicHandler(eventService, ticketService)

	// Create request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.HomePage(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Discover Amazing Events") {
		t.Error("Expected homepage to contain hero text")
	}

	if !strings.Contains(body, "Tech Conference 2024") {
		t.Error("Expected homepage to contain featured events")
	}

	if !strings.Contains(body, "Business Workshop") {
		t.Error("Expected homepage to contain upcoming events")
	}
}

func TestPublicHandler_EventsListPage_Integration(t *testing.T) {
	// Setup
	eventService := &services.MockEventService{}
	ticketService := &services.MockTicketService{}
	handler := NewPublicHandler(eventService, ticketService)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedText   []string
	}{
		{
			name:           "Basic events list",
			url:            "/events",
			expectedStatus: http.StatusOK,
			expectedText:   []string{"Browse Events", "Tech Conference 2024", "Music Festival Summer"},
		},
		{
			name:           "Search events",
			url:            "/events?q=tech",
			expectedStatus: http.StatusOK,
			expectedText:   []string{"Tech Conference 2024"},
		},
		{
			name:           "Pagination",
			url:            "/events?page=1",
			expectedStatus: http.StatusOK,
			expectedText:   []string{"Browse Events"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			// Execute
			handler.EventsListPage(w, req)

			// Assert status
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Assert content
			body := w.Body.String()
			for _, expectedText := range tt.expectedText {
				if !strings.Contains(body, expectedText) {
					t.Errorf("Expected response to contain '%s'", expectedText)
				}
			}
		})
	}
}

func TestPublicHandler_EventsListPage_HTMX_Integration(t *testing.T) {
	// Setup
	eventService := &services.MockEventService{}
	ticketService := &services.MockTicketService{}
	handler := NewPublicHandler(eventService, ticketService)

	// Create HTMX request
	req := httptest.NewRequest("GET", "/events?q=tech", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	// Execute
	handler.EventsListPage(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	// Should return partial content, not full page
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("HTMX request should not return full HTML page")
	}

	if !strings.Contains(body, "Tech Conference 2024") {
		t.Error("Expected HTMX response to contain filtered events")
	}
}

func TestPublicHandler_EventDetailsPage_Integration(t *testing.T) {
	// Setup
	eventService := &services.MockEventService{}
	ticketService := &services.MockTicketService{}
	handler := NewPublicHandler(eventService, ticketService)

	// Create router to test URL parameters
	r := chi.NewRouter()
	r.Get("/events/{id}", handler.EventDetailsPage)

	tests := []struct {
		name           string
		eventID        string
		expectedStatus int
		expectedText   []string
	}{
		{
			name:           "Valid event ID",
			eventID:        "1",
			expectedStatus: http.StatusOK,
			expectedText:   []string{"Tech Conference 2024", "General Admission", "VIP Access"},
		},
		{
			name:           "Invalid event ID",
			eventID:        "abc",
			expectedStatus: http.StatusBadRequest,
			expectedText:   []string{},
		},
		{
			name:           "Non-existent event",
			eventID:        "999",
			expectedStatus: http.StatusNotFound,
			expectedText:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", "/events/"+tt.eventID, nil)
			w := httptest.NewRecorder()

			// Execute
			r.ServeHTTP(w, req)

			// Assert status
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Assert content for successful requests
			if tt.expectedStatus == http.StatusOK {
				body := w.Body.String()
				for _, expectedText := range tt.expectedText {
					if !strings.Contains(body, expectedText) {
						t.Errorf("Expected response to contain '%s'", expectedText)
					}
				}
			}
		})
	}
}

func TestPublicHandler_SearchEvents_Integration(t *testing.T) {
	// Setup
	eventService := &services.MockEventService{}
	ticketService := &services.MockTicketService{}
	handler := NewPublicHandler(eventService, ticketService)

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		expectedText   []string
	}{
		{
			name:           "Search with results",
			query:          "tech",
			expectedStatus: http.StatusOK,
			expectedText:   []string{"Tech Conference 2024"},
		},
		{
			name:           "Search with no results",
			query:          "nonexistent",
			expectedStatus: http.StatusOK,
			expectedText:   []string{},
		},
		{
			name:           "Empty search",
			query:          "",
			expectedStatus: http.StatusOK,
			expectedText:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", "/search?q="+tt.query, nil)
			w := httptest.NewRecorder()

			// Execute
			handler.SearchEvents(w, req)

			// Assert status
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Assert content for searches with results
			if len(tt.expectedText) > 0 {
				body := w.Body.String()
				for _, expectedText := range tt.expectedText {
					if !strings.Contains(body, expectedText) {
						t.Errorf("Expected response to contain '%s'", expectedText)
					}
				}
			}
		})
	}
}

func TestPublicHandler_SearchEvents_HTMX_Integration(t *testing.T) {
	// Setup
	eventService := &services.MockEventService{}
	ticketService := &services.MockTicketService{}
	handler := NewPublicHandler(eventService, ticketService)

	// Create HTMX search request
	req := httptest.NewRequest("GET", "/search?q=tech", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()

	// Execute
	handler.SearchEvents(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	// Should return partial content for search results
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("HTMX search should not return full HTML page")
	}

	if !strings.Contains(body, "Tech Conference 2024") {
		t.Error("Expected search results to contain matching events")
	}
}

func TestPublicHandler_EventFiltering_Integration(t *testing.T) {
	// Setup
	eventService := &services.MockEventService{}
	ticketService := &services.MockTicketService{}
	handler := NewPublicHandler(eventService, ticketService)

	tests := []struct {
		name           string
		filters        string
		expectedStatus int
		shouldContain  []string
	}{
		{
			name:           "Filter by location",
			filters:        "location=San%20Francisco",
			expectedStatus: http.StatusOK,
			shouldContain:  []string{"Tech Conference 2024"},
		},
		{
			name:           "Filter by category",
			filters:        "category=technology",
			expectedStatus: http.StatusOK,
			shouldContain:  []string{},
		},
		{
			name:           "Multiple filters",
			filters:        "q=tech&location=San%20Francisco",
			expectedStatus: http.StatusOK,
			shouldContain:  []string{"Tech Conference 2024"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", "/events?"+tt.filters, nil)
			w := httptest.NewRecorder()

			// Execute
			handler.EventsListPage(w, req)

			// Assert status
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Assert content
			body := w.Body.String()
			for _, expectedText := range tt.shouldContain {
				if !strings.Contains(body, expectedText) {
					t.Errorf("Expected response to contain '%s'", expectedText)
				}
			}
		})
	}
}