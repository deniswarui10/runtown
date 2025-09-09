package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/database"
	"event-ticketing-platform/internal/models"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("Starting Authboss user data migration...")

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

	// Run the migration
	migrator := NewAuthbossMigrator(db.DB)
	
	// First, verify the migration can be performed
	if err := migrator.VerifyMigration(); err != nil {
		log.Fatal("Migration verification failed:", err)
	}

	// Perform the migration
	if err := migrator.MigrateUsers(); err != nil {
		log.Fatal("Migration failed:", err)
	}

	// Verify the migration was successful
	if err := migrator.VerifyMigrationSuccess(); err != nil {
		log.Fatal("Migration verification failed:", err)
	}

	log.Println("Authboss user data migration completed successfully!")
}

// AuthbossMigrator handles the migration of user data to Authboss format
type AuthbossMigrator struct {
	db *sql.DB
}

// NewAuthbossMigrator creates a new Authboss migrator
func NewAuthbossMigrator(db *sql.DB) *AuthbossMigrator {
	return &AuthbossMigrator{db: db}
}

// VerifyMigration checks if the migration can be performed safely
func (m *AuthbossMigrator) VerifyMigration() error {
	log.Println("Verifying migration prerequisites...")

	// Check if Authboss columns exist
	authbossColumns := []string{
		"confirmed_at", "confirm_selector", "confirm_verifier",
		"locked_until", "attempt_count", "last_attempt",
		"password_changed_at", "recover_selector", "recover_verifier", "recover_token_expires",
	}

	for _, column := range authbossColumns {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'users' AND column_name = $1
			)`
		
		err := m.db.QueryRow(query, column).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check column %s: %w", column, err)
		}
		
		if !exists {
			return fmt.Errorf("required Authboss column %s does not exist. Please run database migrations first", column)
		}
	}

	// Check if there are users to migrate
	var userCount int
	err := m.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	log.Printf("Found %d users to migrate", userCount)

	// Check if any users already have Authboss data
	var authbossUserCount int
	err = m.db.QueryRow("SELECT COUNT(*) FROM users WHERE confirmed_at IS NOT NULL OR attempt_count > 0").Scan(&authbossUserCount)
	if err != nil {
		return fmt.Errorf("failed to count Authboss users: %w", err)
	}

	if authbossUserCount > 0 {
		log.Printf("Warning: %d users already have Authboss data. Migration will update existing data.", authbossUserCount)
	}

	return nil
}

// MigrateUsers migrates existing users to Authboss format
func (m *AuthbossMigrator) MigrateUsers() error {
	log.Println("Starting user migration...")

	// Begin transaction
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get all users
	rows, err := tx.Query(`
		SELECT id, email, password_hash, first_name, last_name, role, 
		       email_verified_at, created_at, updated_at
		FROM users
		ORDER BY id
	`)
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var migratedCount int
	var skippedCount int

	for rows.Next() {
		var user models.User
		var emailVerifiedAt *time.Time

		err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName, &user.Role,
			&emailVerifiedAt, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to scan user: %w", err)
		}

		// Migrate this user
		err = m.migrateUser(tx, &user, emailVerifiedAt)
		if err != nil {
			log.Printf("Failed to migrate user %s (ID: %d): %v", user.Email, user.ID, err)
			skippedCount++
			continue
		}

		migratedCount++
		if migratedCount%100 == 0 {
			log.Printf("Migrated %d users...", migratedCount)
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating users: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Migration completed: %d users migrated, %d skipped", migratedCount, skippedCount)
	return nil
}

// migrateUser migrates a single user to Authboss format
func (m *AuthbossMigrator) migrateUser(tx *sql.Tx, user *models.User, emailVerifiedAt *time.Time) error {
	now := time.Now()

	// Set default Authboss values
	var confirmedAt *time.Time
	if emailVerifiedAt != nil {
		// User is already verified
		confirmedAt = emailVerifiedAt
	} else {
		// User is not verified - they'll need to verify their email
		confirmedAt = nil
	}

	// Update user with Authboss fields
	query := `
		UPDATE users SET
			confirmed_at = $2,
			confirm_selector = $3,
			confirm_verifier = $4,
			locked_until = $5,
			attempt_count = $6,
			last_attempt = $7,
			password_changed_at = $8,
			recover_selector = $9,
			recover_verifier = $10,
			recover_token_expires = $11,
			updated_at = $12
		WHERE id = $1
	`

	_, err := tx.Exec(query,
		user.ID,                // $1
		confirmedAt,            // $2 - confirmed_at
		nil,                    // $3 - confirm_selector (empty for migrated users)
		nil,                    // $4 - confirm_verifier (empty for migrated users)
		nil,                    // $5 - locked_until (not locked)
		0,                      // $6 - attempt_count (reset to 0)
		nil,                    // $7 - last_attempt (no previous attempts)
		&user.CreatedAt,        // $8 - password_changed_at (use creation date)
		nil,                    // $9 - recover_selector (empty)
		nil,                    // $10 - recover_verifier (empty)
		nil,                    // $11 - recover_token_expires (empty)
		now,                    // $12 - updated_at
	)

	if err != nil {
		return fmt.Errorf("failed to update user %d: %w", user.ID, err)
	}

	return nil
}

// VerifyMigrationSuccess verifies that the migration was successful
func (m *AuthbossMigrator) VerifyMigrationSuccess() error {
	log.Println("Verifying migration success...")

	// Count total users
	var totalUsers int
	err := m.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	if err != nil {
		return fmt.Errorf("failed to count total users: %w", err)
	}

	// Count users with Authboss data
	var authbossUsers int
	err = m.db.QueryRow(`
		SELECT COUNT(*) FROM users 
		WHERE attempt_count IS NOT NULL 
		AND password_changed_at IS NOT NULL
	`).Scan(&authbossUsers)
	if err != nil {
		return fmt.Errorf("failed to count Authboss users: %w", err)
	}

	// Count confirmed users
	var confirmedUsers int
	err = m.db.QueryRow("SELECT COUNT(*) FROM users WHERE confirmed_at IS NOT NULL").Scan(&confirmedUsers)
	if err != nil {
		return fmt.Errorf("failed to count confirmed users: %w", err)
	}

	// Count unconfirmed users
	var unconfirmedUsers int
	err = m.db.QueryRow("SELECT COUNT(*) FROM users WHERE confirmed_at IS NULL").Scan(&unconfirmedUsers)
	if err != nil {
		return fmt.Errorf("failed to count unconfirmed users: %w", err)
	}

	log.Printf("Migration verification results:")
	log.Printf("  Total users: %d", totalUsers)
	log.Printf("  Users with Authboss data: %d", authbossUsers)
	log.Printf("  Confirmed users: %d", confirmedUsers)
	log.Printf("  Unconfirmed users: %d", unconfirmedUsers)

	if authbossUsers != totalUsers {
		return fmt.Errorf("migration incomplete: expected %d users with Authboss data, got %d", totalUsers, authbossUsers)
	}

	log.Println("Migration verification successful!")
	return nil
}