package main

import (
	"fmt"
	"log"

	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/database"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
	"event-ticketing-platform/internal/utils"
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

	// Check if admin user already exists
	existingAdmin, err := userRepo.GetByEmail("admin@example.com")
	if err == nil {
		fmt.Printf("Admin user already exists with ID: %d\n", existingAdmin.ID)
		
		// Update the password with correct hash
		passwordHash, err := utils.HashPassword("admin123")
		if err != nil {
			log.Fatal("Failed to hash password:", err)
		}
		
		// Update the user's password directly in the database
		_, err = db.DB.Exec("UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", passwordHash, existingAdmin.ID)
		if err != nil {
			log.Fatal("Failed to update admin password:", err)
		}
		
		fmt.Printf("Admin password updated successfully!\n")
		fmt.Printf("Email: admin@example.com\n")
		fmt.Printf("Password: admin123\n")
		return
	}

	// Hash password using the same method as the auth service
	passwordHash, err := utils.HashPassword("admin123")
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	// Create admin user
	adminUser := &models.UserCreateRequest{
		Email:     "admin@example.com",
		Password:  passwordHash,
		FirstName: "Admin",
		LastName:  "User",
		Role:      models.UserRoleAdmin,
	}

	user, err := userRepo.Create(adminUser)
	if err != nil {
		log.Fatal("Failed to create admin user:", err)
	}

	// Mark email as verified and user as active
	err = userRepo.VerifyEmail(user.ID)
	if err != nil {
		log.Fatal("Failed to verify admin email:", err)
	}

	fmt.Printf("Admin user created successfully!\n")
	fmt.Printf("Email: admin@example.com\n")
	fmt.Printf("Password: admin123\n")
	fmt.Printf("User ID: %d\n", user.ID)
	fmt.Printf("You can now log in and access the admin dashboard at /admin\n")
}