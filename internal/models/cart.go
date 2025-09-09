package models

// Cart represents a shopping cart
type Cart struct {
	EventID     int        `json:"event_id"`
	EventTitle  string     `json:"event_title"`
	Items       []CartItem `json:"items"`
	TotalAmount int        `json:"total_amount"` // in cents
	ExpiresAt   int64      `json:"expires_at"`   // Unix timestamp
}

// CartItem represents an item in the shopping cart
type CartItem struct {
	TicketTypeID int    `json:"ticket_type_id"`
	TicketName   string `json:"ticket_name"`
	Price        int    `json:"price"`    // in cents
	Quantity     int    `json:"quantity"`
	Subtotal     int    `json:"subtotal"` // in cents
}