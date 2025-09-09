package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"event-ticketing-platform/internal/models"
)

// AuditLogRepository handles audit log data operations
type AuditLogRepository struct {
	db *sql.DB
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *sql.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create creates a new audit log entry
func (r *AuditLogRepository) Create(req *models.AuditLogCreateRequest) (*models.AuditLog, error) {
	query := `
		INSERT INTO admin_audit_log (admin_user_id, action, target_type, target_id, details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, admin_user_id, action, target_type, target_id, details, ip_address, user_agent, created_at`

	auditLog := &models.AuditLog{}
	now := time.Now()

	err := r.db.QueryRow(
		query,
		req.AdminUserID,
		req.Action,
		req.TargetType,
		req.TargetID,
		req.Details,
		req.IPAddress,
		req.UserAgent,
		now,
	).Scan(
		&auditLog.ID,
		&auditLog.AdminUserID,
		&auditLog.Action,
		&auditLog.TargetType,
		&auditLog.TargetID,
		&auditLog.Details,
		&auditLog.IPAddress,
		&auditLog.UserAgent,
		&auditLog.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create audit log: %w", err)
	}

	return auditLog, nil
}

// GetByAdminUser retrieves audit logs for a specific admin user
func (r *AuditLogRepository) GetByAdminUser(adminUserID int, limit, offset int) ([]*models.AuditLog, int, error) {
	// Get total count
	countQuery := "SELECT COUNT(*) FROM admin_audit_log WHERE admin_user_id = $1"
	var totalCount int
	err := r.db.QueryRow(countQuery, adminUserID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get audit log count: %w", err)
	}

	// Get audit logs
	query := `
		SELECT al.id, al.admin_user_id, al.action, al.target_type, al.target_id, 
		       al.details, al.ip_address, al.user_agent, al.created_at,
		       u.first_name, u.last_name, u.email
		FROM admin_audit_log al
		JOIN users u ON al.admin_user_id = u.id
		WHERE al.admin_user_id = $1
		ORDER BY al.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(query, adminUserID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var auditLogs []*models.AuditLog
	for rows.Next() {
		auditLog := &models.AuditLog{
			AdminUser: &models.User{},
		}

		err := rows.Scan(
			&auditLog.ID,
			&auditLog.AdminUserID,
			&auditLog.Action,
			&auditLog.TargetType,
			&auditLog.TargetID,
			&auditLog.Details,
			&auditLog.IPAddress,
			&auditLog.UserAgent,
			&auditLog.CreatedAt,
			&auditLog.AdminUser.FirstName,
			&auditLog.AdminUser.LastName,
			&auditLog.AdminUser.Email,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}

		auditLogs = append(auditLogs, auditLog)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return auditLogs, totalCount, nil
}

// GetAll retrieves all audit logs with pagination
func (r *AuditLogRepository) GetAll(limit, offset int, action, targetType string) ([]*models.AuditLog, int, error) {
	// Build WHERE clause
	var whereConditions []string
	var args []interface{}
	argIndex := 1

	if action != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("al.action = $%d", argIndex))
		args = append(args, action)
		argIndex++
	}

	if targetType != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("al.target_type = $%d", argIndex))
		args = append(args, targetType)
		argIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + fmt.Sprintf("%s", whereConditions[0])
		for i := 1; i < len(whereConditions); i++ {
			whereClause += " AND " + whereConditions[i]
		}
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM admin_audit_log al %s", whereClause)
	var totalCount int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get audit log count: %w", err)
	}

	// Get audit logs
	query := fmt.Sprintf(`
		SELECT al.id, al.admin_user_id, al.action, al.target_type, al.target_id,
		       al.details, al.ip_address, al.user_agent, al.created_at,
		       u.first_name, u.last_name, u.email
		FROM admin_audit_log al
		JOIN users u ON al.admin_user_id = u.id
		%s
		ORDER BY al.created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var auditLogs []*models.AuditLog
	for rows.Next() {
		auditLog := &models.AuditLog{
			AdminUser: &models.User{},
		}

		err := rows.Scan(
			&auditLog.ID,
			&auditLog.AdminUserID,
			&auditLog.Action,
			&auditLog.TargetType,
			&auditLog.TargetID,
			&auditLog.Details,
			&auditLog.IPAddress,
			&auditLog.UserAgent,
			&auditLog.CreatedAt,
			&auditLog.AdminUser.FirstName,
			&auditLog.AdminUser.LastName,
			&auditLog.AdminUser.Email,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}

		auditLogs = append(auditLogs, auditLog)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return auditLogs, totalCount, nil
}

// GetByTarget retrieves audit logs for a specific target
func (r *AuditLogRepository) GetByTarget(targetType string, targetID int, limit, offset int) ([]*models.AuditLog, int, error) {
	// Get total count
	countQuery := "SELECT COUNT(*) FROM admin_audit_log WHERE target_type = $1 AND target_id = $2"
	var totalCount int
	err := r.db.QueryRow(countQuery, targetType, targetID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get audit log count: %w", err)
	}

	// Get audit logs
	query := `
		SELECT al.id, al.admin_user_id, al.action, al.target_type, al.target_id,
		       al.details, al.ip_address, al.user_agent, al.created_at,
		       u.first_name, u.last_name, u.email
		FROM admin_audit_log al
		JOIN users u ON al.admin_user_id = u.id
		WHERE al.target_type = $1 AND al.target_id = $2
		ORDER BY al.created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(query, targetType, targetID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var auditLogs []*models.AuditLog
	for rows.Next() {
		auditLog := &models.AuditLog{
			AdminUser: &models.User{},
		}

		err := rows.Scan(
			&auditLog.ID,
			&auditLog.AdminUserID,
			&auditLog.Action,
			&auditLog.TargetType,
			&auditLog.TargetID,
			&auditLog.Details,
			&auditLog.IPAddress,
			&auditLog.UserAgent,
			&auditLog.CreatedAt,
			&auditLog.AdminUser.FirstName,
			&auditLog.AdminUser.LastName,
			&auditLog.AdminUser.Email,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}

		auditLogs = append(auditLogs, auditLog)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return auditLogs, totalCount, nil
}