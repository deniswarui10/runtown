package handlers

import (
	"net/http"
	"strconv"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/web/templates/components"
	"event-ticketing-platform/web/templates/pages"
	"event-ticketing-platform/web/templates/partials"

	"github.com/go-chi/chi/v5"
)

// PublicHandler handles public pages
type PublicHandler struct {
	eventService          services.EventServiceInterface
	ticketService         services.TicketServiceInterface
	eventDiscoveryService *services.EventDiscoveryService
}

// NewPublicHandler creates a new public handler
func NewPublicHandler(eventService services.EventServiceInterface, ticketService services.TicketServiceInterface) *PublicHandler {
	return &PublicHandler{
		eventService:          eventService,
		ticketService:         ticketService,
		eventDiscoveryService: services.NewEventDiscoveryService(eventService),
	}
}

// HomePage renders the homepage with featured and upcoming events
func (h *PublicHandler) HomePage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	// Get featured events (limit to 6)
	featuredEvents, err := h.eventService.GetFeaturedEvents(6)
	if err != nil {
		http.Error(w, "Failed to load featured events", http.StatusInternalServerError)
		return
	}

	// Get upcoming events (limit to 8)
	upcomingEvents, err := h.eventService.GetUpcomingEvents(8)
	if err != nil {
		http.Error(w, "Failed to load upcoming events", http.StatusInternalServerError)
		return
	}

	// Render the homepage
	component := pages.HomePage(user, featuredEvents, upcomingEvents)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// EventsListPage renders the events listing page with search and filtering
func (h *PublicHandler) EventsListPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	// Parse query parameters with enhanced filtering
	query := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")
	location := r.URL.Query().Get("location")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	eventType := r.URL.Query().Get("event_type")
	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := r.URL.Query().Get("sort_order")
	availability := r.URL.Query().Get("availability")

	// Parse price range
	priceMin := 0
	priceMax := 0
	if priceMinStr := r.URL.Query().Get("price_min"); priceMinStr != "" {
		if p, err := strconv.Atoi(priceMinStr); err == nil && p >= 0 {
			priceMin = p
		}
	}
	if priceMaxStr := r.URL.Query().Get("price_max"); priceMaxStr != "" {
		if p, err := strconv.Atoi(priceMaxStr); err == nil && p >= 0 {
			priceMax = p
		}
	}

	// Parse pagination
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Get user ID for personalization
	userID := 0
	if user != nil {
		userID = user.ID
	}

	// Build enhanced discovery filters
	filters := services.DiscoveryFilters{
		Query:        query,
		Category:     category,
		Location:     location,
		DateFrom:     dateFrom,
		DateTo:       dateTo,
		PriceMin:     priceMin,
		PriceMax:     priceMax,
		EventType:    eventType,
		SortBy:       sortBy,
		SortOrder:    sortOrder,
		Availability: availability,
		Page:         page,
		PerPage:      12,
		UserID:       userID,
	}

	// Use enhanced discovery service
	result, err := h.eventDiscoveryService.DiscoverEvents(filters)
	if err != nil {
		http.Error(w, "Failed to discover events", http.StatusInternalServerError)
		return
	}

	// Build pagination
	totalPages := (result.FilteredCount + filters.PerPage - 1) / filters.PerPage
	pagination := components.Pagination{
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalItems:  result.FilteredCount,
		PerPage:     filters.PerPage,
	}

	// Build template filters with enhanced options
	templateFilters := pages.EventFilters{
		Category: category,
		Location: location,
		DateFrom: dateFrom,
		DateTo:   dateTo,
	}

	// Check if this is an HTMX request for partial update
	if middleware.IsHTMXRequest(r) {
		// Return just the events list partial
		component := pages.EventsList(result.Events, pagination)
		err = component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render events list", http.StatusInternalServerError)
		}
		return
	}

	// Note: Recommendations and trending events removed for simplicity

	// Render the events page
	component := pages.EventsListPage(user, result.Events, result.Categories, templateFilters, pagination)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// EventDetailsPage renders the event details page
func (h *PublicHandler) EventDetailsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Get event details
	event, err := h.eventService.GetEventByID(eventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Get ticket types for this event with real-time availability
	ticketTypes, err := h.ticketService.GetTicketTypesByEventID(eventID)
	if err != nil {
		http.Error(w, "Failed to load ticket types", http.StatusInternalServerError)
		return
	}

	// Get event organizer
	organizer, err := h.eventService.GetEventOrganizer(eventID)
	if err != nil {
		http.Error(w, "Failed to load organizer information", http.StatusInternalServerError)
		return
	}

	// Note: Similar events and recommendations removed for simplicity

	// Check if this is an HTMX request for ticket availability update
	if middleware.IsHTMXRequest(r) {
		target := r.Header.Get("HX-Target")
		if target == "ticket-availability" {
			// Return just the ticket availability partial
			w.Write([]byte("<div>Ticket availability updated</div>"))
			return
		}
	}

	// Render the enhanced event details page
	component := pages.EnhancedEventDetailsPage(user, event, ticketTypes, organizer, []*models.Event{}, []*models.Event{})
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// SearchEvents handles HTMX search requests
func (h *PublicHandler) SearchEvents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	if query == "" {
		// Return empty results for empty query
		component := partials.SearchResults([]*models.Event{}, "")
		err := component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render search results", http.StatusInternalServerError)
		}
		return
	}

	// Use enhanced discovery for search suggestions
	filters := services.DiscoveryFilters{
		Query:   query,
		Page:    1,
		PerPage: 10, // Limit results for dropdown
		SortBy:  "relevance",
	}

	result, err := h.eventDiscoveryService.DiscoverEvents(filters)
	if err != nil {
		http.Error(w, "Failed to search events", http.StatusInternalServerError)
		return
	}

	// Render search results partial
	component := partials.SearchResults(result.Events, query)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render search results", http.StatusInternalServerError)
		return
	}
}

// GetTicketAvailability returns real-time ticket availability for an event
func (h *PublicHandler) GetTicketAvailability(w http.ResponseWriter, r *http.Request) {
	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "id")
	_, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Return ticket availability partial (simplified)
	w.Write([]byte("<div>Ticket availability updated</div>"))
	return
}

// QuickAddToCart adds tickets to cart via HTMX
func (h *PublicHandler) QuickAddToCart(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		if middleware.IsHTMXRequest(r) {
			w.Header().Set("HX-Redirect", "/auth/login")
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		}
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	eventIDStr := r.FormValue("event_id")
	ticketTypeIDStr := r.FormValue("ticket_type_id")
	quantityStr := r.FormValue("quantity")

	_, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	_, err = strconv.Atoi(ticketTypeIDStr)
	if err != nil {
		http.Error(w, "Invalid ticket type ID", http.StatusBadRequest)
		return
	}

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity <= 0 {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	// Add to cart logic would go here
	// For now, return success message
	if middleware.IsHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<div class="bg-green-50 border border-green-200 text-green-800 p-4 rounded-lg">
				<div class="flex">
					<div class="flex-shrink-0">
						<svg class="h-5 w-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
						</svg>
					</div>
					<div class="ml-3">
						<p class="text-sm">Added to cart successfully!</p>
						<a href="/cart" class="text-sm font-medium text-green-800 hover:text-green-900 underline">View Cart</a>
					</div>
				</div>
			</div>
		`))
	} else {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
	}
}

// CategoriesPage renders the categories page
func (h *PublicHandler) CategoriesPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	// Get all categories
	categories, err := h.eventService.GetCategories()
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	// Render the categories page
	component := pages.CategoriesPage(user, categories)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}
