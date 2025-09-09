package models

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderPending   OrderStatus = "pending"
	OrderCompleted OrderStatus = "completed"
	OrderCancelled OrderStatus = "cancelled"
	OrderRefunded  OrderStatus = "refunded"
)

// Order represents an order in the system
type Order struct {
	ID           int         `json:"id" db:"id"`
	UserID       int         `json:"user_id" db:"user_id"`
	EventID      int         `json:"event_id" db:"event_id"`
	OrderNumber  string      `json:"order_number" db:"order_number"`
	TotalAmount  int         `json:"total_amount" db:"total_amount"` // Amount in cents
	Status       OrderStatus `json:"status" db:"status"`
	PaymentID    string      `json:"payment_id" db:"payment_id"`
	BillingEmail string      `json:"billing_email" db:"billing_email"`
	BillingName  string      `json:"billing_name" db:"billing_name"`
	CreatedAt    time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at" db:"updated_at"`
}

// OrderCreateRequest represents the data needed to create a new order
type OrderCreateRequest struct {
	UserID       int         `json:"user_id"`
	EventID      int         `json:"event_id"`
	TotalAmount  int         `json:"total_amount"`
	BillingEmail string      `json:"billing_email"`
	BillingName  string      `json:"billing_name"`
	Status       OrderStatus `json:"status"`
}

// OrderUpdateRequest represents the data that can be updated for an order
type OrderUpdateRequest struct {
	Status    OrderStatus `json:"status"`
	PaymentID string      `json:"payment_id"`
}

var (
	// Order number format: ORD-YYYYMMDD-XXXXXX (e.g., ORD-20240101-123456)
	orderNumberRegex = regexp.MustCompile(`^ORD-\d{8}-\d{6}$`)
	// Email validation regex for orders
	orderEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// Validate validates the order data
func (o *Order) Validate() error {
	if err := o.validateOrderNumber(); err != nil {
		return err
	}

	if err := o.validateTotalAmount(); err != nil {
		return err
	}

	if err := o.validateStatus(); err != nil {
		return err
	}

	if err := o.validateBillingInfo(); err != nil {
		return err
	}

	return nil
}

// ValidateCreate validates order creation data
func (req *OrderCreateRequest) Validate() error {
	if err := validateOrderTotalAmount(req.TotalAmount); err != nil {
		return err
	}

	if err := validateOrderStatus(req.Status); err != nil {
		return err
	}

	if err := validateOrderBillingInfo(req.BillingEmail, req.BillingName); err != nil {
		return err
	}

	return nil
}

// ValidateUpdate validates order update data
func (req *OrderUpdateRequest) Validate() error {
	if err := validateOrderStatus(req.Status); err != nil {
		return err
	}

	return nil
}

// validateOrderNumber validates the order number
func (o *Order) validateOrderNumber() error {
	if o.OrderNumber == "" {
		return errors.New("order number is required")
	}

	if !orderNumberRegex.MatchString(o.OrderNumber) {
		return errors.New("order number format is invalid")
	}

	return nil
}

// validateTotalAmount validates the order total amount
func (o *Order) validateTotalAmount() error {
	return validateOrderTotalAmount(o.TotalAmount)
}

// validateStatus validates the order status
func (o *Order) validateStatus() error {
	return validateOrderStatus(o.Status)
}

// validateBillingInfo validates the order billing information
func (o *Order) validateBillingInfo() error {
	return validateOrderBillingInfo(o.BillingEmail, o.BillingName)
}

// validateOrderTotalAmount validates an order total amount
func validateOrderTotalAmount(totalAmount int) error {
	if totalAmount < 0 {
		return errors.New("total amount cannot be negative")
	}

	// Maximum order amount of $100,000 (10,000,000 cents)
	if totalAmount > 10000000 {
		return errors.New("total amount cannot exceed $100,000")
	}

	return nil
}

// validateOrderStatus validates an order status
func validateOrderStatus(status OrderStatus) error {
	switch status {
	case OrderPending, OrderCompleted, OrderCancelled, OrderRefunded:
		return nil
	default:
		return errors.New("invalid order status")
	}
}

// validateOrderBillingInfo validates order billing information
func validateOrderBillingInfo(billingEmail, billingName string) error {
	if billingEmail == "" {
		return errors.New("billing email is required")
	}

	if billingName == "" {
		return errors.New("billing name is required")
	}

	if len(billingEmail) > 255 {
		return errors.New("billing email must be less than 255 characters")
	}

	if len(billingName) > 255 {
		return errors.New("billing name must be less than 255 characters")
	}

	// Validate email format
	if !orderEmailRegex.MatchString(billingEmail) {
		return errors.New("billing email format is invalid")
	}

	if strings.TrimSpace(billingName) == "" {
		return errors.New("billing name cannot be only whitespace")
	}

	return nil
}

// GenerateOrderNumber generates a unique order number
func GenerateOrderNumber() string {
	now := time.Now()
	dateStr := now.Format("20060102")

	// Generate a 6-digit random number using crypto/rand for better uniqueness
	max := big.NewInt(1000000)
	randomNum, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback to timestamp-based generation if crypto/rand fails
		timestamp := now.UnixNano()
		randomPart := timestamp % 1000000
		return fmt.Sprintf("ORD-%s-%06d", dateStr, randomPart)
	}

	return fmt.Sprintf("ORD-%s-%06d", dateStr, randomNum.Int64())
}

// IsPending returns true if the order is pending
func (o *Order) IsPending() bool {
	return o.Status == OrderPending
}

// IsCompleted returns true if the order is completed
func (o *Order) IsCompleted() bool {
	return o.Status == OrderCompleted
}

// IsCancelled returns true if the order is cancelled
func (o *Order) IsCancelled() bool {
	return o.Status == OrderCancelled
}

// IsRefunded returns true if the order is refunded
func (o *Order) IsRefunded() bool {
	return o.Status == OrderRefunded
}

// CanBeCancelled returns true if the order can be cancelled
func (o *Order) CanBeCancelled() bool {
	return o.Status == OrderPending
}

// CanBeRefunded returns true if the order can be refunded
func (o *Order) CanBeRefunded() bool {
	return o.Status == OrderCompleted
}

// TotalAmountInCurrency returns the total amount in the main currency as a float
func (o *Order) TotalAmountInCurrency() float64 {
	return float64(o.TotalAmount) / 100.0
}

// TotalAmountInDollars returns the total amount in dollars as a float (legacy method)
func (o *Order) TotalAmountInDollars() float64 {
	return o.TotalAmountInCurrency()
}

// IsExpired returns true if the order has expired (for pending orders)
func (o *Order) IsExpired(expirationDuration time.Duration) bool {
	if o.Status != OrderPending {
		return false
	}

	return time.Since(o.CreatedAt) > expirationDuration
}

// CanBeCompleted returns true if the order can be marked as completed
func (o *Order) CanBeCompleted() bool {
	return o.Status == OrderPending
}

// GetStatusDisplayName returns a human-readable status name
func (o *Order) GetStatusDisplayName() string {
	switch o.Status {
	case OrderPending:
		return "Pending Payment"
	case OrderCompleted:
		return "Completed"
	case OrderCancelled:
		return "Cancelled"
	case OrderRefunded:
		return "Refunded"
	default:
		return string(o.Status)
	}
}
