package main

import (
	"database/sql"
	"fmt"
	"log"

	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/database"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("Starting Authboss migration rollback...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize database connection
	dbConfig := database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	log.Println("Database connection established successfully")

	// Run the rollback
	rollback := NewAuthbossRollback(db.DB)
	
	// Verify rollback can be performed
	if err := rollback.VerifyRollback(); err != nil {
		log.Fatal("Rollback verification failed:", err)
	}

	// Perform the rollback
	if err := rollback.RollbackAuthbossData(); err != nil {
		log.Fatal("Rollback failed:", err)
	}

	log.Println("Authboss migration rollback completed successfully!")
}

// AuthbossRollback handles rolling back Authboss migration
type AuthbossRollback struct {
	db *sql.DB
}

// NewAuthbossRollback creates a new Authboss rollback handler
func NewAuthbossRollback(db *sql.DB) *AuthbossRollback {
	return &AuthbossRollback{db: db}
}

// VerifyRollback checks if rollback can be performed safely
func (r *AuthbossRollback) VerifyRollback() error {
	log.Println("Verifying rollback prerequisites...")

	// Count users with Authboss data
	var authbossUsers int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM users 
		WHERE confirmed_at IS NOT NULL 
		OR attempt_count > 0 
		OR locked_until IS NOT NULL
		OR password_changed_at IS NOT NULL
	`).Scan(&authbossUsers)
	if err != nil {
		return fmt.Errorf("failed to count Authboss users: %w", err)
	}

	if authbossUsers == 0 {
		return fmt.Errorf("no Authboss data found to rollback")
	}

	log.Printf("Found %d users with Authboss data to rollback", authbossUsers)
	return nil
}

// RollbackAuthbossData removes Authboss-specific data from users
func (r *AuthbossRollback) RollbackAuthbossData() error {
	log.Println("Starting Authboss data rollback...")

	// Begin transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear Authboss fields but preserve core user data
	query := `
		UPDATE users SET
			confirmed_at = NULL,
			confirm_selector = NULL,
			confirm_verifier = NULL,
			locked_until = NULL,
			attempt_count = 0,
			last_attempt = NULL,
			password_changed_at = NULL,
			recover_selector = NULL,
			recover_verifier = NULL,
			recover_token_expires = NULL
	`

	result, err := tx.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear Authboss data: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	// Clear remember tokens table
	_, err = tx.Exec("DELETE FROM authboss_remember_tokens")
	if err != nil {
		log.Printf("Warning: failed to clear remember tokens: %v", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Rollback completed: %d users processed", rowsAffected)
	return nil
}