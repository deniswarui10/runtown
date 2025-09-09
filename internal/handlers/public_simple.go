package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"

	"github.com/go-chi/chi/v5"
)

// SimplePublicHandler handles public pages with simple HTML responses
type SimplePublicHandler struct {
	eventService  services.EventServiceInterface
	ticketService services.TicketServiceInterface
}

// NewSimplePublicHandler creates a new simple public handler
func NewSimplePublicHandler(eventService services.EventServiceInterface, ticketService services.TicketServiceInterface) *SimplePublicHandler {
	return &SimplePublicHandler{
		eventService:  eventService,
		ticketService: ticketService,
	}
}

// HomePage renders a simple homepage
func (h *SimplePublicHandler) HomePage(w http.ResponseWriter, r *http.Request) {
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

	// Render simple HTML response
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>EventHub - Home</title>
    <link href="/static/css/output.css" rel="stylesheet">
</head>
<body class="bg-gray-50">
    <nav class="bg-white shadow-sm border-b border-gray-200">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div class="flex justify-between h-16">
                <div class="flex items-center">
                    <span class="text-2xl font-bold text-primary-600">EventHub</span>
                </div>
                <div class="flex items-center space-x-4">
                    %s
                </div>
            </div>
        </div>
    </nav>
    
    <main>
        <section class="bg-gradient-to-r from-primary-600 to-primary-800 text-white">
            <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-20">
                <div class="text-center">
                    <h1 class="text-4xl md:text-6xl font-bold mb-6">Discover Amazing Events</h1>
                    <p class="text-xl md:text-2xl mb-8 text-primary-100">
                        Find and book tickets for concerts, conferences, workshops, and more
                    </p>
                </div>
            </div>
        </section>
        
        <section class="py-16 bg-white">
            <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                <h2 class="text-3xl font-bold text-gray-900 mb-8">Featured Events</h2>
                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
                    %s
                </div>
            </div>
        </section>
        
        <section class="py-16 bg-gray-50">
            <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                <h2 class="text-3xl font-bold text-gray-900 mb-8">Upcoming Events</h2>
                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                    %s
                </div>
            </div>
        </section>
    </main>
</body>
</html>`, 
		h.renderUserNav(user),
		h.renderEventCards(featuredEvents),
		h.renderEventCards(upcomingEvents))
}

// EventsListPage renders a simple events listing page
func (h *SimplePublicHandler) EventsListPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	// Parse query parameters
	query := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")
	location := r.URL.Query().Get("location")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	
	// Parse pagination
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Build search filters
	filters := services.EventSearchFilters{
		Query:    query,
		Category: category,
		Location: location,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Page:     page,
		PerPage:  12,
	}

	// Search events
	events, _, err := h.eventService.SearchEvents(filters)
	if err != nil {
		http.Error(w, "Failed to search events", http.StatusInternalServerError)
		return
	}

	// Check if this is an HTMX request for partial update
	if r.Header.Get("HX-Request") == "true" {
		// Return just the events list partial
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">%s</div>`, 
			h.renderEventCards(events))
		return
	}

	// Get categories for filters (only for full page requests)
	categories, err := h.eventService.GetCategories()
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	// Render the full events page
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Browse Events - EventHub</title>
    <link href="/static/css/output.css" rel="stylesheet">
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body class="bg-gray-50">
    <nav class="bg-white shadow-sm border-b border-gray-200">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div class="flex justify-between h-16">
                <div class="flex items-center">
                    <a href="/" class="text-2xl font-bold text-primary-600">EventHub</a>
                </div>
                <div class="flex items-center space-x-4">
                    %s
                </div>
            </div>
        </div>
    </nav>
    
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <h1 class="text-3xl font-bold text-gray-900 mb-8">Browse Events</h1>
        
        <!-- Filters -->
        <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 mb-6">
            <h3 class="text-lg font-medium text-gray-900 mb-4">Filter Events</h3>
            <form hx-get="/events" hx-target="#events-list" class="grid grid-cols-1 md:grid-cols-4 gap-4">
                <input type="text" name="q" value="%s" placeholder="Search events..." class="form-input">
                <select name="category" class="form-input">
                    <option value="">All Categories</option>
                    %s
                </select>
                <input type="text" name="location" value="%s" placeholder="Location" class="form-input">
                <button type="submit" class="btn btn-primary">Search</button>
            </form>
        </div>
        
        <!-- Events List -->
        <div id="events-list">
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
                %s
            </div>
        </div>
    </main>
</body>
</html>`, 
		h.renderUserNav(user),
		query,
		h.renderCategoryOptions(categories, category),
		location,
		h.renderEventCards(events))
}

// EventDetailsPage renders a simple event details page
func (h *SimplePublicHandler) EventDetailsPage(w http.ResponseWriter, r *http.Request) {
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

	// Get ticket types for this event
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

	// Render the event details page
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - EventHub</title>
    <link href="/static/css/output.css" rel="stylesheet">
</head>
<body class="bg-gray-50">
    <nav class="bg-white shadow-sm border-b border-gray-200">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div class="flex justify-between h-16">
                <div class="flex items-center">
                    <a href="/" class="text-2xl font-bold text-primary-600">EventHub</a>
                </div>
                <div class="flex items-center space-x-4">
                    %s
                </div>
            </div>
        </div>
    </nav>
    
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
            <div class="lg:col-span-2">
                <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                    <h1 class="text-3xl font-bold text-gray-900 mb-2">%s</h1>
                    <p class="text-lg text-gray-600 mb-4">Organized by %s %s</p>
                    <p class="text-gray-700 mb-4">%s</p>
                    <p class="text-gray-600">Location: %s</p>
                    <p class="text-gray-600">Date: %s</p>
                </div>
            </div>
            
            <div class="lg:col-span-1">
                <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                    <h3 class="text-lg font-semibold text-gray-900 mb-4">Tickets</h3>
                    %s
                </div>
            </div>
        </div>
    </main>
</body>
</html>`, 
		event.Title,
		h.renderUserNav(user),
		event.Title,
		organizer.FirstName, organizer.LastName,
		event.Description,
		event.Location,
		event.StartDate.Format("January 2, 2006 at 3:04 PM"),
		h.renderTicketTypes(ticketTypes, user))
}

// Helper methods for rendering HTML components
func (h *SimplePublicHandler) renderUserNav(user *models.User) string {
	if user != nil {
		return fmt.Sprintf(`
			<span class="text-gray-700">Hello, %s</span>
			<a href="/dashboard" class="text-gray-700 hover:text-primary-600">Dashboard</a>
			<a href="/auth/logout" class="text-gray-700 hover:text-primary-600">Logout</a>
		`, user.FirstName)
	}
	return `
		<a href="/auth/login" class="text-gray-700 hover:text-primary-600">Login</a>
		<a href="/auth/register" class="bg-primary-600 hover:bg-primary-700 text-white px-4 py-2 rounded-lg">Sign Up</a>
	`
}

func (h *SimplePublicHandler) renderEventCards(events []*models.Event) string {
	if len(events) == 0 {
		return `<p class="text-gray-600">No events found.</p>`
	}

	html := ""
	for _, event := range events {
		html += fmt.Sprintf(`
			<div class="bg-white rounded-lg shadow-md overflow-hidden hover:shadow-lg transition-shadow">
				<div class="p-6">
					<h3 class="text-lg font-semibold text-gray-900 mb-2">
						<a href="/events/%d" class="hover:text-primary-600">%s</a>
					</h3>
					<p class="text-gray-600 text-sm mb-4">%s</p>
					<div class="flex items-center justify-between">
						<span class="text-sm text-gray-500">%s</span>
						<a href="/events/%d" class="bg-primary-600 hover:bg-primary-700 text-white px-4 py-2 rounded-lg text-sm">
							View Details
						</a>
					</div>
				</div>
			</div>
		`, event.ID, event.Title, event.Description, event.StartDate.Format("Jan 2, 2006"), event.ID)
	}
	return html
}

func (h *SimplePublicHandler) renderCategoryOptions(categories []*models.Category, selected string) string {
	html := ""
	for _, category := range categories {
		selectedAttr := ""
		if category.Slug == selected {
			selectedAttr = " selected"
		}
		html += fmt.Sprintf(`<option value="%s"%s>%s</option>`, category.Slug, selectedAttr, category.Name)
	}
	return html
}

func (h *SimplePublicHandler) renderTicketTypes(ticketTypes []*models.TicketType, user *models.User) string {
	if len(ticketTypes) == 0 {
		return `<p class="text-gray-600">No tickets available for this event.</p>`
	}

	html := ""
	for _, ticketType := range ticketTypes {
		available := ticketType.Quantity - ticketType.Sold
		html += fmt.Sprintf(`
			<div class="border border-gray-200 rounded-lg p-4 mb-4">
				<div class="flex justify-between items-start mb-2">
					<div>
						<h4 class="font-medium text-gray-900">%s</h4>
						<p class="text-sm text-gray-600">%s</p>
					</div>
					<span class="text-lg font-bold text-primary-600">$%.2f</span>
				</div>
				<div class="flex items-center justify-between">
					<span class="text-sm text-gray-500">%d available</span>
					%s
				</div>
			</div>
		`, ticketType.Name, ticketType.Description, float64(ticketType.Price)/100, available, h.renderTicketButton(available, user))
	}
	return html
}

func (h *SimplePublicHandler) renderTicketButton(available int, user *models.User) string {
	if available <= 0 {
		return `<span class="text-sm text-red-600 font-medium">Sold Out</span>`
	}
	
	if user == nil {
		return `<a href="/auth/login" class="bg-primary-600 hover:bg-primary-700 text-white px-4 py-2 rounded-lg text-sm">Login to Purchase</a>`
	}
	
	return `<button class="bg-primary-600 hover:bg-primary-700 text-white px-4 py-2 rounded-lg text-sm">Select Tickets</button>`
}

// SearchEvents handles HTMX search requests
func (h *SimplePublicHandler) SearchEvents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	
	if query == "" {
		// Return empty results for empty query
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="text-center py-4"><p class="text-gray-600">Enter a search term to find events.</p></div>`)
		return
	}

	// Search events with a limit for dropdown results
	filters := services.EventSearchFilters{
		Query:   query,
		Page:    1,
		PerPage: 10, // Limit results for dropdown
	}

	events, _, err := h.eventService.SearchEvents(filters)
	if err != nil {
		http.Error(w, "Failed to search events", http.StatusInternalServerError)
		return
	}

	// Render search results
	w.Header().Set("Content-Type", "text/html")
	if len(events) == 0 {
		fmt.Fprintf(w, `<div class="text-center py-4"><p class="text-gray-600">No events found for "%s".</p></div>`, query)
		return
	}

	html := `<div class="divide-y divide-gray-200">`
	for _, event := range events {
		html += fmt.Sprintf(`
			<a href="/events/%d" class="block p-4 hover:bg-gray-50 transition-colors">
				<div class="flex items-center space-x-4">
					<div class="flex-1 min-w-0">
						<h4 class="text-sm font-medium text-gray-900 truncate">%s</h4>
						<p class="text-sm text-gray-500">%s • %s</p>
					</div>
				</div>
			</a>
		`, event.ID, event.Title, event.StartDate.Format("Jan 2, 2006"), event.Location)
	}
	html += `</div>`
	
	fmt.Fprint(w, html)
}