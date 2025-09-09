package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/internal/types"
	"event-ticketing-platform/web/templates/pages"

	"github.com/go-chi/chi/v5"
)

// DashboardHandler handles dashboard-related requests
type DashboardHandler struct {
	orderService  services.OrderServiceInterface
	eventService  services.EventServiceInterface
	ticketService services.TicketServiceInterface
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(orderService services.OrderServiceInterface, eventService services.EventServiceInterface, ticketService services.TicketServiceInterface) *DashboardHandler {
	return &DashboardHandler{
		orderService:  orderService,
		eventService:  eventService,
		ticketService: ticketService,
	}
}

// DashboardPage renders the main dashboard page
func (h *DashboardHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	fmt.Printf("[DEBUG] Dashboard handler - User from context: %+v\n", user)
	if user == nil {
		fmt.Printf("[DEBUG] Dashboard handler - No user found, redirecting to login\n")
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	fmt.Printf("[DEBUG] Dashboard handler - User found: ID=%d, Email=%s\n", user.ID, user.Email)

	// Get user's orders (recent 10)
	recentOrdersWithDetails, _, err := h.orderService.GetUserOrders(user.ID, 10, 0)
	if err != nil {
		recentOrdersWithDetails = []*repositories.OrderWithDetails{}
	}

	// Get all user's orders to calculate totals
	allOrdersWithDetails, _, err := h.orderService.GetUserOrders(user.ID, 1000, 0) // Get up to 1000 orders for calculation
	if err != nil {
		allOrdersWithDetails = []*repositories.OrderWithDetails{}
	}

	// Calculate total spent and total tickets
	totalSpent := 0.0
	totalTickets := 0
	for _, orderDetail := range allOrdersWithDetails {
		if orderDetail.Order.Status == models.OrderCompleted {
			// Convert from cents to currency
			totalSpent += float64(orderDetail.Order.TotalAmount) / 100.0
			// Count tickets in this order
			tickets, err := h.ticketService.GetTicketsByOrderID(orderDetail.Order.ID)
			if err == nil {
				totalTickets += len(tickets)
			}
		}
	}

	// Get upcoming events for the user
	upcomingEvents, err := h.getUpcomingEventsForUser(user.ID)
	if err != nil {
		upcomingEvents = []*models.Event{}
	}

	// Get past events for the user
	pastEvents, err := h.getPastEventsForUser(user.ID)
	if err != nil {
		pastEvents = []*models.Event{}
	}

	// For now, use empty recommended events (could be enhanced with ML later)
	recommendedEvents := []*models.Event{}

	// Create dashboard data
	dashboardData := &types.AttendeeDashboardData{
		TotalOrders:       len(allOrdersWithDetails), // Use all orders for total count
		TotalSpent:        totalSpent,
		TotalTickets:      totalTickets,
		UpcomingEvents:    upcomingEvents,
		PastEvents:        pastEvents,
		RecentOrders:      recentOrdersWithDetails, // Use recent orders for display
		RecommendedEvents: recommendedEvents,
	}

	component := pages.AttendeeDashboard(user, dashboardData)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render dashboard", http.StatusInternalServerError)
		return
	}
}

// OrdersPage renders the user's orders page with enhanced filtering and pagination
func (h *DashboardHandler) OrdersPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Parse query parameters for filtering and pagination
	filters := h.parseOrderFilters(r)

	// Build search filters
	searchFilters := repositories.OrderSearchFilters{
		UserID:   user.ID,
		Limit:    filters.PerPage,
		Offset:   (filters.Page - 1) * filters.PerPage,
		SortBy:   "created_at",
		SortDesc: true,
	}

	// Apply status filter
	if filters.Status != "" {
		searchFilters.Status = models.OrderStatus(filters.Status)
	}

	// Apply date filters
	if filters.DateFrom != "" {
		if dateFrom, err := time.Parse("2006-01-02", filters.DateFrom); err == nil {
			searchFilters.DateFrom = &dateFrom
		}
	}
	if filters.DateTo != "" {
		if dateTo, err := time.Parse("2006-01-02", filters.DateTo); err == nil {
			// Set to end of day
			dateTo = dateTo.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			searchFilters.DateTo = &dateTo
		}
	}

	// Get orders with details
	ordersWithDetails, total, err := h.orderService.SearchUserOrders(user.ID, searchFilters, user.ID)
	if err != nil {
		http.Error(w, "Failed to load orders", http.StatusInternalServerError)
		return
	}

	// Filter by event name if specified (post-database filtering for simplicity)
	if filters.EventName != "" {
		filteredOrders := make([]*repositories.OrderWithDetails, 0)
		for _, orderDetail := range ordersWithDetails {
			if strings.Contains(strings.ToLower(orderDetail.EventTitle), strings.ToLower(filters.EventName)) {
				filteredOrders = append(filteredOrders, orderDetail)
			}
		}
		ordersWithDetails = filteredOrders
		total = len(filteredOrders) // Approximate total for event name filtering
	}

	// Get upcoming events for sidebar
	upcomingEvents, err := h.getUpcomingEventsForUser(user.ID)
	if err != nil {
		// Log error but don't fail the page
		upcomingEvents = []*models.Event{}
	}

	// Check if this is an HTMX request for partial update
	if r.Header.Get("HX-Request") == "true" {
		// Render only the content part
		component := pages.OrderHistoryContent(ordersWithDetails, total, filters)
		err = component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render orders content", http.StatusInternalServerError)
		}
		return
	}

	// Render full page
	component := pages.OrderHistoryPage(user, ordersWithDetails, total, filters, upcomingEvents)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render orders page", http.StatusInternalServerError)
		return
	}
}

// OrderDetailsPage renders the enhanced order details page
func (h *DashboardHandler) OrderDetailsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Get order ID from URL
	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	// Get order details
	order, err := h.orderService.GetOrderByID(orderID, user.ID)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Get event details
	event, err := h.eventService.GetEventByID(order.EventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Get tickets for this order
	tickets, err := h.ticketService.GetTicketsByOrderID(orderID)
	if err != nil {
		http.Error(w, "Failed to load tickets", http.StatusInternalServerError)
		return
	}

	// Get ticket types for additional information
	ticketTypes := make(map[int]*models.TicketType)
	eventTicketTypes, err := h.ticketService.GetTicketTypesByEventID(event.ID)
	if err == nil {
		for _, tt := range eventTicketTypes {
			ticketTypes[tt.ID] = tt
		}
	}

	// Render enhanced order details page
	component := pages.OrderDetailsEnhancedPage(user, order, event, tickets, ticketTypes)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render order details", http.StatusInternalServerError)
		return
	}
}

// DownloadTickets handles ticket download requests
func (h *DashboardHandler) DownloadTickets(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Get order ID from URL
	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	// Get order details
	order, err := h.orderService.GetOrderByID(orderID, user.ID)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Verify order is completed
	if order.Status != models.OrderCompleted {
		http.Error(w, "Tickets are not available for this order", http.StatusBadRequest)
		return
	}

	// Get tickets for this order
	tickets, err := h.ticketService.GetTicketsByOrderID(orderID)
	if err != nil {
		http.Error(w, "Failed to load tickets", http.StatusInternalServerError)
		return
	}

	// Get event details
	event, err := h.eventService.GetEventByID(order.EventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Generate PDF tickets
	pdfData, err := h.ticketService.GenerateTicketsPDF(tickets, event, order)
	if err != nil {
		http.Error(w, "Failed to generate tickets PDF", http.StatusInternalServerError)
		return
	}

	// Set headers for PDF download
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"tickets-"+order.OrderNumber+".pdf\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfData)))

	// Write PDF data
	_, err = w.Write(pdfData)
	if err != nil {
		// Log error but can't return error response at this point
		return
	}
}

// OrderConfirmationPage renders the order confirmation page
func (h *DashboardHandler) OrderConfirmationPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Get order ID from URL
	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	// Get order details
	order, err := h.orderService.GetOrderByID(orderID, user.ID)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Get event details
	event, err := h.eventService.GetEventByID(order.EventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Get tickets for this order
	tickets, err := h.ticketService.GetTicketsByOrderID(orderID)
	if err != nil {
		http.Error(w, "Failed to load tickets", http.StatusInternalServerError)
		return
	}

	// Render order confirmation page
	component := pages.OrderConfirmationPage(user, order, tickets, event)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render order confirmation", http.StatusInternalServerError)
		return
	}
}

// DownloadSingleTicket handles single ticket download requests with enhanced validation
func (h *DashboardHandler) DownloadSingleTicket(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Get ticket ID from URL
	ticketIDStr := chi.URLParam(r, "id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	// Get ticket details
	ticket, err := h.ticketService.GetTicketByID(ticketID)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	}

	// Get order to verify ownership
	order, err := h.orderService.GetOrderByID(ticket.OrderID, user.ID)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Verify order is completed
	if order.Status != models.OrderCompleted {
		http.Error(w, "Ticket is not available for download", http.StatusBadRequest)
		return
	}

	// Verify ticket is active (not refunded)
	if ticket.Status == models.TicketRefunded {
		http.Error(w, "This ticket has been refunded and is no longer valid", http.StatusBadRequest)
		return
	}

	// Get event details
	event, err := h.eventService.GetEventByID(order.EventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Generate PDF for single ticket with enhanced formatting
	pdfData, err := h.ticketService.GenerateTicketsPDF([]*models.Ticket{ticket}, event, order)
	if err != nil {
		http.Error(w, "Failed to generate ticket PDF", http.StatusInternalServerError)
		return
	}

	// Set headers for PDF download with better filename
	filename := fmt.Sprintf("ticket-%s-%s.pdf", order.OrderNumber, ticket.QRCode[:8])
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfData)))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Write PDF data
	_, err = w.Write(pdfData)
	if err != nil {
		// Log error but can't return error response at this point
		return
	}
}

// RedownloadTickets handles ticket re-download requests for lost tickets
func (h *DashboardHandler) RedownloadTickets(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Get order ID from URL
	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	// Get order details with enhanced validation
	order, err := h.orderService.GetOrderByID(orderID, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Order not found", http.StatusNotFound)
		} else if strings.Contains(err.Error(), "insufficient permissions") {
			http.Error(w, "Access denied", http.StatusForbidden)
		} else {
			http.Error(w, "Failed to retrieve order", http.StatusInternalServerError)
		}
		return
	}

	// Verify order is completed
	if order.Status != models.OrderCompleted {
		http.Error(w, "Tickets are not available for this order status", http.StatusBadRequest)
		return
	}

	// Get tickets for this order
	tickets, err := h.ticketService.GetTicketsByOrderID(orderID)
	if err != nil {
		http.Error(w, "Failed to load tickets", http.StatusInternalServerError)
		return
	}

	// Filter out refunded tickets
	var validTickets []*models.Ticket
	for _, ticket := range tickets {
		if ticket.Status != models.TicketRefunded {
			validTickets = append(validTickets, ticket)
		}
	}

	if len(validTickets) == 0 {
		http.Error(w, "No valid tickets available for download", http.StatusBadRequest)
		return
	}

	// Get event details
	event, err := h.eventService.GetEventByID(order.EventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Generate PDF tickets with enhanced formatting
	pdfData, err := h.ticketService.GenerateTicketsPDF(validTickets, event, order)
	if err != nil {
		http.Error(w, "Failed to generate tickets PDF", http.StatusInternalServerError)
		return
	}

	// Set headers for PDF download with timestamp for re-downloads
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("tickets-redownload-%s-%s.pdf", order.OrderNumber, timestamp)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfData)))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Write PDF data
	_, err = w.Write(pdfData)
	if err != nil {
		// Log error but can't return error response at this point
		return
	}
}

// getUpcomingEventsForUser gets events that the user has tickets for with enhanced filtering
func (h *DashboardHandler) getUpcomingEventsForUser(userID int) ([]*models.Event, error) {
	// Get user's orders with a higher limit to ensure we get all orders
	ordersWithDetails, _, err := h.orderService.GetUserOrders(userID, 200, 0)
	if err != nil {
		return nil, err
	}

	// Get unique event IDs from completed orders only
	eventIDs := make(map[int]bool)
	for _, orderDetail := range ordersWithDetails {
		if orderDetail.Order.Status == models.OrderCompleted {
			eventIDs[orderDetail.Order.EventID] = true
		}
	}

	// Get events and filter for upcoming ones with enhanced sorting
	var upcomingEvents []*models.Event
	now := time.Now()

	for eventID := range eventIDs {
		event, err := h.eventService.GetEventByID(eventID)
		if err != nil {
			continue // Skip events that can't be loaded
		}

		// Only include upcoming events (events that haven't started yet)
		if event.StartDate.After(now) {
			upcomingEvents = append(upcomingEvents, event)
		}
	}

	// Sort upcoming events by start date (earliest first)
	for i := 0; i < len(upcomingEvents)-1; i++ {
		for j := i + 1; j < len(upcomingEvents); j++ {
			if upcomingEvents[i].StartDate.After(upcomingEvents[j].StartDate) {
				upcomingEvents[i], upcomingEvents[j] = upcomingEvents[j], upcomingEvents[i]
			}
		}
	}

	// Limit to top 10 upcoming events for performance
	if len(upcomingEvents) > 10 {
		upcomingEvents = upcomingEvents[:10]
	}

	return upcomingEvents, nil
}

// parseOrderFilters parses query parameters for order filtering
func (h *DashboardHandler) parseOrderFilters(r *http.Request) pages.OrderHistoryFilters {
	filters := pages.OrderHistoryFilters{
		Status:    strings.TrimSpace(r.URL.Query().Get("status")),
		DateFrom:  strings.TrimSpace(r.URL.Query().Get("date_from")),
		DateTo:    strings.TrimSpace(r.URL.Query().Get("date_to")),
		EventName: strings.TrimSpace(r.URL.Query().Get("event_name")),
		Page:      1,
		PerPage:   20,
	}

	// Parse page
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filters.Page = page
		}
	}

	// Parse per_page
	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if perPage, err := strconv.Atoi(perPageStr); err == nil && perPage > 0 && perPage <= 100 {
			filters.PerPage = perPage
		}
	}

	return filters
}

// getPastEventsForUser gets events that the user has attended (past events)
func (h *DashboardHandler) getPastEventsForUser(userID int) ([]*models.Event, error) {
	// Get user's orders
	ordersWithDetails, _, err := h.orderService.GetUserOrders(userID, 200, 0)
	if err != nil {
		return nil, err
	}

	// Get unique event IDs from completed orders only
	eventIDs := make(map[int]bool)
	for _, orderDetail := range ordersWithDetails {
		if orderDetail.Order.Status == models.OrderCompleted {
			eventIDs[orderDetail.Order.EventID] = true
		}
	}

	// Get events and filter for past ones
	var pastEvents []*models.Event
	now := time.Now()

	for eventID := range eventIDs {
		event, err := h.eventService.GetEventByID(eventID)
		if err != nil {
			continue // Skip events that can't be loaded
		}

		// Only include past events (events that have ended)
		if event.EndDate.Before(now) {
			pastEvents = append(pastEvents, event)
		}
	}

	// Sort past events by end date (most recent first)
	for i := 0; i < len(pastEvents)-1; i++ {
		for j := i + 1; j < len(pastEvents); j++ {
			if pastEvents[i].EndDate.Before(pastEvents[j].EndDate) {
				pastEvents[i], pastEvents[j] = pastEvents[j], pastEvents[i]
			}
		}
	}

	// Limit to top 10 past events
	if len(pastEvents) > 10 {
		pastEvents = pastEvents[:10]
	}

	return pastEvents, nil
}

// CancelOrder handles order cancellation requests
func (h *DashboardHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get order ID from URL
	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	// Cancel the order
	err = h.orderService.CancelOrder(orderID, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Order not found", http.StatusNotFound)
		} else if strings.Contains(err.Error(), "cannot be cancelled") {
			http.Error(w, "Order cannot be cancelled", http.StatusBadRequest)
		} else {
			http.Error(w, "Failed to cancel order", http.StatusInternalServerError)
		}
		return
	}

	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// Redirect to orders page with success message
		w.Header().Set("HX-Redirect", "/dashboard/orders?cancelled=1")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Regular redirect
	http.Redirect(w, r, "/dashboard/orders?cancelled=1", http.StatusSeeOther)
}
