package main

import (
	"fmt"
	"log"

	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/database"
	"event-ticketing-platform/internal/utils"
)

func main() {
	fmt.Println("💳 Creating Sample Orders for Revenue")
	
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

	// Create sample customers
	customers := []struct {
		Email     string
		FirstName string
		LastName  string
	}{
		{"customer1@example.com", "John", "Smith"},
		{"customer2@example.com", "Sarah", "Johnson"},
		{"customer3@example.com", "Mike", "Davis"},
	}

	for i, customer := range customers {
		hashedPassword, _ := utils.HashPassword("CustomerPass123!")
		
		// Insert customer
		customerQuery := `
			INSERT INTO users (email, password_hash, first_name, last_name, role, email_verified, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'attendee', true, true, NOW(), NOW())
			ON CONFLICT (email) DO UPDATE SET updated_at = NOW()
			RETURNING id`
		
		var customerID int
		err = db.DB.QueryRow(customerQuery, customer.Email, hashedPassword, customer.FirstName, customer.LastName).Scan(&customerID)
		if err != nil {
			log.Printf("Failed to create customer %s: %v", customer.Email, err)
			continue
		}
		
		fmt.Printf("✅ Created customer: %s %s (%s) - ID: %d\n", customer.FirstName, customer.LastName, customer.Email, customerID)

		// Get first 2 events for creating orders
		eventQuery := `SELECT id, title FROM events WHERE organizer_id = (SELECT id FROM users WHERE email = 'denistakeprofit@gmail.com') ORDER BY id LIMIT 2`
		eventRows, err := db.DB.Query(eventQuery)
		if err != nil {
			log.Printf("Failed to get events: %v", err)
			continue
		}

		var eventIDs []int
		var eventTitles []string
		for eventRows.Next() {
			var eventID int
			var eventTitle string
			eventRows.Scan(&eventID, &eventTitle)
			eventIDs = append(eventIDs, eventID)
			eventTitles = append(eventTitles, eventTitle)
		}
		eventRows.Close()

		// Create orders for each event
		for j, eventID := range eventIDs {
			// Get ticket types for this event
			ticketQuery := `SELECT id, name, price FROM ticket_types WHERE event_id = $1 ORDER BY price ASC LIMIT 1`
			var ticketTypeID int
			var ticketTypeName string
			var ticketPrice int
			
			err = db.DB.QueryRow(ticketQuery, eventID).Scan(&ticketTypeID, &ticketTypeName, &ticketPrice)
			if err != nil {
				log.Printf("Failed to get ticket type: %v", err)
				continue
			}

			quantity := 1 + (i+j)%3 // 1, 2, or 3 tickets
			totalAmount := ticketPrice * quantity
			orderNumber := fmt.Sprintf("ORD-%d-%d-%d", eventID, customerID, j)
			
			// Create order
			orderQuery := `
				INSERT INTO orders (user_id, event_id, order_number, total_amount, status, billing_email, billing_name, created_at, updated_at)
				VALUES ($1, $2, $3, $4, 'completed', $5, $6, NOW(), NOW())
				RETURNING id`
			
			var orderID int
			err = db.DB.QueryRow(orderQuery, customerID, eventID, orderNumber, totalAmount, 
				customer.Email, fmt.Sprintf("%s %s", customer.FirstName, customer.LastName)).Scan(&orderID)
			if err != nil {
				log.Printf("Failed to create order: %v", err)
				continue
			}

			// Update ticket type sold count
			updateQuery := `UPDATE ticket_types SET sold = sold + $1 WHERE id = $2`
			_, err = db.DB.Exec(updateQuery, quantity, ticketTypeID)
			if err != nil {
				log.Printf("Failed to update ticket sold count: %v", err)
			}

			fmt.Printf("   💳 Created order: %s - $%.2f (%d x %s tickets) for %s\n", 
				orderNumber, float64(totalAmount)/100, quantity, ticketTypeName, eventTitles[j])
		}
	}

	fmt.Println("\n🎉 Sample orders created successfully!")
	fmt.Println("💰 The organizer now has revenue from ticket sales")
}