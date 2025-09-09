package main

import (
	"fmt"
	"log"
	"time"

	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/database"
	"event-ticketing-platform/internal/utils"
)

func main() {
	fmt.Println("🧪 Creating Test User with Orders")

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

	// Create test user
	hashedPassword, _ := utils.HashPassword("password123")

	userQuery := `
		INSERT INTO users (email, password_hash, first_name, last_name, role, email_verified, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, true, NOW(), NOW())
		ON CONFLICT (email) 
		DO UPDATE SET 
			password_hash = EXCLUDED.password_hash,
			updated_at = NOW()
		RETURNING id, first_name, last_name, email`

	var userID int
	var firstName, lastName, email string
	err = db.DB.QueryRow(userQuery, "testuser@example.com", hashedPassword, "Test", "User", "attendee").Scan(&userID, &firstName, &lastName, &email)
	if err != nil {
		log.Fatal("Failed to create test user:", err)
	}

	fmt.Printf("✅ Test user created: %s %s (%s) - ID: %d\n", firstName, lastName, email, userID)

	// Get available events
	eventQuery := `SELECT id, title FROM events WHERE status = 'published' ORDER BY created_at DESC LIMIT 3`
	eventRows, err := db.DB.Query(eventQuery)
	if err != nil {
		log.Fatal("Failed to get events:", err)
	}

	var events []struct {
		ID    int
		Title string
	}

	for eventRows.Next() {
		var event struct {
			ID    int
			Title string
		}
		eventRows.Scan(&event.ID, &event.Title)
		events = append(events, event)
	}
	eventRows.Close()

	if len(events) == 0 {
		log.Fatal("No published events found. Please run seed-simple first.")
	}

	fmt.Printf("📅 Found %d events to create orders for\n", len(events))

	// Create orders for the test user
	for i, event := range events {
		// Get ticket types for this event
		ticketQuery := `SELECT id, price FROM ticket_types WHERE event_id = $1 ORDER BY price ASC LIMIT 1`
		var ticketTypeID, ticketPrice int
		err := db.DB.QueryRow(ticketQuery, event.ID).Scan(&ticketTypeID, &ticketPrice)
		if err != nil {
			fmt.Printf("❌ No ticket types found for event %d\n", event.ID)
			continue
		}

		// Create different types of orders
		var status string
		var orderSuffix string
		switch i {
		case 0:
			status = "completed"
			orderSuffix = "COMPLETED"
		case 1:
			status = "pending"
			orderSuffix = "PENDING"
		default:
			status = "completed"
			orderSuffix = "COMPLETED"
		}

		quantity := i + 1 // 1, 2, 3 tickets
		totalAmount := ticketPrice * quantity
		orderNumber := fmt.Sprintf("TEST-%s-%d", orderSuffix, time.Now().Unix()+int64(i))

		// Create order
		orderQuery := `
			INSERT INTO orders (user_id, event_id, order_number, total_amount, status, billing_email, billing_name, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW() - INTERVAL '%d days', NOW() - INTERVAL '%d days')
			RETURNING id, order_number, total_amount`

		var orderID int
		var orderNum string
		var orderAmount int

		// Create orders from different dates (recent history)
		daysAgo := i * 5 // 0, 5, 10 days ago

		err = db.DB.QueryRow(fmt.Sprintf(orderQuery, daysAgo, daysAgo), userID, event.ID, orderNumber, totalAmount, status, email, fmt.Sprintf("%s %s", firstName, lastName)).Scan(&orderID, &orderNum, &orderAmount)
		if err != nil {
			fmt.Printf("❌ Failed to create order for event %s: %v\n", event.Title, err)
			continue
		}

		// Create tickets for completed orders
		if status == "completed" {
			for j := 0; j < quantity; j++ {
				qrCode := fmt.Sprintf("QR-%d-%d-%d", orderID, ticketTypeID, j+1)
				ticketInsertQuery := `
					INSERT INTO tickets (order_id, ticket_type_id, qr_code, status, created_at, updated_at)
					VALUES ($1, $2, $3, 'active', NOW() - INTERVAL '%d days', NOW() - INTERVAL '%d days')`

				_, err = db.DB.Exec(fmt.Sprintf(ticketInsertQuery, daysAgo, daysAgo), orderID, ticketTypeID, qrCode)
				if err != nil {
					fmt.Printf("⚠️ Failed to create ticket %d for order %s: %v\n", j+1, orderNum, err)
				}
			}
		}

		fmt.Printf("✅ Created %s order: %s - KSh %.2f (%d tickets) for '%s'\n",
			status, orderNum, float64(orderAmount)/100, quantity, event.Title)
	}

	fmt.Println("\n🎉 Test user and orders created successfully!")
	fmt.Println("🔗 You can now:")
	fmt.Println("   1. Login with testuser@example.com / password123")
	fmt.Println("   2. View orders at /dashboard/orders")
	fmt.Println("   3. Check order details")
	fmt.Println("   4. Browse your dashboard at /dashboard")
}
