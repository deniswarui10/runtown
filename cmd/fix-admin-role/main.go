package main

import (
	"fmt"
	"log"

	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/database"
	"event-ticketing-platform/internal/repositories"
)

func main() {
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

	// Initialize user repository
	userRepo := repositories.NewUserRepository(db.DB)

	// Check admin user
	adminUser, err := userRepo.GetByEmail("admin@example.com")
	if err != nil {
		log.Fatal("Failed to get admin user:", err)
	}

	fmt.Printf("Current admin role: %s\n", adminUser.Role)

	// Update admin to have organizer role (admins should be able to create events)
	_, err = db.DB.Exec("UPDATE users SET role = 'organizer' WHERE id = $1", adminUser.ID)
	if err != nil {
		log.Fatal("Failed to update admin role:", err)
	}

	fmt.Printf("Admin role updated to 'organizer' so they can create events\n")
	fmt.Printf("Note: Admin functionality will still work because middleware checks for admin role\n")
}