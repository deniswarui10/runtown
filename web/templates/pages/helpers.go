package pages

import (
	"context"
	"fmt"
	"strconv"
	"event-ticketing-platform/internal/models"
)

func getFormValue(formData map[string]string, key, defaultValue string) string {
	if value, exists := formData[key]; exists {
		return value
	}
	return defaultValue
}

// Helper functions for organizer event templates

// getStringValue safely gets a string value from form data
func getStringValue(formData map[string]interface{}, key string) string {
	if formData == nil {
		return ""
	}
	if value, ok := formData[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// getFormDataFromEvent converts an event to form data for editing
func getFormDataFromEvent(event *models.Event, formData map[string]interface{}) map[string]interface{} {
	// If formData is provided (from form submission), use it (for validation errors)
	if formData != nil && len(formData) > 0 {
		return formData
	}
	
	// Otherwise, populate from event data
	return map[string]interface{}{
		"title":       event.Title,
		"description": event.Description,
		"location":    event.Location,
		"start_date":  event.StartDate.Format("2006-01-02T15:04"),
		"end_date":    event.EndDate.Format("2006-01-02T15:04"),
		"category_id": strconv.Itoa(event.CategoryID),
		"status":      string(event.Status),
	}
}

// getFormDataFromTicketType converts a ticket type to form data for editing
func getFormDataFromTicketType(ticketType *models.TicketType, formData map[string]interface{}) map[string]interface{} {
	// If formData is provided (from form submission), use it (for validation errors)
	if formData != nil && len(formData) > 0 {
		return formData
	}
	
	// Otherwise, populate from ticket type data
	return map[string]interface{}{
		"name":        ticketType.Name,
		"description": ticketType.Description,
		"price":       fmt.Sprintf("%.2f", ticketType.PriceInDollars()),
		"quantity":    strconv.Itoa(ticketType.Quantity),
		"sale_start":  ticketType.SaleStart.Format("2006-01-02T15:04"),
		"sale_end":    ticketType.SaleEnd.Format("2006-01-02T15:04"),
	}
}

// getCSRFToken gets the CSRF token from the request context
func getCSRFToken(ctx context.Context) string {
	if token, ok := ctx.Value("csrf_token").(string); ok {
		return token
	}
	return ""
}