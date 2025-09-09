package main

import (
	"log"

	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/database"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("Listing existing users...")

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

	// Query users
	rows, err := db.DB.Query(`
		SELECT id, email, first_name, last_name, role, 
		       email_verified_at, confirmed_at, created_at
		FROM users 
		ORDER BY id
	`)
	if err != nil {
		log.Fatal("Failed to query users:", err)
	}
	defer rows.Close()

	log.Println("Existing users:")
	log.Println("ID | Email | Name | Role | Email Verified | Confirmed | Created")
	log.Println("---|-------|------|------|----------------|-----------|--------")

	for rows.Next() {
		var id int
		var email, firstName, lastName, role string
		var emailVerified, confirmed, created interface{}

		err := rows.Scan(&id, &email, &firstName, &lastName, &role, &emailVerified, &confirmed, &created)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		log.Printf("%d | %s | %s %s | %s | %v | %v | %v", 
			id, email, firstName, lastName, role, emailVerified, confirmed, created)
	}

	if err = rows.Err(); err != nil {
		log.Fatal("Error iterating rows:", err)
	}
}