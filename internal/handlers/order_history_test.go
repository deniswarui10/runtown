package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
	"event-ticketing-platform/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOrderService for testing
type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) GetUserOrders(userID int, limit, offset int) ([]*repositories.OrderWithDetails, int, error) {
	args := m.Called(userID, limit, offset)
	return args.Get(0).([]*repositories.OrderWithDetails), args.Int(1), args.Error(2)
}

func (m *MockOrderService) GetOrderByID(orderID int, requestingUserID int) (*models.Order, error) {
	args := m.Called(orderID, requestingUserID)
	return args.Get(0).(*models.Order), args.Error(1)
}

func (m *MockOrderService) GetOrderWithTickets(orderID int, requestingUserID int) (*models.Order, []*models.Ticket, error) {
	args := m.Called(orderID, requestingUserID)
	return args.Get(0).(*models.Order), args.Get(1).([]*models.Ticket), args.Error(2)
}

func (m *MockOrderService) CancelOrder(orderID int, requestingUserID int) error {
	args := m.Called(orderID, requestingUserID)
	return args.Error(0)
}

func (m *MockOrderService) GetEventOrders(eventID int, requestingUserID int, limit, offset int) ([]*repositories.OrderWithDetails, int, error) {
	args := m.Called(eventID, requestingUserID, limit, offset)
	return args.Get(0).([]*repositories.OrderWithDetails), args.Int(1), args.Error(2)
}

func (m *MockOrderService) GetOrderStatistics(eventID *int, userID *int, requestingUserID int) (map[string]interface{}, error) {
	args := m.Called(eventID, userID, requestingUserID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockOrderService) SearchUserOrders(userID int, filters repositories.OrderSearchFilters, requestingUserID int) ([]*repositories.OrderWithDetails, int, error) {
	args := m.Called(userID, filters, requestingUserID)
	return args.Get(0).([]*repositories.OrderWithDetails), args.Int(1), args.Error(2)
}

func (m *MockOrderService) GetUpcomingEventsForUser(userID int, requestingUserID int) ([]*models.Event, error) {
	args := m.Called(userID, requestingUserID)
	return args.Get(0).([]*models.Event), args.Error(1)
}

// MockEventService for testing
type MockEventService struct {
	mock.Mock
}

func (m *MockEventService) GetFeaturedEvents(limit int) ([]*models.Event, error) {
	args := m.Called(limit)
	return args.Get(0).([]*models.Event), args.Error(1)
}

func (m *MockEventService) GetUpcomingEvents(limit int) ([]*models.Event, error) {
	args := m.Called(limit)
	return args.Get(0).([]*models.Event), args.Error(1)
}

func (m *MockEventService) SearchEvents(filters services.EventSearchFilters) ([]*models.Event, int, error) {
	args := m.Called(filters)
	return args.Get(0).([]*models.Event), args.Int(1), args.Error(2)
}

func (m *MockEventService) GetCategories() ([]*models.Category, error) {
	args := m.Called()
	return args.Get(0).([]*models.Category), args.Error(1)
}

func (m *MockEventService) GetEventByID(id int) (*models.Event, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventService) GetEventOrganizer(eventID int) (*models.User, error) {
	args := m.Called(eventID)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockEventService) CreateEvent(req *services.EventCreateRequest) (*models.Event, error) {
	args := m.Called(req)
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventService) UpdateEvent(id int, req *services.EventUpdateRequest) (*models.Event, error) {
	args := m.Called(id, req)
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventService) DeleteEvent(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockEventService) GetEventsByOrganizer(organizerID int) ([]*models.Event, error) {
	args := m.Called(organizerID)
	return args.Get(0).([]*models.Event), args.Error(1)
}

// MockTicketService for testing
type MockTicketService struct {
	mock.Mock
}

func (m *MockTicketService) GetTicketTypesByEventID(eventID int) ([]*models.TicketType, error) {
	args := m.Called(eventID)
	return args.Get(0).([]*models.TicketType), args.Error(1)
}

func (m *MockTicketService) CreateTicketType(req *models.TicketTypeCreateRequest) (*models.TicketType, error) {
	args := m.Called(req)
	return args.Get(0).(*models.TicketType), args.Error(1)
}

func (m *MockTicketService) UpdateTicketType(id int, req *models.TicketTypeUpdateRequest) (*models.TicketType, error) {
	args := m.Called(id, req)
	return args.Get(0).(*models.TicketType), args.Error(1)
}

func (m *MockTicketService) DeleteTicketType(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockTicketService) GetTicketByID(id int) (*models.Ticket, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Ticket), args.Error(1)
}

func (m *MockTicketService) GetTicketsByOrderID(orderID int) ([]*models.Ticket, error) {
	args := m.Called(orderID)
	return args.Get(0).([]*models.Ticket), args.Error(1)
}

func (m *MockTicketService) GenerateTicketsPDF(tickets []*models.Ticket, event *models.Event, order *models.Order) ([]byte, error) {
	args := m.Called(tickets, event, order)
	return args.Get(0).([]byte), args.Error(1)
}

func TestOrderHistoryPage(t *testing.T) {
	// Create mock services
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventService)
	mockTicketService := new(MockTicketService)

	// Create handler
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Create test user
	user := &models.User{
		ID:        1,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleAttendee,
	}

	// Create test orders
	testOrders := []*repositories.OrderWithDetails{
		{
			Order: &models.Order{
				ID:           1,
				UserID:       1,
				EventID:      1,
				OrderNumber:  "ORD-20240101-123456",
				TotalAmount:  5000, // $50.00
				Status:       models.OrderCompleted,
				BillingEmail: "test@example.com",
				BillingName:  "Test User",
				CreatedAt:    time.Now().Add(-24 * time.Hour),
				UpdatedAt:    time.Now().Add(-24 * time.Hour),
			},
			EventTitle:  "Test Event",
			EventDate:   time.Now().Add(7 * 24 * time.Hour),
			TicketCount: 2,
		},
	}

	// Create test upcoming events
	upcomingEvents := []*models.Event{
		{
			ID:        1,
			Title:     "Test Event",
			StartDate: time.Now().Add(7 * 24 * time.Hour),
			Location:  "Test Location",
			Status:    models.StatusPublished,
		},
	}

	t.Run("successful order history page load", func(t *testing.T) {
		// Setup expectations
		mockOrderService.On("SearchUserOrders", 1, mock.AnythingOfType("repositories.OrderSearchFilters"), 1).Return(testOrders, 1, nil)

		// Create request
		req := httptest.NewRequest("GET", "/dashboard/orders", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.OrdersPage(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Order History")
		assert.Contains(t, w.Body.String(), "ORD-20240101-123456")
		assert.Contains(t, w.Body.String(), "Test Event")

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
	})

	t.Run("order history with filters", func(t *testing.T) {
		// Setup expectations
		mockOrderService.On("SearchUserOrders", 1, mock.MatchedBy(func(filters repositories.OrderSearchFilters) bool {
			return filters.Status == models.OrderCompleted && filters.Limit == 10
		}), 1).Return(testOrders, 1, nil)

		// Create request with query parameters
		req := httptest.NewRequest("GET", "/dashboard/orders?status=completed&per_page=10", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.OrdersPage(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Order History")

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
	})

	t.Run("HTMX partial update", func(t *testing.T) {
		// Setup expectations
		mockOrderService.On("SearchUserOrders", 1, mock.AnythingOfType("repositories.OrderSearchFilters"), 1).Return(testOrders, 1, nil)

		// Create HTMX request
		req := httptest.NewRequest("GET", "/dashboard/orders", nil)
		req.Header.Set("HX-Request", "true")
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.OrdersPage(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		// Should not contain full page layout for HTMX requests
		assert.NotContains(t, w.Body.String(), "<html>")

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
	})

	t.Run("unauthorized access", func(t *testing.T) {
		// Create request without user context
		req := httptest.NewRequest("GET", "/dashboard/orders", nil)

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.OrdersPage(w, req)

		// Assert redirect to login
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/auth/login", w.Header().Get("Location"))
	})
}

func TestOrderDetailsEnhancedPage(t *testing.T) {
	// Create mock services
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventService)
	mockTicketService := new(MockTicketService)

	// Create handler
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Create test user
	user := &models.User{
		ID:        1,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleAttendee,
	}

	// Create test order
	testOrder := &models.Order{
		ID:           1,
		UserID:       1,
		EventID:      1,
		OrderNumber:  "ORD-20240101-123456",
		TotalAmount:  5000, // $50.00
		Status:       models.OrderCompleted,
		BillingEmail: "test@example.com",
		BillingName:  "Test User",
		CreatedAt:    time.Now().Add(-24 * time.Hour),
		UpdatedAt:    time.Now().Add(-24 * time.Hour),
	}

	// Create test event
	testEvent := &models.Event{
		ID:          1,
		Title:       "Test Event",
		Description: "Test event description",
		StartDate:   time.Now().Add(7 * 24 * time.Hour),
		Location:    "Test Location",
		Status:      models.StatusPublished,
	}

	// Create test tickets
	testTickets := []*models.Ticket{
		{
			ID:           1,
			OrderID:      1,
			TicketTypeID: 1,
			QRCode:       "QR123456",
			Status:       models.TicketActive,
			CreatedAt:    time.Now().Add(-24 * time.Hour),
		},
		{
			ID:           2,
			OrderID:      1,
			TicketTypeID: 1,
			QRCode:       "QR789012",
			Status:       models.TicketActive,
			CreatedAt:    time.Now().Add(-24 * time.Hour),
		},
	}

	// Create test ticket types
	testTicketTypes := []*models.TicketType{
		{
			ID:          1,
			EventID:     1,
			Name:        "General Admission",
			Description: "General admission ticket",
			Price:       2500, // $25.00
			Quantity:    100,
			Sold:        50,
		},
	}

	t.Run("successful order details page load", func(t *testing.T) {
		// Setup expectations
		mockOrderService.On("GetOrderByID", 1, 1).Return(testOrder, nil)
		mockEventService.On("GetEventByID", 1).Return(testEvent, nil)
		mockTicketService.On("GetTicketsByOrderID", 1).Return(testTickets, nil)
		mockTicketService.On("GetTicketTypesByEventID", 1).Return(testTicketTypes, nil)

		// Create request
		req := httptest.NewRequest("GET", "/dashboard/orders/1", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Add URL parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(chi.NewContext(req.Context(), rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.OrderDetailsPage(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Order Details")
		assert.Contains(t, w.Body.String(), "ORD-20240101-123456")
		assert.Contains(t, w.Body.String(), "Test Event")
		assert.Contains(t, w.Body.String(), "QR123456")
		assert.Contains(t, w.Body.String(), "General Admission")

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
		mockEventService.AssertExpectations(t)
		mockTicketService.AssertExpectations(t)
	})

	t.Run("order not found", func(t *testing.T) {
		// Setup expectations
		mockOrderService.On("GetOrderByID", 999, 1).Return((*models.Order)(nil), fmt.Errorf("order not found"))

		// Create request
		req := httptest.NewRequest("GET", "/dashboard/orders/999", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Add URL parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "999")
		req = req.WithContext(chi.NewContext(req.Context(), rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.OrderDetailsPage(w, req)

		// Assert response
		assert.Equal(t, http.StatusNotFound, w.Code)

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
	})

	t.Run("invalid order ID", func(t *testing.T) {
		// Create request with invalid ID
		req := httptest.NewRequest("GET", "/dashboard/orders/invalid", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Add URL parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "invalid")
		req = req.WithContext(chi.NewContext(req.Context(), rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.OrderDetailsPage(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCancelOrder(t *testing.T) {
	// Create mock services
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventService)
	mockTicketService := new(MockTicketService)

	// Create handler
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Create test user
	user := &models.User{
		ID:        1,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleAttendee,
	}

	t.Run("successful order cancellation", func(t *testing.T) {
		// Setup expectations
		mockOrderService.On("CancelOrder", 1, 1).Return(nil)

		// Create request
		req := httptest.NewRequest("POST", "/dashboard/orders/1/cancel", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Add URL parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(chi.NewContext(req.Context(), rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.CancelOrder(w, req)

		// Assert response
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/dashboard/orders")
		assert.Contains(t, w.Header().Get("Location"), "cancelled=1")

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
	})

	t.Run("HTMX order cancellation", func(t *testing.T) {
		// Setup expectations
		mockOrderService.On("CancelOrder", 1, 1).Return(nil)

		// Create HTMX request
		req := httptest.NewRequest("POST", "/dashboard/orders/1/cancel", nil)
		req.Header.Set("HX-Request", "true")
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Add URL parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(chi.NewContext(req.Context(), rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.CancelOrder(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "/dashboard/orders?cancelled=1", w.Header().Get("HX-Redirect"))

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
	})

	t.Run("order cannot be cancelled", func(t *testing.T) {
		// Setup expectations
		mockOrderService.On("CancelOrder", 1, 1).Return(fmt.Errorf("order cannot be cancelled"))

		// Create request
		req := httptest.NewRequest("POST", "/dashboard/orders/1/cancel", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Add URL parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(chi.NewContext(req.Context(), rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.CancelOrder(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
	})

	t.Run("unauthorized access", func(t *testing.T) {
		// Create request without user context
		req := httptest.NewRequest("POST", "/dashboard/orders/1/cancel", nil)

		// Add URL parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(chi.NewContext(req.Context(), rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.CancelOrder(w, req)

		// Assert response
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestDownloadTickets(t *testing.T) {
	// Create mock services
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventService)
	mockTicketService := new(MockTicketService)

	// Create handler
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Create test user
	user := &models.User{
		ID:        1,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleAttendee,
	}

	// Create test order
	testOrder := &models.Order{
		ID:           1,
		UserID:       1,
		EventID:      1,
		OrderNumber:  "ORD-20240101-123456",
		TotalAmount:  5000,
		Status:       models.OrderCompleted,
		BillingEmail: "test@example.com",
		BillingName:  "Test User",
	}

	// Create test event
	testEvent := &models.Event{
		ID:        1,
		Title:     "Test Event",
		StartDate: time.Now().Add(7 * 24 * time.Hour),
		Location:  "Test Location",
	}

	// Create test tickets
	testTickets := []*models.Ticket{
		{
			ID:           1,
			OrderID:      1,
			TicketTypeID: 1,
			QRCode:       "QR123456",
			Status:       models.TicketActive,
		},
	}

	// Mock PDF data
	mockPDFData := []byte("mock pdf data")

	t.Run("successful ticket download", func(t *testing.T) {
		// Setup expectations
		mockOrderService.On("GetOrderByID", 1, 1).Return(testOrder, nil)
		mockTicketService.On("GetTicketsByOrderID", 1).Return(testTickets, nil)
		mockEventService.On("GetEventByID", 1).Return(testEvent, nil)
		mockTicketService.On("GenerateTicketsPDF", testTickets, testEvent, testOrder).Return(mockPDFData, nil)

		// Create request
		req := httptest.NewRequest("GET", "/dashboard/orders/1/tickets/download", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Add URL parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(chi.NewContext(req.Context(), rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.DownloadTickets(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Header().Get("Content-Disposition"), "tickets-ORD-20240101-123456.pdf")
		assert.Equal(t, mockPDFData, w.Body.Bytes())

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
		mockEventService.AssertExpectations(t)
		mockTicketService.AssertExpectations(t)
	})

	t.Run("order not completed", func(t *testing.T) {
		// Create pending order
		pendingOrder := &models.Order{
			ID:           1,
			UserID:       1,
			EventID:      1,
			OrderNumber:  "ORD-20240101-123456",
			TotalAmount:  5000,
			Status:       models.OrderPending,
			BillingEmail: "test@example.com",
			BillingName:  "Test User",
		}

		// Setup expectations
		mockOrderService.On("GetOrderByID", 1, 1).Return(pendingOrder, nil)

		// Create request
		req := httptest.NewRequest("GET", "/dashboard/orders/1/tickets/download", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Add URL parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(chi.NewContext(req.Context(), rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.DownloadTickets(w, req)

		// Assert response
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
	})
}

func TestDownloadSingleTicket(t *testing.T) {
	// Create mock services
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventService)
	mockTicketService := new(MockTicketService)

	// Create handler
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Create test user
	user := &models.User{
		ID:        1,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleAttendee,
	}

	// Create test ticket
	testTicket := &models.Ticket{
		ID:           1,
		OrderID:      1,
		TicketTypeID: 1,
		QRCode:       "QR123456",
		Status:       models.TicketActive,
	}

	// Create test order
	testOrder := &models.Order{
		ID:           1,
		UserID:       1,
		EventID:      1,
		OrderNumber:  "ORD-20240101-123456",
		TotalAmount:  5000,
		Status:       models.OrderCompleted,
		BillingEmail: "test@example.com",
		BillingName:  "Test User",
	}

	// Create test event
	testEvent := &models.Event{
		ID:        1,
		Title:     "Test Event",
		StartDate: time.Now().Add(7 * 24 * time.Hour),
		Location:  "Test Location",
	}

	// Mock PDF data
	mockPDFData := []byte("mock single ticket pdf data")

	t.Run("successful single ticket download", func(t *testing.T) {
		// Setup expectations
		mockTicketService.On("GetTicketByID", 1).Return(testTicket, nil)
		mockOrderService.On("GetOrderByID", 1, 1).Return(testOrder, nil)
		mockEventService.On("GetEventByID", 1).Return(testEvent, nil)
		mockTicketService.On("GenerateTicketsPDF", []*models.Ticket{testTicket}, testEvent, testOrder).Return(mockPDFData, nil)

		// Create request
		req := httptest.NewRequest("GET", "/dashboard/tickets/1/download", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Add URL parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(chi.NewContext(req.Context(), rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.DownloadSingleTicket(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Header().Get("Content-Disposition"), "ticket-QR123456.pdf")
		assert.Equal(t, mockPDFData, w.Body.Bytes())

		// Verify mock calls
		mockTicketService.AssertExpectations(t)
		mockOrderService.AssertExpectations(t)
		mockEventService.AssertExpectations(t)
	})

	t.Run("ticket not found", func(t *testing.T) {
		// Setup expectations
		mockTicketService.On("GetTicketByID", 999).Return((*models.Ticket)(nil), fmt.Errorf("ticket not found"))

		// Create request
		req := httptest.NewRequest("GET", "/dashboard/tickets/999/download", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Add URL parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "999")
		req = req.WithContext(chi.NewContext(req.Context(), rctx))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.DownloadSingleTicket(w, req)

		// Assert response
		assert.Equal(t, http.StatusNotFound, w.Code)

		// Verify mock calls
		mockTicketService.AssertExpectations(t)
	})
}

func TestOrderSearchAndFiltering(t *testing.T) {
	// Create mock services
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventService)
	mockTicketService := new(MockTicketService)

	// Create handler
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Create test user
	user := &models.User{
		ID:        1,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleAttendee,
	}

	t.Run("filter by status", func(t *testing.T) {
		// Create test orders
		completedOrders := []*repositories.OrderWithDetails{
			{
				Order: &models.Order{
					ID:           1,
					UserID:       1,
					EventID:      1,
					OrderNumber:  "ORD-20240101-123456",
					TotalAmount:  5000,
					Status:       models.OrderCompleted,
					BillingEmail: "test@example.com",
					BillingName:  "Test User",
					CreatedAt:    time.Now().Add(-24 * time.Hour),
				},
				EventTitle:  "Test Event",
				EventDate:   time.Now().Add(7 * 24 * time.Hour),
				TicketCount: 2,
			},
		}

		// Setup expectations
		mockOrderService.On("SearchUserOrders", 1, mock.MatchedBy(func(filters repositories.OrderSearchFilters) bool {
			return filters.Status == models.OrderCompleted
		}), 1).Return(completedOrders, 1, nil)

		// Create request with status filter
		req := httptest.NewRequest("GET", "/dashboard/orders?status=completed", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.OrdersPage(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-20240101-123456")

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
	})

	t.Run("filter by date range", func(t *testing.T) {
		// Create test orders
		dateFilteredOrders := []*repositories.OrderWithDetails{
			{
				Order: &models.Order{
					ID:           1,
					UserID:       1,
					EventID:      1,
					OrderNumber:  "ORD-20240101-123456",
					TotalAmount:  5000,
					Status:       models.OrderCompleted,
					BillingEmail: "test@example.com",
					BillingName:  "Test User",
					CreatedAt:    time.Now().Add(-24 * time.Hour),
				},
				EventTitle:  "Test Event",
				EventDate:   time.Now().Add(7 * 24 * time.Hour),
				TicketCount: 2,
			},
		}

		// Setup expectations
		mockOrderService.On("SearchUserOrders", 1, mock.MatchedBy(func(filters repositories.OrderSearchFilters) bool {
			return filters.DateFrom != nil && filters.DateTo != nil
		}), 1).Return(dateFilteredOrders, 1, nil)

		// Create request with date filters
		yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
		tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
		req := httptest.NewRequest("GET", fmt.Sprintf("/dashboard/orders?date_from=%s&date_to=%s", yesterday, tomorrow), nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.OrdersPage(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-20240101-123456")

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
	})

	t.Run("filter by event name", func(t *testing.T) {
		// Create test orders
		eventFilteredOrders := []*repositories.OrderWithDetails{
			{
				Order: &models.Order{
					ID:           1,
					UserID:       1,
					EventID:      1,
					OrderNumber:  "ORD-20240101-123456",
					TotalAmount:  5000,
					Status:       models.OrderCompleted,
					BillingEmail: "test@example.com",
					BillingName:  "Test User",
					CreatedAt:    time.Now().Add(-24 * time.Hour),
				},
				EventTitle:  "Concert Event",
				EventDate:   time.Now().Add(7 * 24 * time.Hour),
				TicketCount: 2,
			},
		}

		// Setup expectations
		mockOrderService.On("SearchUserOrders", 1, mock.AnythingOfType("repositories.OrderSearchFilters"), 1).Return(eventFilteredOrders, 1, nil)

		// Create request with event name filter
		req := httptest.NewRequest("GET", "/dashboard/orders?event_name=concert", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.OrdersPage(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Concert Event")

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
	})

	t.Run("pagination", func(t *testing.T) {
		// Create test orders for pagination
		paginatedOrders := []*repositories.OrderWithDetails{
			{
				Order: &models.Order{
					ID:           1,
					UserID:       1,
					EventID:      1,
					OrderNumber:  "ORD-20240101-123456",
					TotalAmount:  5000,
					Status:       models.OrderCompleted,
					BillingEmail: "test@example.com",
					BillingName:  "Test User",
					CreatedAt:    time.Now().Add(-24 * time.Hour),
				},
				EventTitle:  "Test Event",
				EventDate:   time.Now().Add(7 * 24 * time.Hour),
				TicketCount: 2,
			},
		}

		// Setup expectations
		mockOrderService.On("SearchUserOrders", 1, mock.MatchedBy(func(filters repositories.OrderSearchFilters) bool {
			return filters.Limit == 10 && filters.Offset == 10 // Page 2 with 10 per page
		}), 1).Return(paginatedOrders, 25, nil) // Total of 25 orders

		// Create request with pagination
		req := httptest.NewRequest("GET", "/dashboard/orders?page=2&per_page=10", nil)
		req = req.WithContext(setUserInContext(req.Context(), user))

		// Create response recorder
		w := httptest.NewRecorder()

		// Call handler
		handler.OrdersPage(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Page 2 of 3") // 25 orders / 10 per page = 3 pages

		// Verify mock calls
		mockOrderService.AssertExpectations(t)
	})
}

// Helper function to set user in context for testing
func setUserInContext(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, middleware.UserContextKey, user)
}
func TestUpcomingEventsIntegration(t *testing.T) {
	// Create mock services
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventService)
	mockTicketService := new(MockTicketService)

	// Create handler
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Create test user
	user := &models.User{
		ID:        1,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleAttendee,
	}

	t.Run("display upcoming events in order history", func(t *testing.T) {
		// Create test orders
		testOrders := []*repositories.OrderWithDetails{
			{
				Order: &models.Order{
					ID:           1,
					UserID:       1,
					EventID:      1,
					OrderNumber:  "ORD-20240101-123456",
					TotalAmount:  5000,
					Status:       models.OrderCompleted,
					BillingEmail: "test@example.com",
					BillingName:  "Test User",
					CreatedAt:    time.Now().Add(-24 * time.Hour),
				},
				EventTitle:  "Upcoming Concert",
				EventDate:   time.Now().Add(7 * 24 * time.Hour),
				TicketCount: 2,
			},
		}

		// Create upcoming events
		upcomingEvents := []*models.Event{
			{
				ID:        1,
				Title:     "Upcoming Concert",
				StartDate: time.Now().Add(7 * 24 * time.Hour),
				Location:  "Concert Hall",
			},
		}

		// Setup mock expectations
		mockOrderService.On("GetOrdersWithDetailsByUserID", user.ID).Return(testOrders, nil)
		mockEventService.On("GetUpcomingEvents").Return(upcomingEvents, nil)

		// Test would continue with assertions...
		// For now, just verify mocks were called
		mockOrderService.AssertExpectations(t)
		mockEventService.AssertExpectations(t)
	})
}
