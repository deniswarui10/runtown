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
	log.Println("Verifying Authboss migration status...")

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

	// Run verification
	verifier := NewAuthbossVerifier(db.DB)
	if err := verifier.VerifyMigrationStatus(); err != nil {
		log.Fatal("Verification failed:", err)
	}

	log.Println("Authboss migration verification completed!")
}

// AuthbossVerifier handles verification of Authboss migration
type AuthbossVerifier struct {
	db *sql.DB
}

// NewAuthbossVerifier creates a new Authboss verifier
func NewAuthbossVerifier(db *sql.DB) *AuthbossVerifier {
	return &AuthbossVerifier{db: db}
}

// VerifyMigrationStatus checks the current migration status
func (v *AuthbossVerifier) VerifyMigrationStatus() error {
	log.Println("Checking migration status...")

	// Check if Authboss columns exist
	if err := v.checkAuthbossColumns(); err != nil {
		return err
	}

	// Check user data
	if err := v.checkUserData(); err != nil {
		return err
	}

	// Check remember tokens table
	if err := v.checkRememberTokensTable(); err != nil {
		return err
	}

	return nil
}

// checkAuthbossColumns verifies that all required Authboss columns exist
func (v *AuthbossVerifier) checkAuthbossColumns() error {
	log.Println("Checking Authboss columns...")

	requiredColumns := []string{
		"confirmed_at",
		"confirm_selector",
		"confirm_verifier",
		"locked_until",
		"attempt_count",
		"last_attempt",
		"password_changed_at",
		"recover_selector",
		"recover_verifier",
		"recover_token_expires",
	}

	for _, column := range requiredColumns {
		var exists bool
		var dataType string
		
		query := `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'users' AND column_name = $1
			), COALESCE(
				(SELECT data_type FROM information_schema.columns 
				 WHERE table_name = 'users' AND column_name = $1), 
				'missing'
			)`
		
		err := v.db.QueryRow(query, column).Scan(&exists, &dataType)
		if err != nil {
			return fmt.Errorf("failed to check column %s: %w", column, err)
		}
		
		if !exists {
			log.Printf("❌ Column %s is missing", column)
		} else {
			log.Printf("✅ Column %s exists (type: %s)", column, dataType)
		}
	}

	return nil
}

// checkUserData analyzes the current state of user data
func (v *AuthbossVerifier) checkUserData() error {
	log.Println("Analyzing user data...")

	// Total users
	var totalUsers int
	err := v.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}
	log.Printf("📊 Total users: %d", totalUsers)

	if totalUsers == 0 {
		log.Println("ℹ️  No users found in database")
		return nil
	}

	// Users with Authboss data
	var authbossUsers int
	err = v.db.QueryRow(`
		SELECT COUNT(*) FROM users 
		WHERE attempt_count IS NOT NULL 
		AND password_changed_at IS NOT NULL
	`).Scan(&authbossUsers)
	if err != nil {
		return fmt.Errorf("failed to count Authboss users: %w", err)
	}
	log.Printf("📊 Users with Authboss data: %d", authbossUsers)

	// Confirmed users
	var confirmedUsers int
	err = v.db.QueryRow("SELECT COUNT(*) FROM users WHERE confirmed_at IS NOT NULL").Scan(&confirmedUsers)
	if err != nil {
		return fmt.Errorf("failed to count confirmed users: %w", err)
	}
	log.Printf("📊 Confirmed users: %d", confirmedUsers)

	// Locked users
	var lockedUsers int
	err = v.db.QueryRow("SELECT COUNT(*) FROM users WHERE locked_until IS NOT NULL AND locked_until > NOW()").Scan(&lockedUsers)
	if err != nil {
		return fmt.Errorf("failed to count locked users: %w", err)
	}
	log.Printf("📊 Currently locked users: %d", lockedUsers)

	// Users with failed attempts
	var usersWithAttempts int
	err = v.db.QueryRow("SELECT COUNT(*) FROM users WHERE attempt_count > 0").Scan(&usersWithAttempts)
	if err != nil {
		return fmt.Errorf("failed to count users with attempts: %w", err)
	}
	log.Printf("📊 Users with failed attempts: %d", usersWithAttempts)

	// Migration status
	migrationPercentage := float64(authbossUsers) / float64(totalUsers) * 100
	log.Printf("📊 Migration completion: %.1f%%", migrationPercentage)

	if authbossUsers == totalUsers {
		log.Println("✅ All users have been migrated to Authboss format")
	} else if authbossUsers == 0 {
		log.Println("❌ No users have been migrated to Authboss format")
	} else {
		log.Printf("⚠️  Partial migration: %d/%d users migrated", authbossUsers, totalUsers)
	}

	return nil
}

// checkRememberTokensTable verifies the remember tokens table
func (v *AuthbossVerifier) checkRememberTokensTable() error {
	log.Println("Checking remember tokens table...")

	// Check if table exists
	var exists bool
	err := v.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'authboss_remember_tokens'
		)
	`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check remember tokens table: %w", err)
	}

	if !exists {
		log.Println("❌ Remember tokens table does not exist")
		return nil
	}

	log.Println("✅ Remember tokens table exists")

	// Count tokens
	var tokenCount int
	err = v.db.QueryRow("SELECT COUNT(*) FROM authboss_remember_tokens").Scan(&tokenCount)
	if err != nil {
		return fmt.Errorf("failed to count remember tokens: %w", err)
	}
	log.Printf("📊 Remember tokens: %d", tokenCount)

	return nil
}