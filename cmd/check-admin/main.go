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

	fmt.Printf("Admin User Details:\n")
	fmt.Printf("ID: %d\n", adminUser.ID)
	fmt.Printf("Email: %s\n", adminUser.Email)
	fmt.Printf("First Name: %s\n", adminUser.FirstName)
	fmt.Printf("Last Name: %s\n", adminUser.LastName)
	fmt.Printf("Role: %s\n", adminUser.Role)
	fmt.Printf("Is Active: %t\n", adminUser.IsActive)
	fmt.Printf("Email Verified: %t\n", adminUser.EmailVerified)
	fmt.Printf("Created At: %s\n", adminUser.CreatedAt)

	// Fix the admin user if needed
	if !adminUser.IsActive {
		fmt.Printf("\nActivating admin user...\n")
		_, err = db.DB.Exec("UPDATE users SET is_active = true WHERE id = $1", adminUser.ID)
		if err != nil {
			log.Fatal("Failed to activate admin user:", err)
		}
		fmt.Printf("Admin user activated successfully!\n")
	}

	// Also ensure the role is correct
	if adminUser.Role != "admin" {
		fmt.Printf("\nUpdating admin role...\n")
		_, err = db.DB.Exec("UPDATE users SET role = 'admin' WHERE id = $1", adminUser.ID)
		if err != nil {
			log.Fatal("Failed to update admin role:", err)
		}
		fmt.Printf("Admin role updated successfully!\n")
	}
}