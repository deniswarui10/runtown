package models

import "time"

// TicketTypeCreateRequest represents a request to create a new ticket type
type TicketTypeCreateRequest struct {
	EventID     int       `json:"event_id" validate:"required"`
	Name        string    `json:"name" validate:"required"`
	Description string    `json:"description"`
	Price       int       `json:"price" validate:"required,min=0"`
	Quantity    int       `json:"quantity" validate:"required,min=1"`
	SaleStart   time.Time `json:"sale_start" validate:"required"`
	SaleEnd     time.Time `json:"sale_end" validate:"required"`
}

// TicketTypeUpdateRequest represents a request to update a ticket type
type TicketTypeUpdateRequest struct {
	Name        string    `json:"name" validate:"required"`
	Description string    `json:"description"`
	Price       int       `json:"price" validate:"required,min=0"`
	Quantity    int       `json:"quantity" validate:"required,min=1"`
	SaleStart   time.Time `json:"sale_start" validate:"required"`
	SaleEnd     time.Time `json:"sale_end" validate:"required"`
}