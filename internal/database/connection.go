package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

type Config struct {
	URL      string // Full database URL
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func NewConnection(config Config) (*DB, error) {
	var dsn string
	
	// Use full URL if available, otherwise construct from components
	if config.URL != "" {
		dsn = config.URL
	} else {
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)                 // Maximum number of open connections
	db.SetMaxIdleConns(5)                  // Maximum number of idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum lifetime of a connection

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

// RunMigrations runs all pending database migrations
func (db *DB) RunMigrations() error {
	migrator := NewMigrator(db.DB)
	return migrator.RunMigrations()
}

// GetMigrationStatus shows the current migration status
func (db *DB) GetMigrationStatus() error {
	migrator := NewMigrator(db.DB)
	return migrator.GetMigrationStatus()
}