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

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// Integration tests for user account features
func TestDashboardHandler_UserAccountFeatures_Integration(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{"Complete User Dashboard Journey", testCompleteUserDashboardJourney},
		{"Order History Management", testOrderHistoryManagement},
		{"Ticket Download Functionality", testTicketDownloadFunctionality},
		{"Order Details and Re-download", testOrderDetailsAndRedownload},
		{"User Access Control", testUserAccessControl},
		{"Empty State Handling", testEmptyStateHandling},
		{"Error Handling", testErrorHandling},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func testCompleteUserDashboardJourney(t *testing.T) {
	// Setup mocks
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Create test data
	user := createTestUser()
	orders := []*models.Order{
		createTestOrder(),
		{
			ID:           2,
			UserID:       1,
			EventID:      2,
			OrderNumber:  "ORD-789012",
			TotalAmount:  7500,
			Status:       models.OrderCompleted,
			PaymentID:    "pay_789012",
			BillingEmail: "test@example.com",
			BillingName:  "John Doe",
			CreatedAt:    time.Now().Add(-48 * time.Hour),
			UpdatedAt:    time.Now().Add(-48 * time.Hour),
		},
	}
	
	events := []*models.Event{
		createTestEvent(),
		{
			ID:          2,
			Title:       "Another Test Event",
			Description: "Another test event description",
			StartDate:   time.Now().Add(72 * time.Hour),
			EndDate:     time.Now().Add(74 * time.Hour),
			Location:    "Another Test Location",
			CategoryID:  2,
			OrganizerID: 3,
			Status:      models.StatusPublished,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	// Mock service calls for dashboard
	mockOrderService.On("GetOrdersByUserID", 1, 5).Return(orders[:1], nil) // Recent orders (limit 5)
	mockOrderService.On("GetOrdersByUserID", 1, 0).Return(orders, nil)     // All orders
	mockEventService.On("GetEventByID", 1).Return(events[0], nil)
	mockEventService.On("GetEventByID", 2).Return(events[1], nil)

	// Test 1: Dashboard page shows user info and recent orders
	req := httptest.NewRequest("GET", "/dashboard", nil)
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.DashboardPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Welcome back, John!")
	assert.Contains(t, rr.Body.String(), "test@example.com")
	assert.Contains(t, rr.Body.String(), "ORD-123456")
	assert.Contains(t, rr.Body.String(), "Total Orders")
	assert.Contains(t, rr.Body.String(), "Upcoming Events")

	// Test 2: Orders page shows all orders
	req = httptest.NewRequest("GET", "/dashboard/orders", nil)
	req = req.WithContext(ctx)

	rr = httptest.NewRecorder()
	handler.OrdersPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "My Orders")
	assert.Contains(t, rr.Body.String(), "ORD-123456")
	assert.Contains(t, rr.Body.String(), "ORD-789012")
	assert.Contains(t, rr.Body.String(), "Test Event")
	assert.Contains(t, rr.Body.String(), "Another Test Event")

	mockOrderService.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
}

func testOrderHistoryManagement(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	user := createTestUser()
	
	// Test different order statuses
	orders := []*models.Order{
		{
			ID:           1,
			UserID:       1,
			EventID:      1,
			OrderNumber:  "ORD-COMPLETED",
			TotalAmount:  5000,
			Status:       models.OrderCompleted,
			PaymentID:    "pay_completed",
			BillingEmail: "test@example.com",
			BillingName:  "John Doe",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:           2,
			UserID:       1,
			EventID:      2,
			OrderNumber:  "ORD-PENDING",
			TotalAmount:  3000,
			Status:       models.OrderPending,
			PaymentID:    "",
			BillingEmail: "test@example.com",
			BillingName:  "John Doe",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:           3,
			UserID:       1,
			EventID:      3,
			OrderNumber:  "ORD-CANCELLED",
			TotalAmount:  2000,
			Status:       models.OrderCancelled,
			PaymentID:    "pay_cancelled",
			BillingEmail: "test@example.com",
			BillingName:  "John Doe",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	events := []*models.Event{
		{ID: 1, Title: "Completed Event", Status: models.StatusPublished, StartDate: time.Now().Add(24 * time.Hour)},
		{ID: 2, Title: "Pending Event", Status: models.StatusPublished, StartDate: time.Now().Add(48 * time.Hour)},
		{ID: 3, Title: "Cancelled Event", Status: models.StatusCancelled, StartDate: time.Now().Add(72 * time.Hour)},
	}

	mockOrderService.On("GetOrdersByUserID", 1, 0).Return(orders, nil)
	for i, event := range events {
		mockEventService.On("GetEventByID", i+1).Return(event, nil)
	}

	req := httptest.NewRequest("GET", "/dashboard/orders", nil)
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.OrdersPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	
	// Check that all order statuses are displayed correctly
	body := rr.Body.String()
	assert.Contains(t, body, "ORD-COMPLETED")
	assert.Contains(t, body, "ORD-PENDING")
	assert.Contains(t, body, "ORD-CANCELLED")
	assert.Contains(t, body, "completed")
	assert.Contains(t, body, "pending")
	assert.Contains(t, body, "cancelled")

	mockOrderService.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
}

func testTicketDownloadFunctionality(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	user := createTestUser()
	order := createTestOrder()
	event := createTestEvent()
	tickets := createTestTickets()
	pdfData := []byte("Mock PDF content for tickets")

	// Test bulk ticket download
	mockOrderService.On("GetOrderByID", 1).Return(order, nil)
	mockEventService.On("GetEventByID", 1).Return(event, nil)
	mockTicketService.On("GetTicketsByOrderID", 1).Return(tickets, nil)
	mockTicketService.On("GenerateTicketsPDF", tickets, event, order).Return(pdfData, nil)

	req := httptest.NewRequest("GET", "/dashboard/orders/1/tickets/download", nil)
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.DownloadTickets(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/pdf", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Header().Get("Content-Disposition"), "attachment")
	assert.Contains(t, rr.Header().Get("Content-Disposition"), "tickets-ORD-123456.pdf")
	assert.Equal(t, pdfData, rr.Body.Bytes())

	// Test single ticket download
	mockTicketService.On("GetTicketByID", 1).Return(tickets[0], nil)
	mockTicketService.On("GenerateTicketsPDF", []*models.Ticket{tickets[0]}, event, order).Return(pdfData, nil)

	req = httptest.NewRequest("GET", "/dashboard/tickets/1/download", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rr = httptest.NewRecorder()
	handler.DownloadSingleTicket(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/pdf", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Header().Get("Content-Disposition"), "ticket-QR123456789.pdf")

	mockOrderService.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}

func testOrderDetailsAndRedownload(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	user := createTestUser()
	order := createTestOrder()
	event := createTestEvent()
	tickets := createTestTickets()

	mockOrderService.On("GetOrderByID", 1).Return(order, nil)
	mockEventService.On("GetEventByID", 1).Return(event, nil)
	mockTicketService.On("GetTicketsByOrderID", 1).Return(tickets, nil)

	req := httptest.NewRequest("GET", "/dashboard/orders/1", nil)
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.OrderDetailsPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	
	body := rr.Body.String()
	// Check order details are displayed
	assert.Contains(t, body, "Order Details")
	assert.Contains(t, body, "ORD-123456")
	assert.Contains(t, body, "Test Event")
	assert.Contains(t, body, "Test Location")
	assert.Contains(t, body, "$50.00") // Total amount
	assert.Contains(t, body, "completed") // Order status
	
	// Check ticket information is displayed
	assert.Contains(t, body, "Your Tickets")
	assert.Contains(t, body, "QR123456789")
	assert.Contains(t, body, "QR987654321")
	
	// Check download links are present for completed orders
	assert.Contains(t, body, "Download All Tickets")
	assert.Contains(t, body, "Download Ticket")
	
	// Check billing information
	assert.Contains(t, body, "Billing Information")
	assert.Contains(t, body, "John Doe")
	assert.Contains(t, body, "test@example.com")

	mockOrderService.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}

func testUserAccessControl(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	// Test 1: User trying to access another user's order
	user1 := createTestUser()
	user1.ID = 1
	
	order := createTestOrder()
	order.UserID = 2 // Different user

	mockOrderService.On("GetOrderByID", 1).Return(order, nil)

	req := httptest.NewRequest("GET", "/dashboard/orders/1", nil)
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user1)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.OrderDetailsPage(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)

	// Test 2: User trying to download another user's tickets
	ticket := createTestTickets()[0]
	ticket.OrderID = 1

	mockTicketService.On("GetTicketByID", 1).Return(ticket, nil)

	req = httptest.NewRequest("GET", "/dashboard/tickets/1/download", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rr = httptest.NewRecorder()
	handler.DownloadSingleTicket(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)

	// Test 3: Unauthenticated user access
	req = httptest.NewRequest("GET", "/dashboard", nil)
	// No user in context

	rr = httptest.NewRecorder()
	handler.DashboardPage(rr, req)

	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/auth/login", rr.Header().Get("Location"))

	mockOrderService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}

func testEmptyStateHandling(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	user := createTestUser()

	// Test dashboard with no orders
	mockOrderService.On("GetOrdersByUserID", 1, 5).Return([]*models.Order{}, nil)
	mockOrderService.On("GetOrdersByUserID", 1, 0).Return([]*models.Order{}, nil)

	req := httptest.NewRequest("GET", "/dashboard", nil)
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.DashboardPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "No orders yet")
	assert.Contains(t, rr.Body.String(), "Browse Events")

	// Test orders page with no orders
	req = httptest.NewRequest("GET", "/dashboard/orders", nil)
	req = req.WithContext(ctx)

	rr = httptest.NewRecorder()
	handler.OrdersPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "No orders yet")
	assert.Contains(t, rr.Body.String(), "You haven't purchased any tickets yet")

	mockOrderService.AssertExpectations(t)
}

func testErrorHandling(t *testing.T) {
	mockOrderService := new(MockOrderService)
	mockEventService := new(MockEventServiceForDashboard)
	mockTicketService := new(MockTicketServiceForDashboard)
	
	handler := NewDashboardHandler(mockOrderService, mockEventService, mockTicketService)

	user := createTestUser()

	// Test 1: Invalid order ID
	req := httptest.NewRequest("GET", "/dashboard/orders/invalid", nil)
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.OrderDetailsPage(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// Test 2: Order not found
	mockOrderService.On("GetOrderByID", 999).Return(nil, fmt.Errorf("order not found"))

	req = httptest.NewRequest("GET", "/dashboard/orders/999", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rr = httptest.NewRecorder()
	handler.OrderDetailsPage(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)

	// Test 3: Ticket download for pending order
	order := createTestOrder()
	order.Status = models.OrderPending
	mockOrderService.On("GetOrderByID", 1).Return(order, nil)

	req = httptest.NewRequest("GET", "/dashboard/orders/1/tickets/download", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rr = httptest.NewRecorder()
	handler.DownloadTickets(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Tickets are not available for this order")

	// Test 4: PDF generation failure
	completedOrder := createTestOrder()
	completedOrder.ID = 2
	event := createTestEvent()
	tickets := createTestTickets()
	
	mockOrderService.On("GetOrderByID", 2).Return(completedOrder, nil)
	mockEventService.On("GetEventByID", 1).Return(event, nil)
	mockTicketService.On("GetTicketsByOrderID", 2).Return(tickets, nil)
	mockTicketService.On("GenerateTicketsPDF", tickets, event, completedOrder).Return([]byte{}, fmt.Errorf("PDF generation failed"))

	req = httptest.NewRequest("GET", "/dashboard/orders/2/tickets/download", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", "2")
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rr = httptest.NewRecorder()
	handler.DownloadTickets(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	mockOrderService.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
	mockTicketService.AssertExpectations(t)
}