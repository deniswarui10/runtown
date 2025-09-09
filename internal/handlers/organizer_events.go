package handlers

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/web/templates/pages"

	"github.com/go-chi/chi/v5"
)

// OrganizerEventHandler handles organizer event management
type OrganizerEventHandler struct {
	eventService   services.EventServiceInterface
	ticketService  services.TicketServiceInterface
	storageService services.StorageService
	imageService   services.ImageServiceInterface
}

// NewOrganizerEventHandler creates a new organizer event handler
func NewOrganizerEventHandler(eventService services.EventServiceInterface, ticketService services.TicketServiceInterface, storageService services.StorageService, imageService services.ImageServiceInterface) *OrganizerEventHandler {
	return &OrganizerEventHandler{
		eventService:   eventService,
		ticketService:  ticketService,
		storageService: storageService,
		imageService:   imageService,
	}
}


// EventsListPage displays the organizer's events with filtering
func (h *OrganizerEventHandler) EventsListPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters for filtering
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	
	// Parse pagination (for future use)
	_ = 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			_ = p
		}
	}

	// Get organizer's events
	events, err := h.eventService.GetEventsByOrganizer(user.ID)
	if err != nil {
		http.Error(w, "Failed to load events", http.StatusInternalServerError)
		return
	}

	// Apply client-side filtering (in a real app, this would be done in the service/repository)
	filteredEvents := h.filterEvents(events, status, search)

	// Get categories for the create form
	categories, err := h.eventService.GetCategories()
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	// Check if this is an HTMX request for partial update
	if r.Header.Get("HX-Request") == "true" {
		// Return just the events list partial
		component := pages.OrganizerEventsListPartial(filteredEvents)
		err = component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render events list", http.StatusInternalServerError)
		}
		return
	}

	// Render the full organizer events page
	component := pages.OrganizerEventsPage(user, filteredEvents, categories, status, search)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// CreateEventPage displays the event creation form
func (h *OrganizerEventHandler) CreateEventPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user has organizer role or admin role
	if user.Role != models.UserRoleOrganizer && user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied - organizer role required", http.StatusForbidden)
		return
	}

	// Get categories for the form
	categories, err := h.eventService.GetCategories()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load categories: %v", err), http.StatusInternalServerError)
		return
	}

	// Render the create event page
	component := pages.CreateEventPage(user, categories, nil, nil)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to render page: %v", err), http.StatusInternalServerError)
		return
	}
}

// CreateEventSubmit handles event creation form submission
func (h *OrganizerEventHandler) CreateEventSubmit(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user has organizer role or admin role
	if user.Role != models.UserRoleOrganizer && user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied - organizer role required", http.StatusForbidden)
		return
	}

	// Parse form data
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
			http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}
	}

	// Extract form fields
	title := r.FormValue("title")
	description := r.FormValue("description")
	location := r.FormValue("location")
	startDateStr := r.FormValue("start_date")
	endDateStr := r.FormValue("end_date")
	categoryIDStr := r.FormValue("category_id")
	statusValue := r.FormValue("status")
	
	// Extract new fields
	eventType := r.FormValue("event_type")
	maxCapacityStr := r.FormValue("max_capacity")
	ticketName := r.FormValue("ticket_name")
	ticketPriceStr := r.FormValue("ticket_price")
	ticketQuantityStr := r.FormValue("ticket_quantity")
	saleEndDateStr := r.FormValue("sale_end_date")
	
	// Debug logging
	log.Printf("🔍 Form values received:")
	log.Printf("  title: %s", title)
	log.Printf("  status: %s", statusValue)
	log.Printf("  ticket_name: %s", ticketName)
	log.Printf("  ticket_price: %s", ticketPriceStr)

	// Validate required fields
	errors := make(map[string]string)
	if title == "" {
		errors["title"] = "Title is required"
	}
	if description == "" {
		errors["description"] = "Description is required"
	}
	if location == "" {
		errors["location"] = "Location is required"
	}
	if startDateStr == "" {
		errors["start_date"] = "Start date is required"
	}
	if endDateStr == "" {
		errors["end_date"] = "End date is required"
	}
	if categoryIDStr == "" {
		errors["category_id"] = "Category is required"
	}

	// Parse dates
	var startDate, endDate time.Time
	var err error
	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02T15:04", startDateStr)
		if err != nil {
			errors["start_date"] = "Invalid start date format"
		}
	}
	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02T15:04", endDateStr)
		if err != nil {
			errors["end_date"] = "Invalid end date format"
		}
	}

	// Validate date logic
	if !startDate.IsZero() && !endDate.IsZero() {
		if endDate.Before(startDate) {
			errors["end_date"] = "End date must be after start date"
		}
		if startDate.Before(time.Now()) {
			errors["start_date"] = "Start date must be in the future"
		}
	}

	// Parse category ID
	var categoryID int
	if categoryIDStr != "" {
		categoryID, err = strconv.Atoi(categoryIDStr)
		if err != nil {
			errors["category_id"] = "Invalid category"
		}
	}

	// Set status based on button clicked
	var status models.EventStatus
	if statusValue == "published" {
		status = models.StatusPublished
		log.Printf("🔍 Setting status to published")
	} else {
		status = models.StatusDraft
		log.Printf("🔍 Setting status to draft")
	}

	// Handle image upload
	var imageFile *multipart.FileHeader
	if file, fileHeader, err := r.FormFile("image"); err == nil {
		defer file.Close()
		imageFile = fileHeader
		log.Printf("🔍 Image file received: %s (size: %d bytes)", fileHeader.Filename, fileHeader.Size)
	} else {
		log.Printf("🔍 No image file received: %v", err)
	}

	// If there are validation errors, re-render the form
	if len(errors) > 0 {
		categories, _ := h.eventService.GetCategories()
		formData := map[string]interface{}{
			"title":            title,
			"description":      description,
			"location":         location,
			"start_date":       startDateStr,
			"end_date":         endDateStr,
			"category_id":      categoryIDStr,
			"status":           status,
			"event_type":       eventType,
			"max_capacity":     maxCapacityStr,
			"ticket_name":      ticketName,
			"ticket_price":     ticketPriceStr,
			"ticket_quantity":  ticketQuantityStr,
			"sale_end_date":    saleEndDateStr,
		}
		
		component := pages.CreateEventPage(user, categories, formData, errors)
		err = component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// Create the event
	createReq := &services.EventCreateRequest{
		Title:       title,
		Description: description,
		StartDate:   startDate,
		EndDate:     endDate,
		Location:    location,
		CategoryID:  categoryID,
		Status:      status,
		OrganizerID: user.ID,
		Image:       imageFile,
	}

	event, err := h.eventService.CreateEvent(createReq)
	if err != nil {
		// Handle service-level errors
		errors["general"] = fmt.Sprintf("Failed to create event: %v", err)
		categories, _ := h.eventService.GetCategories()
		formData := map[string]interface{}{
			"title":            title,
			"description":      description,
			"location":         location,
			"start_date":       startDateStr,
			"end_date":         endDateStr,
			"category_id":      categoryIDStr,
			"status":           status,
			"event_type":       eventType,
			"max_capacity":     maxCapacityStr,
			"ticket_name":      ticketName,
			"ticket_price":     ticketPriceStr,
			"ticket_quantity":  ticketQuantityStr,
			"sale_end_date":    saleEndDateStr,
		}
		
		component := pages.CreateEventPage(user, categories, formData, errors)
		err = component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// Create basic ticket type if provided
	if ticketName != "" && ticketQuantityStr != "" {
		// Parse ticket data
		var ticketPrice int // Price in cents
		var ticketQuantity int
		var saleEndDate time.Time
		
		if ticketPriceStr != "" {
			if priceFloat, err := strconv.ParseFloat(ticketPriceStr, 64); err == nil {
				ticketPrice = int(priceFloat * 100) // Convert to cents
			}
		}
		
		if quantity, err := strconv.Atoi(ticketQuantityStr); err == nil {
			ticketQuantity = quantity
		}
		
		if saleEndDateStr != "" {
			saleEndDate, _ = time.Parse("2006-01-02T15:04", saleEndDateStr)
		} else {
			// Default to event start time, but ensure it's at least 2 hours from now
			saleEndDate = startDate
			minEndTime := time.Now().Add(2 * time.Hour)
			if saleEndDate.Before(minEndTime) {
				saleEndDate = minEndTime
			}
		}
		
		// Create ticket type
		ticketReq := &models.TicketTypeCreateRequest{
			EventID:     event.ID,
			Name:        ticketName,
			Description: fmt.Sprintf("General admission ticket for %s", event.Title),
			Price:       ticketPrice,
			Quantity:    ticketQuantity,
			SaleStart:   time.Now().Add(-5 * time.Minute), // Start 5 minutes ago to ensure it's active
			SaleEnd:     saleEndDate,
		}
		
		_, err := h.ticketService.CreateTicketType(ticketReq)
		if err != nil {
			// Log error but don't fail the event creation
			// The user can still create ticket types manually
			fmt.Printf("Warning: Failed to create basic ticket type: %v\n", err)
		}
	}

	// Redirect to ticket type management for the new event
	http.Redirect(w, r, fmt.Sprintf("/organizer/events/%d/tickets/", event.ID), http.StatusSeeOther)
}

// EditEventPage displays the event editing form
func (h *OrganizerEventHandler) EditEventPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Check if user can edit this event
	canEdit, err := h.eventService.CanUserEditEvent(eventID, user.ID)
	if err != nil {
		http.Error(w, "Failed to check permissions", http.StatusInternalServerError)
		return
	}
	if !canEdit {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get event details
	event, err := h.eventService.GetEventByID(eventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Get categories for the form
	categories, err := h.eventService.GetCategories()
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	// Render the edit event page
	component := pages.EditEventPage(user, event, categories, nil, nil)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// UpdateEventSubmit handles event update form submission
func (h *OrganizerEventHandler) UpdateEventSubmit(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Check if user can edit this event
	canEdit, err := h.eventService.CanUserEditEvent(eventID, user.ID)
	if err != nil {
		http.Error(w, "Failed to check permissions", http.StatusInternalServerError)
		return
	}
	if !canEdit {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Parse form data
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
			http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}
	}

	// Extract form fields
	title := r.FormValue("title")
	description := r.FormValue("description")
	location := r.FormValue("location")
	startDateStr := r.FormValue("start_date")
	endDateStr := r.FormValue("end_date")
	categoryIDStr := r.FormValue("category_id")
	status := models.EventStatus(r.FormValue("status"))

	// Validate required fields
	errors := make(map[string]string)
	if title == "" {
		errors["title"] = "Title is required"
	}
	if description == "" {
		errors["description"] = "Description is required"
	}
	if location == "" {
		errors["location"] = "Location is required"
	}
	if startDateStr == "" {
		errors["start_date"] = "Start date is required"
	}
	if endDateStr == "" {
		errors["end_date"] = "End date is required"
	}
	if categoryIDStr == "" {
		errors["category_id"] = "Category is required"
	}

	// Parse dates
	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02T15:04", startDateStr)
		if err != nil {
			errors["start_date"] = "Invalid start date format"
		}
	}
	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02T15:04", endDateStr)
		if err != nil {
			errors["end_date"] = "Invalid end date format"
		}
	}

	// Validate date logic
	if !startDate.IsZero() && !endDate.IsZero() {
		if endDate.Before(startDate) {
			errors["end_date"] = "End date must be after start date"
		}
	}

	// Parse category ID
	var categoryID int
	if categoryIDStr != "" {
		categoryID, err = strconv.Atoi(categoryIDStr)
		if err != nil {
			errors["category_id"] = "Invalid category"
		}
	}

	// Handle image upload
	var imageFile *multipart.FileHeader
	if file, fileHeader, err := r.FormFile("image"); err == nil {
		defer file.Close()
		imageFile = fileHeader
	}

	// If there are validation errors, re-render the form
	if len(errors) > 0 {
		event, _ := h.eventService.GetEventByID(eventID)
		categories, _ := h.eventService.GetCategories()
		formData := map[string]interface{}{
			"title":       title,
			"description": description,
			"location":    location,
			"start_date":  startDateStr,
			"end_date":    endDateStr,
			"category_id": categoryIDStr,
			"status":      status,
		}
		
		component := pages.EditEventPage(user, event, categories, formData, errors)
		err = component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// Update the event
	updateReq := &services.EventUpdateRequest{
		Title:       title,
		Description: description,
		StartDate:   startDate,
		EndDate:     endDate,
		Location:    location,
		CategoryID:  categoryID,
		Status:      status,
		OrganizerID: user.ID,
		Image:       imageFile,
	}

	event, err := h.eventService.UpdateEvent(eventID, updateReq)
	if err != nil {
		// Handle service-level errors
		errors["general"] = fmt.Sprintf("Failed to update event: %v", err)
		event, _ := h.eventService.GetEventByID(eventID)
		categories, _ := h.eventService.GetCategories()
		formData := map[string]interface{}{
			"title":       title,
			"description": description,
			"location":    location,
			"start_date":  startDateStr,
			"end_date":    endDateStr,
			"category_id": categoryIDStr,
			"status":      status,
		}
		
		component := pages.EditEventPage(user, event, categories, formData, errors)
		err = component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// Redirect back to the edit page with success message
	http.Redirect(w, r, fmt.Sprintf("/organizer/events/%d/edit?success=updated", event.ID), http.StatusSeeOther)
}

// UpdateEventStatus handles HTMX requests to update event status
func (h *OrganizerEventHandler) UpdateEventStatus(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Get new status from form
	newStatus := models.EventStatus(r.FormValue("status"))
	if newStatus == "" {
		http.Error(w, "Status is required", http.StatusBadRequest)
		return
	}

	// Update event status
	event, err := h.eventService.UpdateEventStatus(eventID, newStatus, user.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update status: %v", err), http.StatusBadRequest)
		return
	}

	// Return updated status component
	component := pages.EventStatusBadge(event.Status)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render status", http.StatusInternalServerError)
		return
	}
}

// DeleteEvent handles event deletion with safety checks
func (h *OrganizerEventHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Check if user can delete this event
	canDelete, err := h.eventService.CanUserDeleteEvent(eventID, user.ID)
	if err != nil {
		http.Error(w, "Failed to check permissions", http.StatusInternalServerError)
		return
	}
	if !canDelete {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get event details for safety checks
	event, err := h.eventService.GetEventByID(eventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Safety check: Don't allow deletion of published events with tickets sold
	// (This would need to be implemented in the service layer with ticket sales data)
	if event.Status == models.StatusPublished {
		// In a real implementation, check if tickets have been sold
		http.Error(w, "Cannot delete published events with ticket sales", http.StatusBadRequest)
		return
	}

	// Delete the event
	err = h.eventService.DeleteEvent(eventID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete event: %v", err), http.StatusInternalServerError)
		return
	}

	// For HTMX requests, return empty response to remove the row
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Redirect to events list
	http.Redirect(w, r, "/organizer/events", http.StatusSeeOther)
}

// DuplicateEvent creates a copy of an existing event
func (h *OrganizerEventHandler) DuplicateEvent(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Get form data for the duplicate
	newTitle := r.FormValue("title")
	startDateStr := r.FormValue("start_date")
	endDateStr := r.FormValue("end_date")

	if newTitle == "" || startDateStr == "" || endDateStr == "" {
		http.Error(w, "Title and dates are required", http.StatusBadRequest)
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02T15:04", startDateStr)
	if err != nil {
		http.Error(w, "Invalid start date format", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse("2006-01-02T15:04", endDateStr)
	if err != nil {
		http.Error(w, "Invalid end date format", http.StatusBadRequest)
		return
	}

	// Duplicate the event
	duplicateEvent, err := h.eventService.DuplicateEvent(eventID, user.ID, newTitle, startDate, endDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to duplicate event: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to edit the duplicated event
	http.Redirect(w, r, fmt.Sprintf("/organizer/events/%d/edit", duplicateEvent.ID), http.StatusSeeOther)
}

// filterEvents applies client-side filtering to events
func (h *OrganizerEventHandler) filterEvents(events []*models.Event, status, search string) []*models.Event {
	var filtered []*models.Event

	for _, event := range events {
		// Status filter
		if status != "" && string(event.Status) != status {
			continue
		}

		// Search filter (title and description)
		if search != "" {
			searchLower := strings.ToLower(search)
			titleMatch := strings.Contains(strings.ToLower(event.Title), searchLower)
			descMatch := strings.Contains(strings.ToLower(event.Description), searchLower)
			if !titleMatch && !descMatch {
				continue
			}
		}

		filtered = append(filtered, event)
	}

	return filtered
}

// PublishEvent handles event publishing
func (h *OrganizerEventHandler) PublishEvent(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user has organizer role or admin role
	if user.Role != models.UserRoleOrganizer && user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied - organizer role required", http.StatusForbidden)
		return
	}

	// Get event ID from URL
	eventID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Update event status to published
	_, err = h.eventService.UpdateEventStatus(eventID, models.StatusPublished, user.ID)
	if err != nil {
		http.Error(w, "Failed to publish event", http.StatusInternalServerError)
		return
	}

	// Redirect back to events list
	http.Redirect(w, r, "/organizer/events", http.StatusSeeOther)
}

// UnpublishEvent handles event unpublishing
func (h *OrganizerEventHandler) UnpublishEvent(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user has organizer role or admin role
	if user.Role != models.UserRoleOrganizer && user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied - organizer role required", http.StatusForbidden)
		return
	}

	// Get event ID from URL
	eventID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Update event status to draft
	_, err = h.eventService.UpdateEventStatus(eventID, models.StatusDraft, user.ID)
	if err != nil {
		http.Error(w, "Failed to unpublish event", http.StatusInternalServerError)
		return
	}

	// Redirect back to events list
	http.Redirect(w, r, "/organizer/events", http.StatusSeeOther)
}