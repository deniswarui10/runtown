package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(".env.local"); err != nil {
		log.Printf("Warning: Could not load .env.local file: %v", err)
	}

	// Get database URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("🔍 Checking all users in the database...")
	fmt.Println(strings.Repeat("=", 50))

	// Query all users
	rows, err := db.Query(`
		SELECT id, email, first_name, last_name, role, is_active, email_verified, created_at
		FROM users 
		ORDER BY id
	`)
	if err != nil {
		log.Fatalf("Failed to query users: %v", err)
	}
	defer rows.Close()

	userCount := 0
	for rows.Next() {
		var id int
		var email, firstName, lastName, role string
		var isActive, emailVerified bool
		var createdAt string

		err := rows.Scan(&id, &email, &firstName, &lastName, &role, &isActive, &emailVerified, &createdAt)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		userCount++
		fmt.Printf("\n👤 User #%d:\n", userCount)
		fmt.Printf("   ID: %d\n", id)
		fmt.Printf("   Email: %s\n", email)
		fmt.Printf("   Name: %s %s\n", firstName, lastName)
		fmt.Printf("   Role: %s\n", role)
		fmt.Printf("   Active: %t\n", isActive)
		fmt.Printf("   Email Verified: %t\n", emailVerified)
		fmt.Printf("   Created: %s\n", createdAt)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating rows: %v", err)
	}

	fmt.Printf("\n📊 Total users found: %d\n", userCount)

	if userCount == 0 {
		fmt.Println("\n⚠️  No users found in database!")
		fmt.Println("   You may need to create a user account first.")
	} else {
		fmt.Println("\n🔐 Available login credentials:")
		fmt.Println("   - admin@example.com (if admin exists)")
		fmt.Println("   - denistakeprofit@gmail.com (if organizer exists)")
		fmt.Println("   - Default password: SecurePassword123!")
	}
}