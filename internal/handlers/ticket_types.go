package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/web/templates/pages"

	"github.com/go-chi/chi/v5"
)

// TicketTypeHandler handles ticket type management for organizers
type TicketTypeHandler struct {
	ticketService services.TicketServiceInterface
	eventService  services.EventServiceInterface
}

// NewTicketTypeHandler creates a new ticket type handler
func NewTicketTypeHandler(ticketService services.TicketServiceInterface, eventService services.EventServiceInterface) *TicketTypeHandler {
	return &TicketTypeHandler{
		ticketService: ticketService,
		eventService:  eventService,
	}
}

// TicketTypesPage displays the ticket types for an event
func (h *TicketTypeHandler) TicketTypesPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "eventId")
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

	// Get ticket types for the event
	ticketTypes, err := h.ticketService.GetTicketTypesByEventID(eventID)
	if err != nil {
		http.Error(w, "Failed to load ticket types", http.StatusInternalServerError)
		return
	}

	// Check if this is an HTMX request for partial update
	if r.Header.Get("HX-Request") == "true" {
		// Return just the ticket types list partial
		component := pages.TicketTypesListPartial(ticketTypes)
		err = component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render ticket types list", http.StatusInternalServerError)
		}
		return
	}

	// Render the full ticket types page
	component := pages.TicketTypesPage(user, event, ticketTypes)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// CreateTicketTypePage displays the ticket type creation form
func (h *TicketTypeHandler) CreateTicketTypePage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "eventId")
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

	// Render the create ticket type page
	component := pages.CreateTicketTypePage(user, event, nil, nil)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// CreateTicketTypeSubmit handles ticket type creation form submission
func (h *TicketTypeHandler) CreateTicketTypeSubmit(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	eventIDStr := chi.URLParam(r, "eventId")
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
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Extract form fields
	name := r.FormValue("name")
	description := r.FormValue("description")
	priceStr := r.FormValue("price")
	quantityStr := r.FormValue("quantity")
	saleStartStr := r.FormValue("sale_start")
	saleEndStr := r.FormValue("sale_end")

	// Validate required fields
	errors := make(map[string]string)
	if name == "" {
		errors["name"] = "Ticket type name is required"
	}
	if priceStr == "" {
		errors["price"] = "Price is required"
	}
	if quantityStr == "" {
		errors["quantity"] = "Quantity is required"
	}
	if saleStartStr == "" {
		errors["sale_start"] = "Sale start date is required"
	}
	if saleEndStr == "" {
		errors["sale_end"] = "Sale end date is required"
	}

	// Parse numeric fields
	var price, quantity int
	if priceStr != "" {
		// Convert price from dollars to cents
		if priceFloat, err := strconv.ParseFloat(priceStr, 64); err != nil {
			errors["price"] = "Invalid price format"
		} else {
			price = int(priceFloat * 100) // Convert to cents
		}
	}
	if quantityStr != "" {
		if quantity, err = strconv.Atoi(quantityStr); err != nil {
			errors["quantity"] = "Invalid quantity format"
		}
	}

	// Parse dates
	var saleStart, saleEnd time.Time
	if saleStartStr != "" {
		saleStart, err = time.Parse("2006-01-02T15:04", saleStartStr)
		if err != nil {
			errors["sale_start"] = "Invalid sale start date format"
		}
	}
	if saleEndStr != "" {
		saleEnd, err = time.Parse("2006-01-02T15:04", saleEndStr)
		if err != nil {
			errors["sale_end"] = "Invalid sale end date format"
		}
	}

	// Validate date logic
	if !saleStart.IsZero() && !saleEnd.IsZero() {
		if saleEnd.Before(saleStart) {
			errors["sale_end"] = "Sale end date must be after sale start date"
		}
	}

	// If there are validation errors, re-render the form
	if len(errors) > 0 {
		event, _ := h.eventService.GetEventByID(eventID)
		formData := map[string]interface{}{
			"name":        name,
			"description": description,
			"price":       priceStr,
			"quantity":    quantityStr,
			"sale_start":  saleStartStr,
			"sale_end":    saleEndStr,
		}

		component := pages.CreateTicketTypePage(user, event, formData, errors)
		err = component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// Create the ticket type
	createReq := &models.TicketTypeCreateRequest{
		EventID:     eventID,
		Name:        name,
		Description: description,
		Price:       price,
		Quantity:    quantity,
		SaleStart:   saleStart,
		SaleEnd:     saleEnd,
	}

	_, err = h.ticketService.CreateTicketType(createReq)
	if err != nil {
		// Handle service-level errors
		errors["general"] = fmt.Sprintf("Failed to create ticket type: %v", err)
		event, _ := h.eventService.GetEventByID(eventID)
		formData := map[string]interface{}{
			"name":        name,
			"description": description,
			"price":       priceStr,
			"quantity":    quantityStr,
			"sale_start":  saleStartStr,
			"sale_end":    saleEndStr,
		}

		component := pages.CreateTicketTypePage(user, event, formData, errors)
		err = component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// Redirect to the ticket types list
	http.Redirect(w, r, fmt.Sprintf("/organizer/events/%d/tickets/", eventID), http.StatusSeeOther)
}

// EditTicketTypePage displays the ticket type editing form
func (h *TicketTypeHandler) EditTicketTypePage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID and ticket type ID from URL
	eventIDStr := chi.URLParam(r, "eventId")
	ticketTypeIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}
	ticketTypeID, err := strconv.Atoi(ticketTypeIDStr)
	if err != nil {
		http.Error(w, "Invalid ticket type ID", http.StatusBadRequest)
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

	// Get ticket type details
	ticketType, err := h.ticketService.GetTicketTypeByID(ticketTypeID)
	if err != nil {
		http.Error(w, "Ticket type not found", http.StatusNotFound)
		return
	}

	// Verify ticket type belongs to the event
	if ticketType.EventID != eventID {
		http.Error(w, "Ticket type does not belong to this event", http.StatusBadRequest)
		return
	}

	// Render the edit ticket type page
	component := pages.EditTicketTypePage(user, event, ticketType, nil, nil)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// UpdateTicketTypeSubmit handles ticket type update form submission
func (h *TicketTypeHandler) UpdateTicketTypeSubmit(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID and ticket type ID from URL
	eventIDStr := chi.URLParam(r, "eventId")
	ticketTypeIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}
	ticketTypeID, err := strconv.Atoi(ticketTypeIDStr)
	if err != nil {
		http.Error(w, "Invalid ticket type ID", http.StatusBadRequest)
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
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Extract form fields
	name := r.FormValue("name")
	description := r.FormValue("description")
	priceStr := r.FormValue("price")
	quantityStr := r.FormValue("quantity")
	saleStartStr := r.FormValue("sale_start")
	saleEndStr := r.FormValue("sale_end")

	// Validate required fields
	errors := make(map[string]string)
	if name == "" {
		errors["name"] = "Ticket type name is required"
	}
	if priceStr == "" {
		errors["price"] = "Price is required"
	}
	if quantityStr == "" {
		errors["quantity"] = "Quantity is required"
	}
	if saleStartStr == "" {
		errors["sale_start"] = "Sale start date is required"
	}
	if saleEndStr == "" {
		errors["sale_end"] = "Sale end date is required"
	}

	// Parse numeric fields
	var price, quantity int
	if priceStr != "" {
		// Convert price from dollars to cents
		if priceFloat, err := strconv.ParseFloat(priceStr, 64); err != nil {
			errors["price"] = "Invalid price format"
		} else {
			price = int(priceFloat * 100) // Convert to cents
		}
	}
	if quantityStr != "" {
		if quantity, err = strconv.Atoi(quantityStr); err != nil {
			errors["quantity"] = "Invalid quantity format"
		}
	}

	// Parse dates
	var saleStart, saleEnd time.Time
	if saleStartStr != "" {
		saleStart, err = time.Parse("2006-01-02T15:04", saleStartStr)
		if err != nil {
			errors["sale_start"] = "Invalid sale start date format"
		}
	}
	if saleEndStr != "" {
		saleEnd, err = time.Parse("2006-01-02T15:04", saleEndStr)
		if err != nil {
			errors["sale_end"] = "Invalid sale end date format"
		}
	}

	// Validate date logic
	if !saleStart.IsZero() && !saleEnd.IsZero() {
		if saleEnd.Before(saleStart) {
			errors["sale_end"] = "Sale end date must be after sale start date"
		}
	}

	// If there are validation errors, re-render the form
	if len(errors) > 0 {
		event, _ := h.eventService.GetEventByID(eventID)
		ticketType, _ := h.ticketService.GetTicketTypeByID(ticketTypeID)
		formData := map[string]interface{}{
			"name":        name,
			"description": description,
			"price":       priceStr,
			"quantity":    quantityStr,
			"sale_start":  saleStartStr,
			"sale_end":    saleEndStr,
		}

		component := pages.EditTicketTypePage(user, event, ticketType, formData, errors)
		err = component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// Update the ticket type
	updateReq := &models.TicketTypeUpdateRequest{
		Name:        name,
		Description: description,
		Price:       price,
		Quantity:    quantity,
		SaleStart:   saleStart,
		SaleEnd:     saleEnd,
	}

	_, err = h.ticketService.UpdateTicketType(ticketTypeID, updateReq)
	if err != nil {
		// Handle service-level errors
		errors["general"] = fmt.Sprintf("Failed to update ticket type: %v", err)
		event, _ := h.eventService.GetEventByID(eventID)
		ticketType, _ := h.ticketService.GetTicketTypeByID(ticketTypeID)
		formData := map[string]interface{}{
			"name":        name,
			"description": description,
			"price":       priceStr,
			"quantity":    quantityStr,
			"sale_start":  saleStartStr,
			"sale_end":    saleEndStr,
		}

		component := pages.EditTicketTypePage(user, event, ticketType, formData, errors)
		err = component.Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// Redirect back to the ticket types list with success message
	http.Redirect(w, r, fmt.Sprintf("/organizer/events/%d/tickets?success=updated", eventID), http.StatusSeeOther)
}

// DeleteTicketType handles ticket type deletion
func (h *TicketTypeHandler) DeleteTicketType(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event ID and ticket type ID from URL
	eventIDStr := chi.URLParam(r, "eventId")
	ticketTypeIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}
	ticketTypeID, err := strconv.Atoi(ticketTypeIDStr)
	if err != nil {
		http.Error(w, "Invalid ticket type ID", http.StatusBadRequest)
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

	// Get ticket type details for validation
	ticketType, err := h.ticketService.GetTicketTypeByID(ticketTypeID)
	if err != nil {
		http.Error(w, "Ticket type not found", http.StatusNotFound)
		return
	}

	// Verify ticket type belongs to the event
	if ticketType.EventID != eventID {
		http.Error(w, "Ticket type does not belong to this event", http.StatusBadRequest)
		return
	}

	// Safety check: Don't allow deletion of ticket types with sold tickets
	if ticketType.Sold > 0 {
		http.Error(w, "Cannot delete ticket type with sold tickets", http.StatusBadRequest)
		return
	}

	// Delete the ticket type
	err = h.ticketService.DeleteTicketType(ticketTypeID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete ticket type: %v", err), http.StatusInternalServerError)
		return
	}

	// For HTMX requests, return empty response to remove the row
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Redirect to ticket types list
	http.Redirect(w, r, fmt.Sprintf("/organizer/events/%d/tickets", eventID), http.StatusSeeOther)
}