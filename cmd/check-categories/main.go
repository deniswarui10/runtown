package main

import (
	"fmt"
	"log"

	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/database"
)

func main() {
	fmt.Println("🔍 Checking Categories")
	
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

	// Check categories
	rows, err := db.DB.Query("SELECT id, name, slug FROM categories ORDER BY id")
	if err != nil {
		log.Fatal("Failed to query categories:", err)
	}
	defer rows.Close()

	fmt.Println("📂 Available Categories:")
	for rows.Next() {
		var id int
		var name, slug string
		err := rows.Scan(&id, &name, &slug)
		if err != nil {
			log.Fatal("Failed to scan category:", err)
		}
		fmt.Printf("   ID: %d, Name: %s, Slug: %s\n", id, name, slug)
	}
}