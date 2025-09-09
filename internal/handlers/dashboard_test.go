package handlers

import (
	"context"
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



func (m *MockOrderService) GetOrdersByUserID(userID int, limit int) ([]*models.Order, error) {
	args := m.Called(userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Order), args.Error(1)
}

func (m *MockOrderService) CreateOrder(req *services.OrderCreateRequest) (*models.Order, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Order), args.Error(1)
}

func (m *MockOrderService) UpdateOrderStatus(id int, status models.OrderStatus) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockOrderService) GetUserOrders(userID int, limit, offset int) ([]*repositories.OrderWithDetails, int, error) {
	args := m.Called(userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int), args.Error(2)
	}
	return args.Get(0).([]*repositories.OrderWithDetails), args.Get(1).(int), args.Error(2)
}

func (m *MockOrderService) GetOrderByID(orderID int, requestingUserID int) (*models.Order, error) {
	args := m.Called(orderID, requestingUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Order), args.Error(1)
}

func (m *MockOrderService) GetOrderWithTickets(orderID int, requestingUserID int) (*models.Order, []*models.Ticket, error) {
	args := m.Called(orderID, requestingUserID)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*models.Order), args.Get(1).([]*models.Ticket), args.Error(2)
}

func (m *MockOrderService) CancelOrder(orderID int, requestingUserID int) error {
	args := m.Called(orderID, requestingUserID)
	return args.Error(0)
}

func (m *MockOrderService) GetEventOrders(eventID int, requestingUserID int, limit, offset int) ([]*repositories.OrderWithDetails, int, error) {
	args := m.Called(eventID, requestingUserID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int), args.Error(2)
	}
	return args.Get(0).([]*repositories.OrderWithDetails), args.Get(1).(int), args.Error(2)
}

func (m *MockOrderService) GetOrderStatistics(eventID *int, userID *int, requestingUserID int) (map[string]interface{}, error) {
	args := m.Called(eventID, userID, requestingUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockOrderService) SearchUserOrders(userID int, filters repositories.OrderSearchFilters, requestingUserID int) ([]*repositories.OrderWithDetails, int, error) {
	args := m.Called(userID, filters, requestingUserID)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int), args.Error(2)
	}
	return args.Get(0).([]*repositories.OrderWithDetails), args.Get(1).(int), args.Error(2)
}

func (m *MockOrderService) GetOrderCount() (int, error) {
	args := m.Called()
	return args.Get(0).(int), args.Error(1)
}

func (m *MockOrderService) GetTotalRevenue() (float64, error) {
	args := m.Called()
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockOrderService) GetUpcomingEventsForUser(userID int, requestingUserID int) ([]*models.Event, error) {
	args := m.Called(userID, requestingUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Event), args.Error(1)
}

func (m *MockOrderService) CompleteOrder(orderID int, paymentID string, ticketData []struct {
	TicketTypeID int
	QRCode       string
}) error {
	args := m.Called(orderID, paymentID, ticketData)
	return args.Error(0)
}

// MockEventService for testing
type MockEventServiceForDashboard struct {
	mock.Mock
}

func (m *MockEventServiceForDashboard) GetEventByID(id int) (*models.Event, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventServiceForDashboard) GetFeaturedEvents(limit int) ([]*models.Event, error) {
	args := m.Called(limit)
	return args.Get(0).([]*models.Event), args.Error(1)
}

func (m *MockEventServiceForDashboard) GetUpcomingEvents(limit int) ([]*models.Event, error) {
	args := m.Called(limit)
	return args.Get(0).([]*models.Event), args.Error(1)
}

func (m *MockEventServiceForDashboard) SearchEvents(filters services.EventSearchFilters) ([]*models.Event, int, error) {
	args := m.Called(filters)
	return args.Get(0).([]*models.Event), args.Get(1).(int), args.Error(2)
}

func (m *MockEventServiceForDashboard) GetCategories() ([]*models.Category, error) {
	args := m.Called()
	return args.Get(0).([]*models.Category), args.Error(1)
}

func (m *MockEventServiceForDashboard) GetEventOrganizer(eventID int) (*models.User, error) {
	args := m.Called(eventID)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockEventServiceForDashboard) CreateEvent(req *services.EventCreateRequest) (*models.Event, error) {
	args := m.Called(req)
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventServiceForDashboard) UpdateEvent(id int, req *services.EventUpdateRequest) (*models.Event, error) {
	args := m.Called(id, req)
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventServiceForDashboard) DeleteEvent(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockEventServiceForDashboard) GetEventsByOrganizer(organizerID int) ([]*models.Event, error) {
	args := m.Called(organizerID)
	return args.Get(0).([]*models.Event), args.Error(1)
}

func (m *MockEventServiceForDashboard) CanUserEditEvent(eventID int, userID int) (bool, error) {
	args := m.Called(eventID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockEventServiceForDashboard) CanUserDeleteEvent(eventID int, userID int) (bool, error) {
	args := m.Called(eventID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockEventServiceForDashboard) UpdateEventStatus(eventID int, status models.EventStatus, organizerID int) (*models.Event, error) {
	args := m.Called(eventID, status, organizerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventServiceForDashboard) DuplicateEvent(eventID int, organizerID int, newTitle string, newStartDate, newEndDate time.Time) (*models.Event, error) {
	args := m.Called(eventID, organizerID, newTitle, newStartDate, newEndDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventServiceForDashboard) GetEventCount() (int, error) {
	args := m.Called()
	return args.Get(0).(int), args.Error(1)
}

func (m *MockEventServiceForDashboard) GetPublishedEventCount() (int, error) {
	args := m.Called()
	return args.Get(0).(int), args.Error(1)
}

// MockTicketService for testing
type MockTicketServiceForDashboard struct {
	mock.Mock
}

func (m *MockTicketServiceForDashboard) GetTicketByID(id int) (*models.Ticket, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ticket), args.Error(1)
}

func (m *MockTicketServiceForDashboard) GetTicketsByOrderID(orderID int) ([]*models.Ticket, error) {
	args := m.Called(orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ticket), args.Error(1)
}

func (m *MockTicketServiceForDashboard) GenerateTicketsPDF(tickets []*models.Ticket, event *models.Event, order *models.Order) ([]byte, error) {
	args := m.Called(tickets, event, order)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockTicketServiceForDashboard) GetTicketTypesByEventID(eventID int) ([]*models.TicketType, error) {
	args := m.Called(eventID)
	return args.Get(0).([]*models.TicketType), args.Error(1)
}

func (m *MockTicketServiceForDashboard) GetTicketTypeByID(id int) (*models.TicketType, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TicketType), args.Error(1)
}

func (m *MockTicketServiceForDashboard) CreateTicketType(req *models.TicketTypeCreateRequest) (*models.TicketType, error) {
	args := m.Called(req)
	return args.Get(0).(*models.TicketType), args.Error(1)
}

func (m *MockTicketServiceForDashboard) UpdateTicketType(id int, req *models.TicketTypeUpdateRequest) (*models.TicketType, error) {
	args := m.Called(id, req)
	return args.Get(0).(*models.TicketType), args.Error(1)
}

func (m *MockTicketServiceForDashboard) DeleteTicketType(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func createTestUser() *models.User {
	return &models.User{
		ID:        1,
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RoleAttendee,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func createTestOrder() *models.Order {
	return &models.Order{
		ID:           1,
		UserID:       1,
		EventID:      1,
		OrderNumber:  "ORD-123456",
		TotalAmount:  5000,
		Status:       models.OrderCompleted,
		PaymentID:    "pay_123456",
		BillingEmail: "test@example.com",
		BillingName:  "John Doe",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func createTestEvent() *models.Event {
	return &models.Event{
		ID:          1,
		Title:       "Test Event",
		Description: "Test event description",
		StartDate:   time.Now().Add(24 * time.Hour),
		EndDate:     time.Now().Add(26 * time.Hour),
		Location:    "Test Location",
		CategoryID:  1,
		OrganizerID: 2,
		Status:      models.StatusPublished,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func createTestTickets() []*models.Ticket {
	return []*models.Ticket{
		{
			ID:           1,
			OrderID:      1,
			TicketTypeID: 1,
			QRCode:       "QR123456789",
			Status:       models.TicketActive,
			CreatedAt:    time.Now(),
		},
		{
			ID:           2,
			OrderID:      1,
			TicketTypeID: 1,
			QRCode:       "QR987654321",
			Status:       models.TicketActive,
			CreatedAt:    time.Now(),
		},
	}
}

func TestDashboardHandler_DashboardPage(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Mock service calls
	orders := []*models.Order{createTestOrder()}
	event := createTestEvent()
	mockOrderService.On("GetOrdersByUserID", 1, 5).Return(orders, nil)
	mockOrderService.On("GetOrdersByUserID", 1, 0).Return(orders, nil)
	mockEventService.On("GetEventByID", 1).Return(event, nil)

	req, err := http.NewRequest("GET", "/dashboard", nil)
	assert.NoError(t, err)

	// Add user to context
	user := createTestUser()
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.DashboardPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Welcome back, John!")
	mockOrderService.AssertExpectations(t)
}

func TestDashboardHandler_DashboardPage_NoUser(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	req, err := http.NewRequest("GET", "/dashboard", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.DashboardPage(rr, req)

	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/auth/login", rr.Header().Get("Location"))
}

func TestDashboardHandler_OrdersPage(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Mock service calls
	orders := []*models.Order{createTestOrder()}
	event := createTestEvent()
	mockOrderService.On("GetOrdersByUserID", 1, 0).Return(orders, nil)
	mockEventService.On("GetEventByID", 1).Return(event, nil)

	req, err := http.NewRequest("GET", "/dashboard/orders", nil)
	assert.NoError(t, err)

	// Add user to context
	user := createTestUser()
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.OrdersPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "My Orders")
	assert.Contains(t, rr.Body.String(), "ORD-123456")
	mockOrderService.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
}

func TestDashboardHandler_OrderDetailsPage(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Mock service calls
	order := createTestOrder()
	event := createTestEvent()
	tickets := createTestTickets()
	mockOrderService.On("GetOrderByID", 1).Return(order, nil)
	mockEventService.On("GetEventByID", 1).Return(event, nil)
	mockTicketService.On("GetTicketsByOrderID", 1).Return(tickets, nil)

	req, err := http.NewRequest("GET", "/dashboard/orders/1", nil)
	assert.NoError(t, err)

	// Add user to context
	user := createTestUser()
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	// Add URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.OrderDetailsPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Order Details")
	assert.Contains(t, rr.Body.String(), "ORD-123456")
	assert.Contains(t, rr.Body.String(), "Test Event")
	mockOrderService.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}

func TestDashboardHandler_OrderDetailsPage_AccessDenied(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Mock service calls - order belongs to different user
	order := createTestOrder()
	order.UserID = 2 // Different user ID
	mockOrderService.On("GetOrderByID", 1).Return(order, nil)

	req, err := http.NewRequest("GET", "/dashboard/orders/1", nil)
	assert.NoError(t, err)

	// Add user to context
	user := createTestUser() // User ID = 1
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	// Add URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.OrderDetailsPage(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	mockOrderService.AssertExpectations(t)
}

func TestDashboardHandler_DownloadTickets(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Mock service calls
	order := createTestOrder()
	event := createTestEvent()
	tickets := createTestTickets()
	pdfData := []byte("PDF content")
	
	mockOrderService.On("GetOrderByID", 1).Return(order, nil)
	mockEventService.On("GetEventByID", 1).Return(event, nil)
	mockTicketService.On("GetTicketsByOrderID", 1).Return(tickets, nil)
	mockTicketService.On("GenerateTicketsPDF", tickets, event, order).Return(pdfData, nil)

	req, err := http.NewRequest("GET", "/dashboard/orders/1/tickets", nil)
	assert.NoError(t, err)

	// Add user to context
	user := createTestUser()
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	// Add URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.DownloadTickets(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/pdf", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Header().Get("Content-Disposition"), "tickets-ORD-123456.pdf")
	assert.Equal(t, pdfData, rr.Body.Bytes())
	
	mockOrderService.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}

func TestDashboardHandler_DownloadTickets_PendingOrder(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Mock service calls - order is pending
	order := createTestOrder()
	order.Status = models.OrderPending
	mockOrderService.On("GetOrderByID", 1).Return(order, nil)

	req, err := http.NewRequest("GET", "/dashboard/orders/1/tickets", nil)
	assert.NoError(t, err)

	// Add user to context
	user := createTestUser()
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	// Add URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.DownloadTickets(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Tickets are not available for this order")
	mockOrderService.AssertExpectations(t)
}

func TestDashboardHandler_DownloadSingleTicket(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Mock service calls
	ticket := createTestTickets()[0]
	order := createTestOrder()
	event := createTestEvent()
	pdfData := []byte("PDF content")
	
	mockTicketService.On("GetTicketByID", 1).Return(ticket, nil)
	mockOrderService.On("GetOrderByID", 1).Return(order, nil)
	mockEventService.On("GetEventByID", 1).Return(event, nil)
	mockTicketService.On("GenerateTicketsPDF", []*models.Ticket{ticket}, event, order).Return(pdfData, nil)

	req, err := http.NewRequest("GET", "/dashboard/tickets/1/download", nil)
	assert.NoError(t, err)

	// Add user to context
	user := createTestUser()
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	// Add URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.DownloadSingleTicket(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/pdf", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Header().Get("Content-Disposition"), "ticket-QR123456789.pdf")
	assert.Equal(t, pdfData, rr.Body.Bytes())
	
	mockOrderService.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}