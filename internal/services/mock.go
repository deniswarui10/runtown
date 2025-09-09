package services

import (
	"fmt"
	"event-ticketing-platform/internal/models"
	"time"
)

// MockEventService provides mock event service for testing/demo
type MockEventService struct{}

func (m *MockEventService) GetFeaturedEvents(limit int) ([]*models.Event, error) {
	events := []*models.Event{
		{
			ID:          1,
			Title:       "Tech Conference 2024",
			Description: "Join us for the biggest tech conference of the year featuring industry leaders and cutting-edge innovations.",
			StartDate:   time.Now().AddDate(0, 1, 0),
			EndDate:     time.Now().AddDate(0, 1, 0).Add(8 * time.Hour),
			Location:    "San Francisco, CA",
			CategoryID:  1,
			OrganizerID: 1,
			ImageURL:    "",
			Status:      models.StatusPublished,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          2,
			Title:       "Music Festival Summer",
			Description: "Experience amazing live music from top artists in a beautiful outdoor setting.",
			StartDate:   time.Now().AddDate(0, 2, 0),
			EndDate:     time.Now().AddDate(0, 2, 2),
			Location:    "Austin, TX",
			CategoryID:  2,
			OrganizerID: 2,
			ImageURL:    "",
			Status:      models.StatusPublished,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          3,
			Title:       "Art Gallery Opening",
			Description: "Discover contemporary art from emerging and established artists in our new gallery space.",
			StartDate:   time.Now().AddDate(0, 0, 15),
			EndDate:     time.Now().AddDate(0, 0, 15).Add(4 * time.Hour),
			Location:    "New York, NY",
			CategoryID:  3,
			OrganizerID: 3,
			ImageURL:    "",
			Status:      models.StatusPublished,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	if limit > 0 && limit < len(events) {
		return events[:limit], nil
	}
	return events, nil
}

func (m *MockEventService) GetUpcomingEvents(limit int) ([]*models.Event, error) {
	events := []*models.Event{
		{
			ID:          4,
			Title:       "Business Workshop",
			Description: "Learn essential business skills from industry experts.",
			StartDate:   time.Now().AddDate(0, 0, 7),
			EndDate:     time.Now().AddDate(0, 0, 7).Add(6 * time.Hour),
			Location:    "Chicago, IL",
			CategoryID:  4,
			OrganizerID: 4,
			ImageURL:    "",
			Status:      models.StatusPublished,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          5,
			Title:       "Food & Wine Tasting",
			Description: "Enjoy an evening of fine dining and wine tasting with local chefs.",
			StartDate:   time.Now().AddDate(0, 0, 10),
			EndDate:     time.Now().AddDate(0, 0, 10).Add(3 * time.Hour),
			Location:    "Napa Valley, CA",
			CategoryID:  5,
			OrganizerID: 5,
			ImageURL:    "",
			Status:      models.StatusPublished,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          6,
			Title:       "Sports Tournament",
			Description: "Watch exciting matches in our annual sports tournament.",
			StartDate:   time.Now().AddDate(0, 0, 20),
			EndDate:     time.Now().AddDate(0, 0, 22),
			Location:    "Miami, FL",
			CategoryID:  6,
			OrganizerID: 6,
			ImageURL:    "",
			Status:      models.StatusPublished,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	if limit > 0 && limit < len(events) {
		return events[:limit], nil
	}
	return events, nil
}

func (m *MockEventService) SearchEvents(filters EventSearchFilters) ([]*models.Event, int, error) {
	allEvents, _ := m.GetFeaturedEvents(0)
	upcomingEvents, _ := m.GetUpcomingEvents(0)
	allEvents = append(allEvents, upcomingEvents...)

	// Simple filtering by query
	var filteredEvents []*models.Event
	if filters.Query == "" {
		filteredEvents = allEvents
	} else {
		for _, event := range allEvents {
			if contains(event.Title, filters.Query) || contains(event.Description, filters.Query) || contains(event.Location, filters.Query) {
				filteredEvents = append(filteredEvents, event)
			}
		}
	}

	// Simple pagination
	start := (filters.Page - 1) * filters.PerPage
	end := start + filters.PerPage
	
	if start >= len(filteredEvents) {
		return []*models.Event{}, len(filteredEvents), nil
	}
	
	if end > len(filteredEvents) {
		end = len(filteredEvents)
	}

	return filteredEvents[start:end], len(filteredEvents), nil
}

func (m *MockEventService) GetCategories() ([]*models.Category, error) {
	return []*models.Category{
		{ID: 1, Name: "Technology", Slug: "technology", Description: "Tech events and conferences"},
		{ID: 2, Name: "Music", Slug: "music", Description: "Concerts and music festivals"},
		{ID: 3, Name: "Arts", Slug: "arts", Description: "Art exhibitions and cultural events"},
		{ID: 4, Name: "Business", Slug: "business", Description: "Business workshops and networking"},
		{ID: 5, Name: "Food & Drink", Slug: "food", Description: "Food and beverage events"},
		{ID: 6, Name: "Sports", Slug: "sports", Description: "Sports events and tournaments"},
	}, nil
}

func (m *MockEventService) GetEventByID(id int) (*models.Event, error) {
	allEvents, _ := m.GetFeaturedEvents(0)
	upcomingEvents, _ := m.GetUpcomingEvents(0)
	allEvents = append(allEvents, upcomingEvents...)

	for _, event := range allEvents {
		if event.ID == id {
			return event, nil
		}
	}

	return nil, models.ErrEventNotFound
}

func (m *MockEventService) GetEventOrganizer(eventID int) (*models.User, error) {
	return &models.User{
		ID:        1,
		Email:     "organizer@example.com",
		FirstName: "John",
		LastName:  "Organizer",
		Role:      models.RoleOrganizer,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *MockEventService) CreateEvent(req *EventCreateRequest) (*models.Event, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockEventService) UpdateEvent(id int, req *EventUpdateRequest) (*models.Event, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockEventService) DeleteEvent(id int) error {
	return models.ErrNotImplemented
}

func (m *MockEventService) GetEventsByOrganizer(organizerID int) ([]*models.Event, error) {
	return []*models.Event{}, nil
}

func (m *MockEventService) CanUserEditEvent(eventID int, userID int) (bool, error) {
	// Mock implementation - allow editing for demo
	return true, nil
}

func (m *MockEventService) CanUserDeleteEvent(eventID int, userID int) (bool, error) {
	// Mock implementation - allow deletion for demo
	return true, nil
}

func (m *MockEventService) UpdateEventStatus(eventID int, status models.EventStatus, organizerID int) (*models.Event, error) {
	// Mock implementation - not implemented
	return nil, models.ErrNotImplemented
}

func (m *MockEventService) DuplicateEvent(eventID int, organizerID int, newTitle string, newStartDate, newEndDate time.Time) (*models.Event, error) {
	// Mock implementation - not implemented
	return nil, models.ErrNotImplemented
}

// MockTicketService provides mock ticket service for testing/demo
type MockTicketService struct{}

func (m *MockTicketService) GetTicketTypesByEventID(eventID int) ([]*models.TicketType, error) {
	return []*models.TicketType{
		{
			ID:          1,
			EventID:     eventID,
			Name:        "General Admission",
			Description: "Standard entry ticket",
			Price:       2500, // $25.00 in cents
			Quantity:    100,
			Sold:        25,
			SaleStart:   time.Now().AddDate(0, -1, 0),
			SaleEnd:     time.Now().AddDate(0, 1, 0),
			CreatedAt:   time.Now(),
		},
		{
			ID:          2,
			EventID:     eventID,
			Name:        "VIP Access",
			Description: "Premium experience with exclusive perks",
			Price:       7500, // $75.00 in cents
			Quantity:    20,
			Sold:        5,
			SaleStart:   time.Now().AddDate(0, -1, 0),
			SaleEnd:     time.Now().AddDate(0, 1, 0),
			CreatedAt:   time.Now(),
		},
	}, nil
}

func (m *MockTicketService) GetTicketTypeByID(id int) (*models.TicketType, error) {
	// Mock implementation - return a sample ticket type
	return &models.TicketType{
		ID:          id,
		EventID:     1,
		Name:        "General Admission",
		Description: "Standard entry ticket",
		Price:       2500, // $25.00 in cents
		Quantity:    100,
		Sold:        25,
		SaleStart:   time.Now().AddDate(0, -1, 0),
		SaleEnd:     time.Now().AddDate(0, 1, 0),
		CreatedAt:   time.Now(),
	}, nil
}

func (m *MockTicketService) CreateTicketType(req *models.TicketTypeCreateRequest) (*models.TicketType, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockTicketService) UpdateTicketType(id int, req *models.TicketTypeUpdateRequest) (*models.TicketType, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockTicketService) DeleteTicketType(id int) error {
	return models.ErrNotImplemented
}

func (m *MockTicketService) GetTicketByID(id int) (*models.Ticket, error) {
	return &models.Ticket{
		ID:           id,
		OrderID:      1,
		TicketTypeID: 1,
		QRCode:       "QR123456789",
		Status:       models.TicketActive,
		CreatedAt:    time.Now(),
	}, nil
}

func (m *MockTicketService) GetTicketsByOrderID(orderID int) ([]*models.Ticket, error) {
	return []*models.Ticket{
		{
			ID:           1,
			OrderID:      orderID,
			TicketTypeID: 1,
			QRCode:       "QR123456789",
			Status:       models.TicketActive,
			CreatedAt:    time.Now(),
		},
		{
			ID:           2,
			OrderID:      orderID,
			TicketTypeID: 1,
			QRCode:       "QR987654321",
			Status:       models.TicketActive,
			CreatedAt:    time.Now(),
		},
	}, nil
}

func (m *MockTicketService) GenerateTicketsPDF(tickets []*models.Ticket, event *models.Event, order *models.Order) ([]byte, error) {
	// Use the real PDF service for mock
	pdfService := NewPDFService()
	return pdfService.GenerateTicketsPDF(tickets, event, order)
}

// MockOrderService provides mock order service for testing/demo
type MockOrderService struct{}

func (m *MockOrderService) GetOrderByID(id int) (*models.Order, error) {
	return &models.Order{
		ID:           id,
		UserID:       1,
		EventID:      1,
		OrderNumber:  "ORD-123456",
		TotalAmount:  5000, // $50.00 in cents
		Status:       models.OrderCompleted,
		PaymentID:    "pay_123456",
		BillingEmail: "user@example.com",
		BillingName:  "John Doe",
		CreatedAt:    time.Now().AddDate(0, 0, -7),
		UpdatedAt:    time.Now().AddDate(0, 0, -7),
	}, nil
}

func (m *MockOrderService) GetOrdersByUserID(userID int, limit int) ([]*models.Order, error) {
	orders := []*models.Order{
		{
			ID:           1,
			UserID:       userID,
			EventID:      1,
			OrderNumber:  "ORD-123456",
			TotalAmount:  5000, // $50.00 in cents
			Status:       models.OrderCompleted,
			PaymentID:    "pay_123456",
			BillingEmail: "user@example.com",
			BillingName:  "John Doe",
			CreatedAt:    time.Now().AddDate(0, 0, -7),
			UpdatedAt:    time.Now().AddDate(0, 0, -7),
		},
		{
			ID:           2,
			UserID:       userID,
			EventID:      2,
			OrderNumber:  "ORD-789012",
			TotalAmount:  7500, // $75.00 in cents
			Status:       models.OrderCompleted,
			PaymentID:    "pay_789012",
			BillingEmail: "user@example.com",
			BillingName:  "John Doe",
			CreatedAt:    time.Now().AddDate(0, 0, -14),
			UpdatedAt:    time.Now().AddDate(0, 0, -14),
		},
		{
			ID:           3,
			UserID:       userID,
			EventID:      3,
			OrderNumber:  "ORD-345678",
			TotalAmount:  2500, // $25.00 in cents
			Status:       models.OrderPending,
			PaymentID:    "",
			BillingEmail: "user@example.com",
			BillingName:  "John Doe",
			CreatedAt:    time.Now().AddDate(0, 0, -2),
			UpdatedAt:    time.Now().AddDate(0, 0, -2),
		},
	}

	if limit > 0 && limit < len(orders) {
		return orders[:limit], nil
	}
	return orders, nil
}

func (m *MockOrderService) CreateOrder(req *OrderCreateRequest) (*models.Order, error) {
	return nil, models.ErrNotImplemented
}

func (m *MockOrderService) UpdateOrderStatus(id int, status models.OrderStatus) error {
	return models.ErrNotImplemented
}

// Helper function for string contains check (case insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && 
			(s[0] == substr[0] || s[0] == substr[0]+32 || s[0] == substr[0]-32) &&
			contains(s[1:], substr[1:])))
}

// Admin-specific methods for MockEventService
func (m *MockEventService) GetEventCount() (int, error) {
	return 10, nil // Mock count
}

func (m *MockEventService) GetPublishedEventCount() (int, error) {
	return 8, nil // Mock count
}

// MockWithdrawalService provides mock withdrawal service for testing/demo
type MockWithdrawalService struct{}

func (m *MockWithdrawalService) CreateWithdrawal(organizerID int, req *models.WithdrawalCreateRequest) (*models.Withdrawal, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockWithdrawalService) GetOrganizerWithdrawals(organizerID int) ([]*models.Withdrawal, error) {
	return []*models.Withdrawal{}, nil
}

func (m *MockWithdrawalService) GetWithdrawalsByStatus(status string) ([]*models.Withdrawal, error) {
	return []*models.Withdrawal{}, nil
}

func (m *MockWithdrawalService) ProcessWithdrawal(withdrawalID int, status models.WithdrawalStatus, adminID int, adminNotes string) error {
	return fmt.Errorf("not implemented")
}

func (m *MockWithdrawalService) GetWithdrawalByID(id int) (*models.Withdrawal, error) {
	return nil, fmt.Errorf("not implemented")
}