package models

import (
	"errors"
	"strings"
	"time"
)

// TicketStatus represents the status of a ticket
type TicketStatus string

const (
	TicketActive   TicketStatus = "active"
	TicketUsed     TicketStatus = "used"
	TicketRefunded TicketStatus = "refunded"
)

// TicketType represents a type of ticket for an event
type TicketType struct {
	ID          int       `json:"id" db:"id"`
	EventID     int       `json:"event_id" db:"event_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Price       int       `json:"price" db:"price"` // Price in cents
	Quantity    int       `json:"quantity" db:"quantity"`
	Sold        int       `json:"sold" db:"sold"`
	SaleStart   time.Time `json:"sale_start" db:"sale_start"`
	SaleEnd     time.Time `json:"sale_end" db:"sale_end"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Ticket represents an individual ticket
type Ticket struct {
	ID           int          `json:"id" db:"id"`
	OrderID      int          `json:"order_id" db:"order_id"`
	TicketTypeID int          `json:"ticket_type_id" db:"ticket_type_id"`
	QRCode       string       `json:"qr_code" db:"qr_code"`
	Status       TicketStatus `json:"status" db:"status"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
}

// Validate validates the ticket type data
func (tt *TicketType) Validate() error {
	if err := tt.validateName(); err != nil {
		return err
	}

	if err := tt.validatePrice(); err != nil {
		return err
	}

	if err := tt.validateQuantity(); err != nil {
		return err
	}

	if err := tt.validateSalePeriod(); err != nil {
		return err
	}

	if err := tt.validateDescription(); err != nil {
		return err
	}

	return nil
}

// ValidateCreate validates ticket type creation data
func (req *TicketTypeCreateRequest) Validate() error {
	if err := validateTicketTypeName(req.Name); err != nil {
		return err
	}

	if err := validateTicketTypePrice(req.Price); err != nil {
		return err
	}

	if err := validateTicketTypeQuantity(req.Quantity); err != nil {
		return err
	}

	if err := validateTicketTypeSalePeriod(req.SaleStart, req.SaleEnd); err != nil {
		return err
	}

	if err := validateTicketTypeDescription(req.Description); err != nil {
		return err
	}

	return nil
}

// ValidateUpdate validates ticket type update data
func (req *TicketTypeUpdateRequest) Validate() error {
	if err := validateTicketTypeName(req.Name); err != nil {
		return err
	}

	if err := validateTicketTypePrice(req.Price); err != nil {
		return err
	}

	if err := validateTicketTypeQuantity(req.Quantity); err != nil {
		return err
	}

	if err := validateTicketTypeSalePeriod(req.SaleStart, req.SaleEnd); err != nil {
		return err
	}

	if err := validateTicketTypeDescription(req.Description); err != nil {
		return err
	}

	return nil
}

// Validate validates the ticket data
func (t *Ticket) Validate() error {
	if err := t.validateQRCode(); err != nil {
		return err
	}

	if err := t.validateStatus(); err != nil {
		return err
	}

	return nil
}

// validateName validates the ticket type name
func (tt *TicketType) validateName() error {
	return validateTicketTypeName(tt.Name)
}

// validatePrice validates the ticket type price
func (tt *TicketType) validatePrice() error {
	return validateTicketTypePrice(tt.Price)
}

// validateQuantity validates the ticket type quantity
func (tt *TicketType) validateQuantity() error {
	return validateTicketTypeQuantity(tt.Quantity)
}

// validateSalePeriod validates the ticket type sale period
func (tt *TicketType) validateSalePeriod() error {
	return validateTicketTypeSalePeriod(tt.SaleStart, tt.SaleEnd)
}

// validateDescription validates the ticket type description
func (tt *TicketType) validateDescription() error {
	return validateTicketTypeDescription(tt.Description)
}

// validateQRCode validates the ticket QR code
func (t *Ticket) validateQRCode() error {
	if t.QRCode == "" {
		return errors.New("QR code is required")
	}

	if len(t.QRCode) > 255 {
		return errors.New("QR code must be less than 255 characters")
	}

	return nil
}

// validateStatus validates the ticket status
func (t *Ticket) validateStatus() error {
	switch t.Status {
	case TicketActive, TicketUsed, TicketRefunded:
		return nil
	default:
		return errors.New("invalid ticket status")
	}
}

// validateTicketTypeName validates a ticket type name
func validateTicketTypeName(name string) error {
	if name == "" {
		return errors.New("ticket type name is required")
	}

	if len(name) > 100 {
		return errors.New("ticket type name must be less than 100 characters")
	}

	if strings.TrimSpace(name) == "" {
		return errors.New("ticket type name cannot be only whitespace")
	}

	return nil
}

// validateTicketTypePrice validates a ticket type price
func validateTicketTypePrice(price int) error {
	if price < 0 {
		return errors.New("ticket price cannot be negative")
	}

	// Maximum price of $10,000 (1,000,000 cents)
	if price > 1000000 {
		return errors.New("ticket price cannot exceed $10,000")
	}

	return nil
}

// validateTicketTypeQuantity validates a ticket type quantity
func validateTicketTypeQuantity(quantity int) error {
	if quantity <= 0 {
		return errors.New("ticket quantity must be greater than 0")
	}

	// Maximum quantity of 100,000 tickets per type
	if quantity > 100000 {
		return errors.New("ticket quantity cannot exceed 100,000")
	}

	return nil
}

// validateTicketTypeSalePeriod validates a ticket type sale period
func validateTicketTypeSalePeriod(saleStart, saleEnd time.Time) error {
	if saleStart.IsZero() {
		return errors.New("sale start date is required")
	}

	if saleEnd.IsZero() {
		return errors.New("sale end date is required")
	}

	if saleStart.After(saleEnd) {
		return errors.New("sale start date must be before sale end date")
	}

	// Sale period should be at least 1 hour
	if saleEnd.Sub(saleStart) < time.Hour {
		return errors.New("sale period must be at least 1 hour")
	}

	return nil
}

// validateTicketTypeDescription validates a ticket type description
func validateTicketTypeDescription(description string) error {
	// Description is optional, but if provided, it should not be too long
	if len(description) > 1000 {
		return errors.New("ticket type description must be less than 1000 characters")
	}

	return nil
}

// IsAvailable returns true if tickets are available for purchase
func (tt *TicketType) IsAvailable() bool {
	now := time.Now()
	return tt.Sold < tt.Quantity &&
		now.After(tt.SaleStart) &&
		now.Before(tt.SaleEnd)
}

// IsSoldOut returns true if all tickets are sold
func (tt *TicketType) IsSoldOut() bool {
	return tt.Sold >= tt.Quantity
}

// Available returns the number of available tickets
func (tt *TicketType) Available() int {
	available := tt.Quantity - tt.Sold
	if available < 0 {
		return 0
	}
	return available
}

// IsOnSale returns true if the ticket type is currently on sale
func (tt *TicketType) IsOnSale() bool {
	now := time.Now()
	return now.After(tt.SaleStart) && now.Before(tt.SaleEnd)
}

// SaleNotStarted returns true if the sale hasn't started yet
func (tt *TicketType) SaleNotStarted() bool {
	return time.Now().Before(tt.SaleStart)
}

// SaleEnded returns true if the sale has ended
func (tt *TicketType) SaleEnded() bool {
	return time.Now().After(tt.SaleEnd)
}

// PriceInCurrency returns the price in the main currency as a float
func (tt *TicketType) PriceInCurrency() float64 {
	return float64(tt.Price) / 100.0
}

// PriceInDollars returns the price in dollars as a float (legacy method)
func (tt *TicketType) PriceInDollars() float64 {
	return tt.PriceInCurrency()
}

// CanUpdateQuantity returns true if the quantity can be updated
func (tt *TicketType) CanUpdateQuantity(newQuantity int) bool {
	// Can only increase quantity or decrease to a value >= sold tickets
	return newQuantity >= tt.Sold
}

// IsActive returns true if the ticket is active
func (t *Ticket) IsActive() bool {
	return t.Status == TicketActive
}

// IsUsed returns true if the ticket has been used
func (t *Ticket) IsUsed() bool {
	return t.Status == TicketUsed
}

// IsRefunded returns true if the ticket has been refunded
func (t *Ticket) IsRefunded() bool {
	return t.Status == TicketRefunded
}

// CanBeUsed returns true if the ticket can be used (scanned)
func (t *Ticket) CanBeUsed() bool {
	return t.Status == TicketActive
}

// CanBeRefunded returns true if the ticket can be refunded
func (t *Ticket) CanBeRefunded() bool {
	return t.Status == TicketActive
}
