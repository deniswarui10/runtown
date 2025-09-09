package types

import (
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
)

// AttendeeDashboardData represents data for the attendee dashboard
type AttendeeDashboardData struct {
	TotalOrders       int
	TotalSpent        float64 // Total amount spent across all orders
	TotalTickets      int     // Total number of tickets purchased
	UpcomingEvents    []*models.Event
	PastEvents        []*models.Event
	RecentOrders      []*repositories.OrderWithDetails
	RecommendedEvents []*models.Event
}
