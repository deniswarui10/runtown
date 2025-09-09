package database

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Migration struct {
	Version int
	Name    string
	SQL     string
}

type Migrator struct {
	db *sql.DB
}

func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

// CreateMigrationsTable creates the migrations tracking table
func (m *Migrator) CreateMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := m.db.Exec(query)
	return err
}

// GetAppliedMigrations returns a list of applied migration versions
func (m *Migrator) GetAppliedMigrations() (map[int]bool, error) {
	applied := make(map[int]bool)
	
	rows, err := m.db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return applied, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return applied, err
		}
		applied[version] = true
	}
	
	return applied, rows.Err()
}

// LoadMigrations loads all migration files from the embedded filesystem
func (m *Migrator) LoadMigrations() ([]Migration, error) {
	var migrations []Migration
	
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}
	
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		
		// Parse version from filename (e.g., "001_create_users_table.sql")
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) != 2 {
			continue
		}
		
		var version int
		if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
			continue
		}
		
		// Read migration content
		content, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}
		
		name := strings.TrimSuffix(parts[1], ".sql")
		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		})
	}
	
	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	
	return migrations, nil
}

// RunMigrations executes all pending migrations
func (m *Migrator) RunMigrations() error {
	// Create migrations table if it doesn't exist
	if err := m.CreateMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	
	// Get applied migrations
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}
	
	// Load all migrations
	migrations, err := m.LoadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}
	
	// Run pending migrations
	for _, migration := range migrations {
		if applied[migration.Version] {
			continue // Skip already applied migrations
		}
		
		fmt.Printf("Running migration %d: %s\n", migration.Version, migration.Name)
		
		// Start transaction
		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction for migration %d: %w", migration.Version, err)
		}
		
		// Execute migration SQL
		if _, err := tx.Exec(migration.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %d: %w", migration.Version, err)
		}
		
		// Record migration as applied
		if _, err := tx.Exec("INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", migration.Version, migration.Name); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}
		
		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}
		
		fmt.Printf("Migration %d completed successfully\n", migration.Version)
	}
	
	return nil
}

// GetMigrationStatus returns the current migration status
func (m *Migrator) GetMigrationStatus() error {
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return err
	}
	
	migrations, err := m.LoadMigrations()
	if err != nil {
		return err
	}
	
	fmt.Println("Migration Status:")
	fmt.Println("================")
	
	for _, migration := range migrations {
		status := "PENDING"
		if applied[migration.Version] {
			status = "APPLIED"
		}
		fmt.Printf("%d: %s [%s]\n", migration.Version, migration.Name, status)
	}
	
	return nil
}