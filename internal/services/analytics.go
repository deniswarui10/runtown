package services

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"event-ticketing-platform/internal/repositories"
)

// AnalyticsService handles analytics and reporting operations
type AnalyticsService struct {
	db              *sql.DB
	orderRepo       *repositories.OrderRepository
	eventRepo       *repositories.EventRepository
	ticketRepo      *repositories.TicketRepository
	userRepo        *repositories.UserRepository
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(db *sql.DB, orderRepo *repositories.OrderRepository, eventRepo *repositories.EventRepository, ticketRepo *repositories.TicketRepository, userRepo *repositories.UserRepository) *AnalyticsService {
	return &AnalyticsService{
		db:         db,
		orderRepo:  orderRepo,
		eventRepo:  eventRepo,
		ticketRepo: ticketRepo,
		userRepo:   userRepo,
	}
}



// GetOrganizerDashboard retrieves dashboard data for an organizer
func (s *AnalyticsService) GetOrganizerDashboard(organizerID int) (*OrganizerDashboardData, error) {
	dashboard := &OrganizerDashboardData{}

	// Get event counts by status
	eventCounts, err := s.getEventCountsByStatus(organizerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event counts: %w", err)
	}
	dashboard.TotalEvents = eventCounts["total"]
	dashboard.PublishedEvents = eventCounts["published"]
	dashboard.DraftEvents = eventCounts["draft"]

	// Get overall revenue and order statistics
	revenueStats, err := s.getOrganizerRevenueStats(organizerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue stats: %w", err)
	}
	dashboard.TotalRevenue = revenueStats["revenue"]
	dashboard.TotalOrders = int(revenueStats["orders"])
	dashboard.TotalTicketsSold = int(revenueStats["tickets"])

	// Get recent events
	dashboard.RecentEvents, err = s.getRecentEvents(organizerID, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}

	// Get top performing events
	dashboard.TopEvents, err = s.getTopPerformingEvents(organizerID, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get top events: %w", err)
	}

	// Get revenue by month (last 12 months)
	dashboard.RevenueByMonth, err = s.getRevenueByMonth(organizerID, 12)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly revenue: %w", err)
	}

	// Get sales over time (last 30 days)
	dashboard.SalesOverTime, err = s.getSalesOverTime(organizerID, 30)
	if err != nil {
		return nil, fmt.Errorf("failed to get sales over time: %w", err)
	}

	return dashboard, nil
}

// GetEventAnalytics retrieves detailed analytics for a specific event
func (s *AnalyticsService) GetEventAnalytics(eventID int, organizerID int) (*EventAnalyticsData, error) {
	// Verify organizer owns the event
	canAccess, err := s.canOrganizerAccessEvent(eventID, organizerID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify event access: %w", err)
	}
	if !canAccess {
		return nil, fmt.Errorf("organizer does not have access to this event")
	}

	analytics := &EventAnalyticsData{}

	// Get event details
	analytics.Event, err = s.eventRepo.GetByID(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	// Get overall event statistics
	eventStats, err := s.getEventStatistics(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event statistics: %w", err)
	}
	analytics.TotalRevenue = eventStats["revenue"]
	analytics.TotalOrders = int(eventStats["orders"])
	analytics.TotalTicketsSold = int(eventStats["tickets_sold"])
	analytics.TotalTicketsAvailable = int(eventStats["tickets_available"])
	
	if analytics.TotalTicketsAvailable > 0 {
		analytics.SoldOutPercentage = (float64(analytics.TotalTicketsSold) / float64(analytics.TotalTicketsAvailable)) * 100
	}

	// Get ticket type breakdown
	analytics.TicketTypeBreakdown, err = s.getTicketTypeAnalytics(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket type analytics: %w", err)
	}

	// Get sales by day
	analytics.SalesByDay, err = s.getEventSalesByDay(eventID, 30)
	if err != nil {
		return nil, fmt.Errorf("failed to get sales by day: %w", err)
	}

	// Get order status breakdown
	analytics.OrderStatusBreakdown, err = s.getOrderStatusBreakdown(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order status breakdown: %w", err)
	}

	// Get recent orders
	filters := repositories.OrderSearchFilters{
		EventID: eventID,
		Limit:   10,
		SortBy:  "created_at",
		SortDesc: true,
	}
	analytics.RecentOrders, _, err = s.orderRepo.GetOrdersWithDetails(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent orders: %w", err)
	}

	// Get attendee data
	analytics.AttendeeData, err = s.getAttendeeData(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attendee data: %w", err)
	}

	return analytics, nil
}

// ExportAttendeeData exports attendee data as CSV
func (s *AnalyticsService) ExportAttendeeData(eventID int, organizerID int) ([]byte, error) {
	// Verify organizer owns the event
	canAccess, err := s.canOrganizerAccessEvent(eventID, organizerID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify event access: %w", err)
	}
	if !canAccess {
		return nil, fmt.Errorf("organizer does not have access to this event")
	}

	// Get attendee data
	attendees, err := s.getAttendeeData(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attendee data: %w", err)
	}

	// Create CSV
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)

	// Write header
	header := []string{
		"Order Number",
		"Attendee Name",
		"Email",
		"Ticket Count",
		"Total Amount",
		"Order Date",
		"Status",
	}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, attendee := range attendees {
		row := []string{
			attendee.OrderNumber,
			attendee.BillingName,
			attendee.BillingEmail,
			fmt.Sprintf("%d", attendee.TicketCount),
			fmt.Sprintf("%.2f", attendee.TotalAmount),
			attendee.OrderDate.Format("2006-01-02 15:04:05"),
			attendee.Status,
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("failed to flush CSV writer: %w", err)
	}

	return []byte(csvData.String()), nil
}

// Helper methods

func (s *AnalyticsService) getEventCountsByStatus(organizerID int) (map[string]int, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'published' THEN 1 END) as published,
			COUNT(CASE WHEN status = 'draft' THEN 1 END) as draft,
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled
		FROM events 
		WHERE organizer_id = $1`

	var total, published, draft, cancelled int
	err := s.db.QueryRow(query, organizerID).Scan(&total, &published, &draft, &cancelled)
	if err != nil {
		return nil, err
	}

	return map[string]int{
		"total":     total,
		"published": published,
		"draft":     draft,
		"cancelled": cancelled,
	}, nil
}

func (s *AnalyticsService) getOrganizerRevenueStats(organizerID int) (map[string]float64, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN o.status = 'completed' THEN o.total_amount END), 0) as revenue,
			COUNT(CASE WHEN o.status = 'completed' THEN 1 END) as orders,
			COUNT(t.id) as tickets
		FROM events e
		LEFT JOIN orders o ON e.id = o.event_id
		LEFT JOIN tickets t ON o.id = t.order_id AND o.status = 'completed'
		WHERE e.organizer_id = $1`

	var revenue, orders, tickets int
	err := s.db.QueryRow(query, organizerID).Scan(&revenue, &orders, &tickets)
	if err != nil {
		return nil, err
	}

	return map[string]float64{
		"revenue": float64(revenue) / 100.0, // Convert cents to dollars
		"orders":  float64(orders),
		"tickets": float64(tickets),
	}, nil
}

func (s *AnalyticsService) getRecentEvents(organizerID int, limit int) ([]*EventSummary, error) {
	query := `
		SELECT 
			e.id, e.title, e.start_date, e.status,
			COUNT(t.id) as tickets_sold,
			COALESCE(SUM(CASE WHEN o.status = 'completed' THEN o.total_amount END), 0) as revenue,
			COUNT(CASE WHEN o.status = 'completed' THEN 1 END) as order_count
		FROM events e
		LEFT JOIN orders o ON e.id = o.event_id
		LEFT JOIN tickets t ON o.id = t.order_id AND o.status = 'completed'
		WHERE e.organizer_id = $1
		GROUP BY e.id, e.title, e.start_date, e.status
		ORDER BY e.created_at DESC
		LIMIT $2`

	rows, err := s.db.Query(query, organizerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*EventSummary
	for rows.Next() {
		event := &EventSummary{}
		var revenue int
		err := rows.Scan(
			&event.ID,
			&event.Title,
			&event.StartDate,
			&event.Status,
			&event.TicketsSold,
			&revenue,
			&event.OrderCount,
		)
		if err != nil {
			return nil, err
		}
		event.Revenue = float64(revenue) / 100.0
		events = append(events, event)
	}

	return events, rows.Err()
}

func (s *AnalyticsService) getTopPerformingEvents(organizerID int, limit int) ([]*EventPerformance, error) {
	query := `
		SELECT 
			e.id, e.title, e.start_date,
			COALESCE(SUM(tt.quantity), 0) as total_tickets,
			COUNT(t.id) as tickets_sold,
			COALESCE(SUM(CASE WHEN o.status = 'completed' THEN o.total_amount END), 0) as revenue,
			COUNT(CASE WHEN o.status = 'completed' THEN 1 END) as order_count
		FROM events e
		LEFT JOIN ticket_types tt ON e.id = tt.event_id
		LEFT JOIN orders o ON e.id = o.event_id
		LEFT JOIN tickets t ON o.id = t.order_id AND o.status = 'completed'
		WHERE e.organizer_id = $1 AND e.status = 'published'
		GROUP BY e.id, e.title, e.start_date
		HAVING COUNT(t.id) > 0
		ORDER BY revenue DESC
		LIMIT $2`

	rows, err := s.db.Query(query, organizerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*EventPerformance
	for rows.Next() {
		event := &EventPerformance{}
		var revenue int
		err := rows.Scan(
			&event.ID,
			&event.Title,
			&event.StartDate,
			&event.TotalTickets,
			&event.TicketsSold,
			&revenue,
			&event.OrderCount,
		)
		if err != nil {
			return nil, err
		}
		event.Revenue = float64(revenue) / 100.0
		
		if event.TotalTickets > 0 {
			event.ConversionRate = (float64(event.TicketsSold) / float64(event.TotalTickets)) * 100
		}
		
		if event.OrderCount > 0 {
			event.AverageOrderValue = event.Revenue / float64(event.OrderCount)
		}
		
		events = append(events, event)
	}

	return events, rows.Err()
}

func (s *AnalyticsService) getRevenueByMonth(organizerID int, months int) ([]*MonthlyRevenue, error) {
	query := `
		SELECT 
			DATE_TRUNC('month', o.created_at) as month,
			COALESCE(SUM(CASE WHEN o.status = 'completed' THEN o.total_amount END), 0) as revenue,
			COUNT(CASE WHEN o.status = 'completed' THEN 1 END) as orders,
			COUNT(t.id) as tickets
		FROM events e
		LEFT JOIN orders o ON e.id = o.event_id
		LEFT JOIN tickets t ON o.id = t.order_id AND o.status = 'completed'
		WHERE e.organizer_id = $1 
			AND o.created_at >= DATE_TRUNC('month', CURRENT_DATE - INTERVAL '%d months')
		GROUP BY DATE_TRUNC('month', o.created_at)
		ORDER BY month DESC`

	formattedQuery := fmt.Sprintf(query, months)
	rows, err := s.db.Query(formattedQuery, organizerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monthlyData []*MonthlyRevenue
	for rows.Next() {
		data := &MonthlyRevenue{}
		var month time.Time
		var revenue int
		err := rows.Scan(&month, &revenue, &data.Orders, &data.Tickets)
		if err != nil {
			return nil, err
		}
		data.Month = month.Format("January")
		data.Year = month.Year()
		data.Revenue = float64(revenue) / 100.0
		monthlyData = append(monthlyData, data)
	}

	return monthlyData, rows.Err()
}

func (s *AnalyticsService) getSalesOverTime(organizerID int, days int) ([]*DailySales, error) {
	query := `
		SELECT 
			DATE(o.created_at) as date,
			COALESCE(SUM(CASE WHEN o.status = 'completed' THEN o.total_amount END), 0) as revenue,
			COUNT(CASE WHEN o.status = 'completed' THEN 1 END) as orders,
			COUNT(t.id) as tickets
		FROM events e
		LEFT JOIN orders o ON e.id = o.event_id
		LEFT JOIN tickets t ON o.id = t.order_id AND o.status = 'completed'
		WHERE e.organizer_id = $1 
			AND o.created_at >= CURRENT_DATE - INTERVAL '%d days'
		GROUP BY DATE(o.created_at)
		ORDER BY date DESC`

	formattedQuery := fmt.Sprintf(query, days)
	rows, err := s.db.Query(formattedQuery, organizerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dailyData []*DailySales
	for rows.Next() {
		data := &DailySales{}
		var date time.Time
		var revenue int
		err := rows.Scan(&date, &revenue, &data.Orders, &data.Tickets)
		if err != nil {
			return nil, err
		}
		data.Date = date.Format("2006-01-02")
		data.Revenue = float64(revenue) / 100.0
		dailyData = append(dailyData, data)
	}

	return dailyData, rows.Err()
}

func (s *AnalyticsService) canOrganizerAccessEvent(eventID int, organizerID int) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM events WHERE id = $1 AND organizer_id = $2`
	err := s.db.QueryRow(query, eventID, organizerID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *AnalyticsService) getEventStatistics(eventID int) (map[string]float64, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN o.status = 'completed' THEN o.total_amount END), 0) as revenue,
			COUNT(CASE WHEN o.status = 'completed' THEN 1 END) as orders,
			COUNT(t.id) as tickets_sold,
			COALESCE(SUM(tt.quantity), 0) as tickets_available
		FROM events e
		LEFT JOIN ticket_types tt ON e.id = tt.event_id
		LEFT JOIN orders o ON e.id = o.event_id
		LEFT JOIN tickets t ON o.id = t.order_id AND o.status = 'completed'
		WHERE e.id = $1
		GROUP BY e.id`

	var revenue, orders, ticketsSold, ticketsAvailable int
	err := s.db.QueryRow(query, eventID).Scan(&revenue, &orders, &ticketsSold, &ticketsAvailable)
	if err != nil {
		return nil, err
	}

	return map[string]float64{
		"revenue":           float64(revenue) / 100.0,
		"orders":            float64(orders),
		"tickets_sold":      float64(ticketsSold),
		"tickets_available": float64(ticketsAvailable),
	}, nil
}

func (s *AnalyticsService) getTicketTypeAnalytics(eventID int) ([]*TicketTypeAnalytics, error) {
	query := `
		SELECT 
			tt.id, tt.name, tt.price, tt.quantity,
			COUNT(t.id) as tickets_sold,
			COALESCE(SUM(CASE WHEN o.status = 'completed' THEN tt.price END), 0) as revenue
		FROM ticket_types tt
		LEFT JOIN tickets t ON tt.id = t.ticket_type_id
		LEFT JOIN orders o ON t.order_id = o.id
		WHERE tt.event_id = $1
		GROUP BY tt.id, tt.name, tt.price, tt.quantity
		ORDER BY tt.price DESC`

	rows, err := s.db.Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analytics []*TicketTypeAnalytics
	for rows.Next() {
		data := &TicketTypeAnalytics{}
		var price, revenue int
		err := rows.Scan(
			&data.ID,
			&data.Name,
			&price,
			&data.TotalTickets,
			&data.TicketsSold,
			&revenue,
		)
		if err != nil {
			return nil, err
		}
		data.Price = float64(price) / 100.0
		data.Revenue = float64(revenue) / 100.0
		
		if data.TotalTickets > 0 {
			data.SoldOutPercentage = (float64(data.TicketsSold) / float64(data.TotalTickets)) * 100
		}
		
		analytics = append(analytics, data)
	}

	return analytics, rows.Err()
}

func (s *AnalyticsService) getEventSalesByDay(eventID int, days int) ([]*DailySales, error) {
	query := `
		SELECT 
			DATE(o.created_at) as date,
			COALESCE(SUM(CASE WHEN o.status = 'completed' THEN o.total_amount END), 0) as revenue,
			COUNT(CASE WHEN o.status = 'completed' THEN 1 END) as orders,
			COUNT(t.id) as tickets
		FROM orders o
		LEFT JOIN tickets t ON o.id = t.order_id AND o.status = 'completed'
		WHERE o.event_id = $1 
			AND o.created_at >= CURRENT_DATE - INTERVAL '%d days'
		GROUP BY DATE(o.created_at)
		ORDER BY date DESC`

	formattedQuery := fmt.Sprintf(query, days)
	rows, err := s.db.Query(formattedQuery, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dailyData []*DailySales
	for rows.Next() {
		data := &DailySales{}
		var date time.Time
		var revenue int
		err := rows.Scan(&date, &revenue, &data.Orders, &data.Tickets)
		if err != nil {
			return nil, err
		}
		data.Date = date.Format("2006-01-02")
		data.Revenue = float64(revenue) / 100.0
		dailyData = append(dailyData, data)
	}

	return dailyData, rows.Err()
}

func (s *AnalyticsService) getOrderStatusBreakdown(eventID int) (map[string]int, error) {
	query := `
		SELECT 
			status,
			COUNT(*) as count
		FROM orders
		WHERE event_id = $1
		GROUP BY status`

	rows, err := s.db.Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	breakdown := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		err := rows.Scan(&status, &count)
		if err != nil {
			return nil, err
		}
		breakdown[status] = count
	}

	return breakdown, rows.Err()
}

func (s *AnalyticsService) getAttendeeData(eventID int) ([]*AttendeeInfo, error) {
	query := `
		SELECT 
			o.id, o.order_number, o.billing_name, o.billing_email,
			COUNT(t.id) as ticket_count, o.total_amount, o.created_at, o.status
		FROM orders o
		LEFT JOIN tickets t ON o.id = t.order_id
		WHERE o.event_id = $1 AND o.status = 'completed'
		GROUP BY o.id, o.order_number, o.billing_name, o.billing_email, o.total_amount, o.created_at, o.status
		ORDER BY o.created_at DESC`

	rows, err := s.db.Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attendees []*AttendeeInfo
	for rows.Next() {
		attendee := &AttendeeInfo{}
		var totalAmount int
		err := rows.Scan(
			&attendee.OrderID,
			&attendee.OrderNumber,
			&attendee.BillingName,
			&attendee.BillingEmail,
			&attendee.TicketCount,
			&totalAmount,
			&attendee.OrderDate,
			&attendee.Status,
		)
		if err != nil {
			return nil, err
		}
		attendee.TotalAmount = float64(totalAmount) / 100.0
		attendees = append(attendees, attendee)
	}

	return attendees, rows.Err()
}

// GetOrganizerBalance calculates the available balance for an organizer
func (s *AnalyticsService) GetOrganizerBalance(organizerID int) (float64, error) {
	// Get organizer's total revenue from completed orders
	stats, err := s.orderRepo.GetOrderStatistics(&organizerID, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get order statistics: %w", err)
	}

	totalRevenue := 0.0
	if revenue, ok := stats["total_revenue"].(float64); ok {
		totalRevenue = revenue
	}

	// TODO: Subtract already withdrawn amounts
	// For now, return the total revenue as available balance
	return totalRevenue, nil
}