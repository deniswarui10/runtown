package services

import (
	"encoding/json"
	"net/http"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
)

// AuditService handles audit logging operations
type AuditService struct {
	auditRepo *repositories.AuditLogRepository
}

// NewAuditService creates a new audit service
func NewAuditService(auditRepo *repositories.AuditLogRepository) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
	}
}

// LogAction logs an administrative action
func (s *AuditService) LogAction(adminUserID int, action, targetType string, targetID int, details interface{}, r *http.Request) error {
	// Convert details to JSON
	var detailsJSON json.RawMessage
	if details != nil {
		detailsBytes, err := json.Marshal(details)
		if err != nil {
			return err
		}
		detailsJSON = detailsBytes
	}

	// Get IP address and user agent from request
	ipAddress := getClientIP(r)
	userAgent := r.UserAgent()

	req := &models.AuditLogCreateRequest{
		AdminUserID: adminUserID,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Details:     detailsJSON,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	}

	_, err := s.auditRepo.Create(req)
	return err
}

// GetAuditLogs retrieves audit logs with pagination and filtering
func (s *AuditService) GetAuditLogs(page, limit int, action, targetType string) ([]*models.AuditLog, int, error) {
	offset := (page - 1) * limit
	return s.auditRepo.GetAll(limit, offset, action, targetType)
}

// GetAuditLogsByAdmin retrieves audit logs for a specific admin user
func (s *AuditService) GetAuditLogsByAdmin(adminUserID int, page, limit int) ([]*models.AuditLog, int, error) {
	offset := (page - 1) * limit
	return s.auditRepo.GetByAdminUser(adminUserID, limit, offset)
}

// GetAuditLogsByTarget retrieves audit logs for a specific target
func (s *AuditService) GetAuditLogsByTarget(targetType string, targetID int, page, limit int) ([]*models.AuditLog, int, error) {
	offset := (page - 1) * limit
	return s.auditRepo.GetByTarget(targetType, targetID, limit, offset)
}

// Helper function to get client IP address
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := len(forwarded); idx > 0 {
			if commaIdx := 0; commaIdx < idx {
				for i, char := range forwarded {
					if char == ',' {
						commaIdx = i
						break
					}
				}
				if commaIdx > 0 {
					return forwarded[:commaIdx]
				}
			}
			return forwarded
		}
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}