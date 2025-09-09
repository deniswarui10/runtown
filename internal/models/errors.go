package models

import "errors"

// Common errors used throughout the application
var (
	ErrEventNotFound   = errors.New("event not found")
	ErrUserNotFound    = errors.New("user not found")
	ErrOrderNotFound   = errors.New("order not found")
	ErrTicketNotFound  = errors.New("ticket not found")
	ErrNotImplemented  = errors.New("feature not implemented")
	ErrUnauthorized    = errors.New("unauthorized access")
	ErrInvalidInput    = errors.New("invalid input")
	ErrDuplicateEntry  = errors.New("duplicate entry")
	ErrInsufficientStock = errors.New("insufficient ticket stock")
)