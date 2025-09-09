package services

import (
	"context"
	"io"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
)

// AuthServiceInterface defines the interface for authentication services
type AuthServiceInterface interface {
	Register(req *RegisterRequest) (*AuthResponse, error)
	Login(req *LoginRequest) (*AuthResponse, error)
	ValidateSession(sessionID string) (*models.User, error)
	Logout(sessionID string) error
	RequestPasswordReset(req *PasswordResetRequest) error
	ChangePassword(userID int, req *PasswordChangeRequest) error
	RequireRole(user *models.User, requiredRole models.UserRole) error
	RequireRoles(user *models.User, requiredRoles ...models.UserRole) error
	CleanupExpiredSessions() error
	VerifyEmail(token string) (*models.User, error)
	ExtendSession(sessionID string, duration time.Duration) error
	LogoutAllSessions(userID int) error
	ResendVerificationEmail(email string) error
	CompletePasswordReset(req *PasswordResetCompleteRequest) error
	ValidatePasswordResetToken(token string) (*models.User, error)
	CleanupExpiredTokens() error
}

// EventServiceInterface defines the interface for event services
type EventServiceInterface interface {
	GetFeaturedEvents(limit int) ([]*models.Event, error)
	GetUpcomingEvents(limit int) ([]*models.Event, error)
	SearchEvents(filters EventSearchFilters) ([]*models.Event, int, error)
	GetCategories() ([]*models.Category, error)
	GetEventByID(id int) (*models.Event, error)
	GetEventOrganizer(eventID int) (*models.User, error)
	CreateEvent(req *EventCreateRequest) (*models.Event, error)
	UpdateEvent(id int, req *EventUpdateRequest) (*models.Event, error)
	DeleteEvent(id int) error
	GetEventsByOrganizer(organizerID int) ([]*models.Event, error)
	CanUserEditEvent(eventID int, userID int) (bool, error)
	CanUserDeleteEvent(eventID int, userID int) (bool, error)
	UpdateEventStatus(eventID int, status models.EventStatus, organizerID int) (*models.Event, error)
	DuplicateEvent(eventID int, organizerID int, newTitle string, newStartDate, newEndDate time.Time) (*models.Event, error)

	// Admin-specific methods
	GetEventCount() (int, error)
	GetPublishedEventCount() (int, error)
}

// TicketServiceInterface defines the interface for ticket services
type TicketServiceInterface interface {
	GetTicketTypesByEventID(eventID int) ([]*models.TicketType, error)
	GetTicketTypeByID(id int) (*models.TicketType, error)
	CreateTicketType(req *models.TicketTypeCreateRequest) (*models.TicketType, error)
	UpdateTicketType(id int, req *models.TicketTypeUpdateRequest) (*models.TicketType, error)
	DeleteTicketType(id int) error
	GetTicketByID(id int) (*models.Ticket, error)
	GetTicketsByOrderID(orderID int) ([]*models.Ticket, error)
	GenerateTicketsPDF(tickets []*models.Ticket, event *models.Event, order *models.Order) ([]byte, error)
}

// OrderServiceInterface defines the interface for order services
type OrderServiceInterface interface {
	CreateOrder(req *models.OrderCreateRequest) (*models.Order, error)
	GetUserOrders(userID int, limit, offset int) ([]*repositories.OrderWithDetails, int, error)
	GetOrderByID(orderID int, requestingUserID int) (*models.Order, error)
	GetOrderWithTickets(orderID int, requestingUserID int) (*models.Order, []*models.Ticket, error)
	CancelOrder(orderID int, requestingUserID int) error
	GetEventOrders(eventID int, requestingUserID int, limit, offset int) ([]*repositories.OrderWithDetails, int, error)
	GetOrderStatistics(eventID *int, userID *int, requestingUserID int) (map[string]interface{}, error)
	SearchUserOrders(userID int, filters repositories.OrderSearchFilters, requestingUserID int) ([]*repositories.OrderWithDetails, int, error)
	GetUpcomingEventsForUser(userID int, requestingUserID int) ([]*models.Event, error)
	CompleteOrder(orderID int, paymentID string, ticketData []struct {
		TicketTypeID int
		QRCode       string
	}) error

	// Admin-specific methods
	GetOrderCount() (int, error)
	GetTotalRevenue() (float64, error)
}

// EmailServiceInterface defines the interface for email services
type EmailServiceInterface interface {
	SendPasswordResetEmail(email, token string) error
	SendWelcomeEmail(email, userName string) error
	SendVerificationEmail(email, userName, token string) error
	SendOrderConfirmation(email, userName, orderNumber, eventTitle, eventDate, totalAmount string) error
	SendOrderConfirmationWithTickets(email, userName, subject, htmlContent, textContent string, order *models.Order, tickets []*models.Ticket) error
}

// StorageServiceInterface defines the interface for file storage operations
type StorageServiceInterface interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error)
	Delete(ctx context.Context, key string) error
	GetURL(key string) string
	GeneratePresignedURL(ctx context.Context, key string, contentType string, expiration time.Duration) (string, error)
	Exists(ctx context.Context, key string) (bool, error)
}

// ImageServiceInterface defines the interface for image processing and storage
type ImageServiceInterface interface {
	UploadImage(ctx context.Context, reader io.Reader, filename string) (*ImageUploadResult, error)
	UploadImageWithOptions(ctx context.Context, reader io.Reader, filename string, options ImageProcessingOptions) (*ImageUploadResult, error)
	DeleteImage(ctx context.Context, keyPrefix string) error
	ValidateImage(reader io.Reader, maxSize int64) error
	GetImageURL(keyPrefix, variant string) string
	GetOptimalImageURL(keyPrefix, variant string, acceptHeader string) string
	GetImageVariants(keyPrefix string) []string
}

// EventSearchFilters represents search filters for events
type EventSearchFilters struct {
	Query    string
	Category string
	Location string
	DateFrom string
	DateTo   string
	Page     int
	PerPage  int
}

// TicketReservation represents a ticket reservation
type TicketReservation struct {
	ID        string
	EventID   int
	UserID    int
	Tickets   []*models.Ticket
	ExpiresAt string
	Total     int
}

// OrderCreateRequest represents a request to create an order
type OrderCreateRequest struct {
	UserID       int                   `json:"user_id"`
	EventID      int                   `json:"event_id"`
	TicketTypes  []OrderTicketTypeItem `json:"ticket_types"`
	BillingEmail string                `json:"billing_email"`
	BillingName  string                `json:"billing_name"`
}

// OrderTicketTypeItem represents a ticket type and quantity in an order
type OrderTicketTypeItem struct {
	TicketTypeID int `json:"ticket_type_id"`
	Quantity     int `json:"quantity"`
}

// AnalyticsServiceInterface defines the interface for analytics services
type AnalyticsServiceInterface interface {
	GetOrganizerDashboard(organizerID int) (*OrganizerDashboardData, error)
	GetEventAnalytics(eventID int, organizerID int) (*EventAnalyticsData, error)
	ExportAttendeeData(eventID int, organizerID int) ([]byte, error)
	GetOrganizerBalance(organizerID int) (float64, error)
}

// Analytics data types
type OrganizerDashboardData struct {
	TotalEvents      int                 `json:"total_events"`
	PublishedEvents  int                 `json:"published_events"`
	DraftEvents      int                 `json:"draft_events"`
	TotalRevenue     float64             `json:"total_revenue"`
	TotalOrders      int                 `json:"total_orders"`
	TotalTicketsSold int                 `json:"total_tickets_sold"`
	RecentEvents     []*EventSummary     `json:"recent_events"`
	TopEvents        []*EventPerformance `json:"top_events"`
	RevenueByMonth   []*MonthlyRevenue   `json:"revenue_by_month"`
	SalesOverTime    []*DailySales       `json:"sales_over_time"`
}

type EventAnalyticsData struct {
	Event                 *models.Event                    `json:"event"`
	TotalRevenue          float64                          `json:"total_revenue"`
	TotalOrders           int                              `json:"total_orders"`
	TotalTicketsSold      int                              `json:"total_tickets_sold"`
	TotalTicketsAvailable int                              `json:"total_tickets_available"`
	SoldOutPercentage     float64                          `json:"sold_out_percentage"`
	TicketTypeBreakdown   []*TicketTypeAnalytics           `json:"ticket_type_breakdown"`
	SalesByDay            []*DailySales                    `json:"sales_by_day"`
	OrderStatusBreakdown  map[string]int                   `json:"order_status_breakdown"`
	RecentOrders          []*repositories.OrderWithDetails `json:"recent_orders"`
	AttendeeData          []*AttendeeInfo                  `json:"attendee_data"`
}

type EventSummary struct {
	ID          int                `json:"id"`
	Title       string             `json:"title"`
	StartDate   time.Time          `json:"start_date"`
	Status      models.EventStatus `json:"status"`
	TicketsSold int                `json:"tickets_sold"`
	Revenue     float64            `json:"revenue"`
	OrderCount  int                `json:"order_count"`
}

type EventPerformance struct {
	ID                int       `json:"id"`
	Title             string    `json:"title"`
	StartDate         time.Time `json:"start_date"`
	TotalTickets      int       `json:"total_tickets"`
	TicketsSold       int       `json:"tickets_sold"`
	Revenue           float64   `json:"revenue"`
	OrderCount        int       `json:"order_count"`
	ConversionRate    float64   `json:"conversion_rate"`
	AverageOrderValue float64   `json:"average_order_value"`
}

type MonthlyRevenue struct {
	Month   string  `json:"month"`
	Year    int     `json:"year"`
	Revenue float64 `json:"revenue"`
	Orders  int     `json:"orders"`
	Tickets int     `json:"tickets"`
}

type DailySales struct {
	Date    string  `json:"date"`
	Revenue float64 `json:"revenue"`
	Orders  int     `json:"orders"`
	Tickets int     `json:"tickets"`
}

type TicketTypeAnalytics struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	Price             float64 `json:"price"`
	TotalTickets      int     `json:"total_tickets"`
	TicketsSold       int     `json:"tickets_sold"`
	Revenue           float64 `json:"revenue"`
	SoldOutPercentage float64 `json:"sold_out_percentage"`
}

type AttendeeInfo struct {
	OrderID      int       `json:"order_id"`
	OrderNumber  string    `json:"order_number"`
	BillingName  string    `json:"billing_name"`
	BillingEmail string    `json:"billing_email"`
	TicketCount  int       `json:"ticket_count"`
	TotalAmount  float64   `json:"total_amount"`
	OrderDate    time.Time `json:"order_date"`
	Status       string    `json:"status"`
}

// WithdrawalServiceInterface defines the interface for withdrawal services
type WithdrawalServiceInterface interface {
	CreateWithdrawal(organizerID int, req *models.WithdrawalCreateRequest) (*models.Withdrawal, error)
	GetOrganizerWithdrawals(organizerID int) ([]*models.Withdrawal, error)
	GetWithdrawalsByStatus(status string) ([]*models.Withdrawal, error)
	ProcessWithdrawal(withdrawalID int, status models.WithdrawalStatus, adminID int, adminNotes string) error
	GetWithdrawalByID(id int) (*models.Withdrawal, error)
}
