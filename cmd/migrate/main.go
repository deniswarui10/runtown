package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/database"
)

func main() {
	var (
		statusFlag = flag.Bool("status", false, "Show migration status")
		upFlag     = flag.Bool("up", false, "Run pending migrations")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	dbConfig := database.Config{
		URL:      cfg.Database.URL,
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	switch {
	case *statusFlag:
		if err := db.GetMigrationStatus(); err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}
	case *upFlag:
		if err := db.RunMigrations(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("All migrations completed successfully!")
	default:
		fmt.Println("Usage:")
		fmt.Println("  go run cmd/migrate/main.go -status   # Show migration status")
		fmt.Println("  go run cmd/migrate/main.go -up       # Run pending migrations")
		os.Exit(1)
	}
}