package models

import (
	"encoding/json"
	"time"
)

// AuditLog represents an administrative action log entry
type AuditLog struct {
	ID           int             `json:"id" db:"id"`
	AdminUserID  int             `json:"admin_user_id" db:"admin_user_id"`
	Action       string          `json:"action" db:"action"`
	TargetType   string          `json:"target_type" db:"target_type"`
	TargetID     int             `json:"target_id" db:"target_id"`
	Details      json.RawMessage `json:"details" db:"details"`
	IPAddress    string          `json:"ip_address" db:"ip_address"`
	UserAgent    string          `json:"user_agent" db:"user_agent"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	
	// Related data
	AdminUser *User `json:"admin_user,omitempty"`
}

// AuditLogCreateRequest represents a request to create an audit log entry
type AuditLogCreateRequest struct {
	AdminUserID int             `json:"admin_user_id"`
	Action      string          `json:"action"`
	TargetType  string          `json:"target_type"`
	TargetID    int             `json:"target_id"`
	Details     json.RawMessage `json:"details"`
	IPAddress   string          `json:"ip_address"`
	UserAgent   string          `json:"user_agent"`
}

// Common audit actions
const (
	AuditActionEventApprove    = "event_approve"
	AuditActionEventReject     = "event_reject"
	AuditActionEventDelete     = "event_delete"
	AuditActionUserSuspend     = "user_suspend"
	AuditActionUserActivate    = "user_activate"
	AuditActionUserRoleChange  = "user_role_change"
	AuditActionCategoryCreate  = "category_create"
	AuditActionCategoryUpdate  = "category_update"
	AuditActionCategoryDelete  = "category_delete"
	AuditActionWithdrawalApprove = "withdrawal_approve"
	AuditActionWithdrawalReject  = "withdrawal_reject"
	AuditActionWithdrawalComplete = "withdrawal_complete"
)

// Common target types
const (
	AuditTargetEvent      = "event"
	AuditTargetUser       = "user"
	AuditTargetCategory   = "category"
	AuditTargetWithdrawal = "withdrawal"
)