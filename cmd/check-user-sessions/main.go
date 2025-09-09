package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(".env.local"); err != nil {
		log.Printf("Warning: Could not load .env.local file: %v", err)
	}

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Check users
	fmt.Println("=== USERS ===")
	rows, err := db.Query("SELECT id, email, first_name, last_name, role FROM users ORDER BY id")
	if err != nil {
		log.Fatal("Failed to query users:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var email, firstName, lastName, role string
		err := rows.Scan(&id, &email, &firstName, &lastName, &role)
		if err != nil {
			log.Fatal("Failed to scan user:", err)
		}
		fmt.Printf("ID: %d, Email: %s, Name: %s %s, Role: %s\n", id, email, firstName, lastName, role)
	}

	// Check sessions
	fmt.Println("\n=== SESSIONS ===")
	rows, err = db.Query(`
		SELECT s.id, s.user_id, u.email, s.expires_at, s.created_at 
		FROM sessions s 
		JOIN users u ON s.user_id = u.id 
		ORDER BY s.created_at DESC 
		LIMIT 10
	`)
	if err != nil {
		log.Fatal("Failed to query sessions:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sessionID string
		var userID int
		var email string
		var expiresAt, createdAt string
		err := rows.Scan(&sessionID, &userID, &email, &expiresAt, &createdAt)
		if err != nil {
			log.Fatal("Failed to scan session:", err)
		}
		fmt.Printf("Session: %s, UserID: %d, Email: %s, Expires: %s, Created: %s\n", 
			sessionID, userID, email, expiresAt, createdAt)
	}

	// Check for any users with ID = 0
	fmt.Println("\n=== CHECKING FOR INVALID USER IDs ===")
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = 0").Scan(&count)
	if err != nil {
		log.Fatal("Failed to check for invalid user IDs:", err)
	}
	fmt.Printf("Users with ID = 0: %d\n", count)

	// Check for any sessions with user_id = 0
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE user_id = 0").Scan(&count)
	if err != nil {
		log.Fatal("Failed to check for invalid session user IDs:", err)
	}
	fmt.Printf("Sessions with user_id = 0: %d\n", count)
}