package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper function to set user in context for testing
func setUserInContext(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, middleware.UserContextKey, user)
}

// MockTicketServiceForTicketTypes implements the TicketServiceInterface for testing
type MockTicketServiceForTicketTypes struct {
	mock.Mock
}

func (m *MockTicketServiceForTicketTypes) GetTicketTypesByEventID(eventID int) ([]*models.TicketType, error) {
	args := m.Called(eventID)
	return args.Get(0).([]*models.TicketType), args.Error(1)
}

func (m *MockTicketServiceForTicketTypes) GetTicketTypeByID(id int) (*models.TicketType, error) {
	args := m.Called(id)
	return args.Get(0).(*models.TicketType), args.Error(1)
}

func (m *MockTicketServiceForTicketTypes) CreateTicketType(req *models.TicketTypeCreateRequest) (*models.TicketType, error) {
	args := m.Called(req)
	return args.Get(0).(*models.TicketType), args.Error(1)
}

func (m *MockTicketServiceForTicketTypes) UpdateTicketType(id int, req *models.TicketTypeUpdateRequest) (*models.TicketType, error) {
	args := m.Called(id, req)
	return args.Get(0).(*models.TicketType), args.Error(1)
}

func (m *MockTicketServiceForTicketTypes) DeleteTicketType(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

// Implement other required methods with minimal functionality
func (m *MockTicketServiceForTicketTypes) GetTicketByID(id int) (*models.Ticket, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockTicketServiceForTicketTypes) GetTicketsByOrderID(orderID int) ([]*models.Ticket, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockTicketServiceForTicketTypes) GenerateTicketsPDF(tickets []*models.Ticket, event *models.Event, order *models.Order) ([]byte, error) {
	return nil, models.ErrNotImplemented
}

// MockEventServiceForTicketTypes implements the EventServiceInterface for testing
type MockEventServiceForTicketTypes struct {
	mock.Mock
}

func (m *MockEventServiceForTicketTypes) GetEventByID(id int) (*models.Event, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventServiceForTicketTypes) CanUserEditEvent(eventID, userID int) (bool, error) {
	args := m.Called(eventID, userID)
	return args.Bool(0), args.Error(1)
}

// Implement other required methods with minimal functionality
func (m *MockEventServiceForTicketTypes) GetFeaturedEvents(limit int) ([]*models.Event, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockEventServiceForTicketTypes) GetUpcomingEvents(limit int) ([]*models.Event, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockEventServiceForTicketTypes) SearchEvents(filters services.EventSearchFilters) ([]*models.Event, int, error) {
	return nil, 0, models.ErrNotImplemented
}

func (m *MockEventServiceForTicketTypes) GetCategories() ([]*models.Category, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockEventServiceForTicketTypes) GetEventOrganizer(eventID int) (*models.User, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockEventServiceForTicketTypes) CreateEvent(req *services.EventCreateRequest) (*models.Event, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockEventServiceForTicketTypes) UpdateEvent(id int, req *services.EventUpdateRequest) (*models.Event, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockEventServiceForTicketTypes) DeleteEvent(id int) error {
	return models.ErrNotImplemented
}

func (m *MockEventServiceForTicketTypes) GetEventsByOrganizer(organizerID int) ([]*models.Event, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockEventServiceForTicketTypes) CanUserDeleteEvent(eventID, userID int) (bool, error) {
	return false, models.ErrNotImplemented
}

func (m *MockEventServiceForTicketTypes) UpdateEventStatus(eventID int, status models.EventStatus, organizerID int) (*models.Event, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockEventServiceForTicketTypes) DuplicateEvent(eventID int, organizerID int, newTitle string, newStartDate, newEndDate time.Time) (*models.Event, error) {
	return nil, models.ErrNotImplemented
}

func TestTicketTypeHandler_TicketTypesPage(t *testing.T) {
	mockTicketService := new(MockTicketServiceForTicketTypes)
	mockEventService := new(MockEventServiceForTicketTypes)
	handler := NewTicketTypeHandler(mockTicketService, mockEventService)

	// Create test data
	testUser := &models.User{
		ID:    1,
		Email: "organizer@example.com",
		Role:  models.RoleOrganizer,
	}

	testEvent := &models.Event{
		ID:          1,
		Title:       "Test Event",
		OrganizerID: 1,
	}

	testTicketTypes := []*models.TicketType{
		{
			ID:          1,
			EventID:     1,
			Name:        "General Admission",
			Description: "Standard entry ticket",
			Price:       2500, // $25.00
			Quantity:    100,
			Sold:        25,
			SaleStart:   time.Now().Add(-1 * time.Hour),
			SaleEnd:     time.Now().Add(24 * time.Hour),
		},
		{
			ID:          2,
			EventID:     1,
			Name:        "VIP",
			Description: "Premium access with perks",
			Price:       5000, // $50.00
			Quantity:    50,
			Sold:        10,
			SaleStart:   time.Now().Add(-1 * time.Hour),
			SaleEnd:     time.Now().Add(24 * time.Hour),
		},
	}

	// Setup expectations
	mockEventService.On("CanUserEditEvent", 1, 1).Return(true, nil)
	mockEventService.On("GetEventByID", 1).Return(testEvent, nil)
	mockTicketService.On("GetTicketTypesByEventID", 1).Return(testTicketTypes, nil)

	// Create request
	req := httptest.NewRequest("GET", "/organizer/events/1/tickets", nil)
	req = req.WithContext(setUserInContext(req.Context(), testUser))

	// Create router and add route
	r := chi.NewRouter()
	r.Get("/organizer/events/{eventId}/tickets", handler.TicketTypesPage)

	// Execute request
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Test Event")
	assert.Contains(t, w.Body.String(), "General Admission")
	assert.Contains(t, w.Body.String(), "VIP")
	assert.Contains(t, w.Body.String(), "$25.00")
	assert.Contains(t, w.Body.String(), "$50.00")

	mockEventService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}

func TestTicketTypeHandler_CreateTicketTypePage(t *testing.T) {
	mockTicketService := new(MockTicketServiceForTicketTypes)
	mockEventService := new(MockEventServiceForTicketTypes)
	handler := NewTicketTypeHandler(mockTicketService, mockEventService)

	// Create test data
	testUser := &models.User{
		ID:    1,
		Email: "organizer@example.com",
		Role:  models.RoleOrganizer,
	}

	testEvent := &models.Event{
		ID:          1,
		Title:       "Test Event",
		OrganizerID: 1,
	}

	// Setup expectations
	mockEventService.On("CanUserEditEvent", 1, 1).Return(true, nil)
	mockEventService.On("GetEventByID", 1).Return(testEvent, nil)

	// Create request
	req := httptest.NewRequest("GET", "/organizer/events/1/tickets/create", nil)
	req = req.WithContext(setUserInContext(req.Context(), testUser))

	// Create router and add route
	r := chi.NewRouter()
	r.Get("/organizer/events/{eventId}/tickets/create", handler.CreateTicketTypePage)

	// Execute request
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Create Ticket Type")
	assert.Contains(t, w.Body.String(), "Test Event")
	assert.Contains(t, w.Body.String(), `name="name"`)
	assert.Contains(t, w.Body.String(), `name="price"`)
	assert.Contains(t, w.Body.String(), `name="quantity"`)

	mockEventService.AssertExpectations(t)
}

func TestTicketTypeHandler_CreateTicketTypeSubmit(t *testing.T) {
	mockTicketService := new(MockTicketServiceForTicketTypes)
	mockEventService := new(MockEventServiceForTicketTypes)
	handler := NewTicketTypeHandler(mockTicketService, mockEventService)

	// Create test data
	testUser := &models.User{
		ID:    1,
		Email: "organizer@example.com",
		Role:  models.RoleOrganizer,
	}

	testTicketType := &models.TicketType{
		ID:          1,
		EventID:     1,
		Name:        "General Admission",
		Description: "Standard entry ticket",
		Price:       2500, // $25.00
		Quantity:    100,
		Sold:        0,
		SaleStart:   time.Now().Add(1 * time.Hour),
		SaleEnd:     time.Now().Add(25 * time.Hour),
	}

	// Setup expectations
	mockEventService.On("CanUserEditEvent", 1, 1).Return(true, nil)
	mockTicketService.On("CreateTicketType", mock.AnythingOfType("*models.TicketTypeCreateRequest")).Return(testTicketType, nil)

	// Create form data
	formData := url.Values{}
	formData.Set("name", "General Admission")
	formData.Set("description", "Standard entry ticket")
	formData.Set("price", "25.00")
	formData.Set("quantity", "100")
	formData.Set("sale_start", time.Now().Add(1*time.Hour).Format("2006-01-02T15:04"))
	formData.Set("sale_end", time.Now().Add(25*time.Hour).Format("2006-01-02T15:04"))

	// Create request
	req := httptest.NewRequest("POST", "/organizer/events/1/tickets", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(setUserInContext(req.Context(), testUser))

	// Create router and add route
	r := chi.NewRouter()
	r.Post("/organizer/events/{eventId}/tickets", handler.CreateTicketTypeSubmit)

	// Execute request
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/organizer/events/1/tickets", w.Header().Get("Location"))

	mockEventService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}

func TestTicketTypeHandler_CreateTicketTypeSubmit_ValidationErrors(t *testing.T) {
	mockTicketService := new(MockTicketServiceForTicketTypes)
	mockEventService := new(MockEventServiceForTicketTypes)
	handler := NewTicketTypeHandler(mockTicketService, mockEventService)

	// Create test data
	testUser := &models.User{
		ID:    1,
		Email: "organizer@example.com",
		Role:  models.RoleOrganizer,
	}

	testEvent := &models.Event{
		ID:          1,
		Title:       "Test Event",
		OrganizerID: 1,
	}

	// Setup expectations
	mockEventService.On("CanUserEditEvent", 1, 1).Return(true, nil)
	mockEventService.On("GetEventByID", 1).Return(testEvent, nil)

	// Create form data with validation errors
	formData := url.Values{}
	formData.Set("name", "") // Missing name
	formData.Set("description", "Standard entry ticket")
	formData.Set("price", "-10.00") // Invalid price
	formData.Set("quantity", "0") // Invalid quantity
	formData.Set("sale_start", "")
	formData.Set("sale_end", "")

	// Create request
	req := httptest.NewRequest("POST", "/organizer/events/1/tickets", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(setUserInContext(req.Context(), testUser))

	// Create router and add route
	r := chi.NewRouter()
	r.Post("/organizer/events/{eventId}/tickets", handler.CreateTicketTypeSubmit)

	// Execute request
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code) // Should re-render form with errors
	assert.Contains(t, w.Body.String(), "Ticket type name is required")
	assert.Contains(t, w.Body.String(), "Sale start date is required")
	assert.Contains(t, w.Body.String(), "Sale end date is required")

	mockEventService.AssertExpectations(t)
}

func TestTicketTypeHandler_EditTicketTypePage(t *testing.T) {
	mockTicketService := new(MockTicketServiceForTicketTypes)
	mockEventService := new(MockEventServiceForTicketTypes)
	handler := NewTicketTypeHandler(mockTicketService, mockEventService)

	// Create test data
	testUser := &models.User{
		ID:    1,
		Email: "organizer@example.com",
		Role:  models.RoleOrganizer,
	}

	testEvent := &models.Event{
		ID:          1,
		Title:       "Test Event",
		OrganizerID: 1,
	}

	testTicketType := &models.TicketType{
		ID:          1,
		EventID:     1,
		Name:        "General Admission",
		Description: "Standard entry ticket",
		Price:       2500, // $25.00
		Quantity:    100,
		Sold:        25,
		SaleStart:   time.Now().Add(-1 * time.Hour),
		SaleEnd:     time.Now().Add(24 * time.Hour),
	}

	// Setup expectations
	mockEventService.On("CanUserEditEvent", 1, 1).Return(true, nil)
	mockEventService.On("GetEventByID", 1).Return(testEvent, nil)
	mockTicketService.On("GetTicketTypeByID", 1).Return(testTicketType, nil)

	// Create request
	req := httptest.NewRequest("GET", "/organizer/events/1/tickets/1/edit", nil)
	req = req.WithContext(setUserInContext(req.Context(), testUser))

	// Create router and add route
	r := chi.NewRouter()
	r.Get("/organizer/events/{eventId}/tickets/{id}/edit", handler.EditTicketTypePage)

	// Execute request
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Edit Ticket Type")
	assert.Contains(t, w.Body.String(), "General Admission")
	assert.Contains(t, w.Body.String(), "25.00") // Price should be displayed in dollars
	assert.Contains(t, w.Body.String(), "100") // Quantity
	assert.Contains(t, w.Body.String(), "25") // Sold tickets

	mockEventService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}

func TestTicketTypeHandler_UpdateTicketTypeSubmit(t *testing.T) {
	mockTicketService := new(MockTicketServiceForTicketTypes)
	mockEventService := new(MockEventServiceForTicketTypes)
	handler := NewTicketTypeHandler(mockTicketService, mockEventService)

	// Create test data
	testUser := &models.User{
		ID:    1,
		Email: "organizer@example.com",
		Role:  models.RoleOrganizer,
	}

	updatedTicketType := &models.TicketType{
		ID:          1,
		EventID:     1,
		Name:        "Updated General Admission",
		Description: "Updated description",
		Price:       3000, // $30.00
		Quantity:    120,
		Sold:        25,
		SaleStart:   time.Now().Add(1 * time.Hour),
		SaleEnd:     time.Now().Add(25 * time.Hour),
	}

	// Setup expectations
	mockEventService.On("CanUserEditEvent", 1, 1).Return(true, nil)
	mockTicketService.On("UpdateTicketType", 1, mock.AnythingOfType("*models.TicketTypeUpdateRequest")).Return(updatedTicketType, nil)

	// Create form data
	formData := url.Values{}
	formData.Set("name", "Updated General Admission")
	formData.Set("description", "Updated description")
	formData.Set("price", "30.00")
	formData.Set("quantity", "120")
	formData.Set("sale_start", time.Now().Add(1*time.Hour).Format("2006-01-02T15:04"))
	formData.Set("sale_end", time.Now().Add(25*time.Hour).Format("2006-01-02T15:04"))

	// Create request
	req := httptest.NewRequest("POST", "/organizer/events/1/tickets/1", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(setUserInContext(req.Context(), testUser))

	// Create router and add route
	r := chi.NewRouter()
	r.Post("/organizer/events/{eventId}/tickets/{id}", handler.UpdateTicketTypeSubmit)

	// Execute request
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/organizer/events/1/tickets?success=updated", w.Header().Get("Location"))

	mockEventService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}

func TestTicketTypeHandler_DeleteTicketType(t *testing.T) {
	mockTicketService := new(MockTicketServiceForTicketTypes)
	mockEventService := new(MockEventServiceForTicketTypes)
	handler := NewTicketTypeHandler(mockTicketService, mockEventService)

	// Create test data
	testUser := &models.User{
		ID:    1,
		Email: "organizer@example.com",
		Role:  models.RoleOrganizer,
	}

	testTicketType := &models.TicketType{
		ID:          1,
		EventID:     1,
		Name:        "General Admission",
		Description: "Standard entry ticket",
		Price:       2500, // $25.00
		Quantity:    100,
		Sold:        0, // No tickets sold, so deletion should be allowed
		SaleStart:   time.Now().Add(1 * time.Hour),
		SaleEnd:     time.Now().Add(25 * time.Hour),
	}

	// Setup expectations
	mockEventService.On("CanUserEditEvent", 1, 1).Return(true, nil)
	mockTicketService.On("GetTicketTypeByID", 1).Return(testTicketType, nil)
	mockTicketService.On("DeleteTicketType", 1).Return(nil)

	// Create request
	req := httptest.NewRequest("DELETE", "/organizer/events/1/tickets/1", nil)
	req = req.WithContext(setUserInContext(req.Context(), testUser))

	// Create router and add route
	r := chi.NewRouter()
	r.Delete("/organizer/events/{eventId}/tickets/{id}", handler.DeleteTicketType)

	// Execute request
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/organizer/events/1/tickets", w.Header().Get("Location"))

	mockEventService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}

func TestTicketTypeHandler_DeleteTicketType_WithSoldTickets(t *testing.T) {
	mockTicketService := new(MockTicketServiceForTicketTypes)
	mockEventService := new(MockEventServiceForTicketTypes)
	handler := NewTicketTypeHandler(mockTicketService, mockEventService)

	// Create test data
	testUser := &models.User{
		ID:    1,
		Email: "organizer@example.com",
		Role:  models.RoleOrganizer,
	}

	testTicketType := &models.TicketType{
		ID:          1,
		EventID:     1,
		Name:        "General Admission",
		Description: "Standard entry ticket",
		Price:       2500, // $25.00
		Quantity:    100,
		Sold:        25, // Has sold tickets, so deletion should be prevented
		SaleStart:   time.Now().Add(-1 * time.Hour),
		SaleEnd:     time.Now().Add(24 * time.Hour),
	}

	// Setup expectations
	mockEventService.On("CanUserEditEvent", 1, 1).Return(true, nil)
	mockTicketService.On("GetTicketTypeByID", 1).Return(testTicketType, nil)

	// Create request
	req := httptest.NewRequest("DELETE", "/organizer/events/1/tickets/1", nil)
	req = req.WithContext(setUserInContext(req.Context(), testUser))

	// Create router and add route
	r := chi.NewRouter()
	r.Delete("/organizer/events/{eventId}/tickets/{id}", handler.DeleteTicketType)

	// Execute request
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Cannot delete ticket type with sold tickets")

	mockEventService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}

func TestTicketTypeHandler_UnauthorizedAccess(t *testing.T) {
	mockTicketService := new(MockTicketServiceForTicketTypes)
	mockEventService := new(MockEventServiceForTicketTypes)
	handler := NewTicketTypeHandler(mockTicketService, mockEventService)

	// Create test data - user who doesn't own the event
	testUser := &models.User{
		ID:    2,
		Email: "other@example.com",
		Role:  models.RoleOrganizer,
	}

	// Setup expectations - user cannot edit this event
	mockEventService.On("CanUserEditEvent", 1, 2).Return(false, nil)

	// Create request
	req := httptest.NewRequest("GET", "/organizer/events/1/tickets", nil)
	req = req.WithContext(setUserInContext(req.Context(), testUser))

	// Create router and add route
	r := chi.NewRouter()
	r.Get("/organizer/events/{eventId}/tickets", handler.TicketTypesPage)

	// Execute request
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusForbidden, w.Code)

	mockEventService.AssertExpectations(t)
}

func TestTicketTypeHandler_HTMXRequests(t *testing.T) {
	mockTicketService := new(MockTicketServiceForTicketTypes)
	mockEventService := new(MockEventServiceForTicketTypes)
	handler := NewTicketTypeHandler(mockTicketService, mockEventService)

	// Create test data
	testUser := &models.User{
		ID:    1,
		Email: "organizer@example.com",
		Role:  models.RoleOrganizer,
	}

	testEvent := &models.Event{
		ID:          1,
		Title:       "Test Event",
		OrganizerID: 1,
	}

	testTicketTypes := []*models.TicketType{
		{
			ID:          1,
			EventID:     1,
			Name:        "General Admission",
			Description: "Standard entry ticket",
			Price:       2500, // $25.00
			Quantity:    100,
			Sold:        25,
			SaleStart:   time.Now().Add(-1 * time.Hour),
			SaleEnd:     time.Now().Add(24 * time.Hour),
		},
	}

	// Setup expectations
	mockEventService.On("CanUserEditEvent", 1, 1).Return(true, nil)
	mockEventService.On("GetEventByID", 1).Return(testEvent, nil)
	mockTicketService.On("GetTicketTypesByEventID", 1).Return(testTicketTypes, nil)

	// Create HTMX request
	req := httptest.NewRequest("GET", "/organizer/events/1/tickets", nil)
	req.Header.Set("HX-Request", "true")
	req = req.WithContext(setUserInContext(req.Context(), testUser))

	// Create router and add route
	r := chi.NewRouter()
	r.Get("/organizer/events/{eventId}/tickets", handler.TicketTypesPage)

	// Execute request
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	// Should return partial content for HTMX, not full page
	assert.NotContains(t, w.Body.String(), "<html>")
	assert.Contains(t, w.Body.String(), "General Admission")

	mockEventService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}
