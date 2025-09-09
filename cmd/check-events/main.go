package main

import (
	"database/sql"
	"fmt"
	"log"

	"event-ticketing-platform/internal/config"

	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("Checking Events")

	var totalEvents int
	err = db.QueryRow("SELECT COUNT(*) FROM events").Scan(&totalEvents)
	if err != nil {
		log.Fatal("Failed to count events:", err)
	}
	fmt.Printf("Total Events: %d\n", totalEvents)

	var publishedEvents int
	err = db.QueryRow("SELECT COUNT(*) FROM events WHERE status = 'published'").Scan(&publishedEvents)
	if err != nil {
		log.Fatal("Failed to count published events:", err)
	}
	fmt.Printf("Published Events: %d\n", publishedEvents)
}