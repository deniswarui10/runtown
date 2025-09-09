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
	fmt.Println("🌱 Simple Seeding for denistakeprofit@gmail.com")
	
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

	// Create or update user
	hashedPassword, _ := utils.HashPassword("SecurePassword123!")
	
	// Insert or update user
	userQuery := `
		INSERT INTO users (email, password_hash, first_name, last_name, role, email_verified, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, true, NOW(), NOW())
		ON CONFLICT (email) 
		DO UPDATE SET 
			role = EXCLUDED.role,
			updated_at = NOW()
		RETURNING id, first_name, last_name, email`
	
	var userID int
	var firstName, lastName, email string
	err = db.DB.QueryRow(userQuery, "denistakeprofit@gmail.com", hashedPassword, "Denis", "TakeProfit", "organizer").Scan(&userID, &firstName, &lastName, &email)
	if err != nil {
		log.Fatal("Failed to create/update user:", err)
	}
	
	fmt.Printf("✅ User ready: %s %s (%s) - ID: %d\n", firstName, lastName, email, userID)

	// Create events with proper NULL handling for image fields
	events := []struct {
		Title       string
		Description string
		Location    string
		StartDate   time.Time
		EndDate     time.Time
		CategoryID  int
	}{
		{
			Title:       "Tech Innovation Summit 2024",
			Description: "Join industry leaders and innovators for a day of cutting-edge technology discussions, networking, and insights into the future of tech. Featuring keynote speakers from major tech companies, startup showcases, and hands-on workshops.",
			Location:    "San Francisco Convention Center, CA",
			StartDate:   time.Now().AddDate(0, 1, 15).Truncate(time.Hour).Add(9 * time.Hour),
			EndDate:     time.Now().AddDate(0, 1, 15).Truncate(time.Hour).Add(17 * time.Hour),
			CategoryID:  105, // Technology
		},
		{
			Title:       "Digital Marketing Masterclass",
			Description: "Learn the latest digital marketing strategies from industry experts. This comprehensive workshop covers SEO, social media marketing, content strategy, and conversion optimization techniques that drive real results.",
			Location:    "New York Business Center, NY",
			StartDate:   time.Now().AddDate(0, 0, 20).Truncate(time.Hour).Add(10 * time.Hour),
			EndDate:     time.Now().AddDate(0, 0, 20).Truncate(time.Hour).Add(16 * time.Hour),
			CategoryID:  103, // Business
		},
		{
			Title:       "Startup Pitch Competition",
			Description: "Watch promising startups pitch their innovative ideas to a panel of experienced investors and entrepreneurs. Network with founders, investors, and industry professionals in this exciting entrepreneurial event.",
			Location:    "Austin Startup Hub, TX",
			StartDate:   time.Now().AddDate(0, 0, 30).Truncate(time.Hour).Add(18 * time.Hour),
			EndDate:     time.Now().AddDate(0, 0, 30).Truncate(time.Hour).Add(21 * time.Hour),
			CategoryID:  103, // Business
		},
		{
			Title:       "AI & Machine Learning Conference",
			Description: "Explore the latest developments in artificial intelligence and machine learning. Sessions cover practical applications, ethical considerations, and future trends in AI technology across various industries.",
			Location:    "Seattle Tech Center, WA",
			StartDate:   time.Now().AddDate(0, 2, 5).Truncate(time.Hour).Add(8 * time.Hour),
			EndDate:     time.Now().AddDate(0, 2, 6).Truncate(time.Hour).Add(17 * time.Hour),
			CategoryID:  105, // Technology
		},
		{
			Title:       "Creative Design Workshop",
			Description: "Unleash your creativity in this hands-on design workshop. Learn modern design principles, typography, color theory, and digital design tools from award-winning designers and creative directors.",
			Location:    "Los Angeles Design Studio, CA",
			StartDate:   time.Now().AddDate(0, 0, 12).Truncate(time.Hour).Add(13 * time.Hour),
			EndDate:     time.Now().AddDate(0, 0, 12).Truncate(time.Hour).Add(18 * time.Hour),
			CategoryID:  102, // Arts & Culture
		},
	}

	fmt.Println("\n🎫 Creating events...")
	
	for i, eventData := range events {
		// Insert event with NULL image fields
		eventQuery := `
			INSERT INTO events (title, description, start_date, end_date, location, category_id, organizer_id, 
			                   image_url, image_key, image_size, image_format, image_width, image_height, image_uploaded_at,
			                   status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, NULL, 0, NULL, 0, 0, NULL, 'published', NOW(), NOW())
			RETURNING id, title`
		
		var eventID int
		var eventTitle string
		err = db.DB.QueryRow(eventQuery, eventData.Title, eventData.Description, eventData.StartDate, eventData.EndDate, 
			eventData.Location, eventData.CategoryID, userID).Scan(&eventID, &eventTitle)
		if err != nil {
			log.Printf("Failed to create event %s: %v", eventData.Title, err)
			continue
		}
		
		fmt.Printf("✅ Created event: %s (ID: %d)\n", eventTitle, eventID)

		// Create ticket types for this event
		ticketTypes := []struct {
			Name        string
			Description string
			Price       int // in cents
			Quantity    int
		}{
			{"Early Bird", "Limited time early bird pricing", 15000, 50},   // $150
			{"General Admission", "Standard event access", 25000, 200},     // $250
			{"VIP Pass", "Premium experience with extras", 50000, 25},      // $500
		}

		for _, ticketData := range ticketTypes {
			ticketQuery := `
				INSERT INTO ticket_types (event_id, name, description, price, quantity, sold, sale_start, sale_end, created_at)
				VALUES ($1, $2, $3, $4, $5, 0, $6, $7, NOW())
				RETURNING id, name, price`
			
			var ticketID int
			var ticketName string
			var ticketPrice int
			saleStart := time.Now().Add(-24 * time.Hour) // Sales started yesterday
			saleEnd := eventData.StartDate.Add(-1 * time.Hour) // Sales end 1 hour before event
			
			err = db.DB.QueryRow(ticketQuery, eventID, ticketData.Name, ticketData.Description, 
				ticketData.Price, ticketData.Quantity, saleStart, saleEnd).Scan(&ticketID, &ticketName, &ticketPrice)
			if err != nil {
				log.Printf("Failed to create ticket type %s: %v", ticketData.Name, err)
				continue
			}
			
			fmt.Printf("   ✅ Created ticket type: %s - $%.2f (%d available)\n", 
				ticketName, float64(ticketPrice)/100, ticketData.Quantity)
		}

		// Create some sample orders for the first 2 events to give revenue
		if i < 2 {
			fmt.Printf("   💰 Creating sample orders for revenue...\n")
			
			// Create sample customer
			customerEmail := fmt.Sprintf("customer%d@example.com", i+1)
			customerPassword, _ := utils.HashPassword("CustomerPass123!")
			
			customerQuery := `
				INSERT INTO users (email, password_hash, first_name, last_name, role, email_verified, is_active, created_at, updated_at)
				VALUES ($1, $2, $3, $4, 'user', true, true, NOW(), NOW())
				ON CONFLICT (email) DO NOTHING
				RETURNING id`
			
			var customerID int
			err = db.DB.QueryRow(customerQuery, customerEmail, customerPassword, "Sample", fmt.Sprintf("Customer%d", i+1)).Scan(&customerID)
			if err != nil {
				// Customer might already exist, get their ID
				err = db.DB.QueryRow("SELECT id FROM users WHERE email = $1", customerEmail).Scan(&customerID)
				if err != nil {
					log.Printf("Failed to get customer ID: %v", err)
					continue
				}
			}

			// Get ticket types for this event
			ticketQuery := `SELECT id, price FROM ticket_types WHERE event_id = $1 ORDER BY price ASC`
			rows, err := db.DB.Query(ticketQuery, eventID)
			if err != nil {
				log.Printf("Failed to get ticket types: %v", err)
				continue
			}
			
			var ticketTypeIDs []int
			var ticketPrices []int
			for rows.Next() {
				var id, price int
				rows.Scan(&id, &price)
				ticketTypeIDs = append(ticketTypeIDs, id)
				ticketPrices = append(ticketPrices, price)
			}
			rows.Close()

			// Create 2-3 sample orders
			for j := 0; j < 3 && j < len(ticketTypeIDs); j++ {
				quantity := 1 + j // 1, 2, 3 tickets
				totalAmount := ticketPrices[j] * quantity
				orderNumber := fmt.Sprintf("ORD-%d-%d-%d", eventID, customerID, j)
				
				orderQuery := `
					INSERT INTO orders (user_id, event_id, order_number, total_amount, status, billing_email, billing_name, created_at, updated_at)
					VALUES ($1, $2, $3, $4, 'completed', $5, $6, NOW(), NOW())
					RETURNING id, order_number, total_amount`
				
				var orderID int
				var orderNum string
				var orderAmount int
				err = db.DB.QueryRow(orderQuery, customerID, eventID, orderNumber, totalAmount, 
					customerEmail, fmt.Sprintf("Sample Customer%d", i+1)).Scan(&orderID, &orderNum, &orderAmount)
				if err != nil {
					log.Printf("Failed to create order: %v", err)
					continue
				}

				// Update ticket type sold count
				updateQuery := `UPDATE ticket_types SET sold = sold + $1 WHERE id = $2`
				_, err = db.DB.Exec(updateQuery, quantity, ticketTypeIDs[j])
				if err != nil {
					log.Printf("Failed to update ticket sold count: %v", err)
				}

				fmt.Printf("   💳 Created order: %s - $%.2f (%d tickets)\n", 
					orderNum, float64(orderAmount)/100, quantity)
			}
		}
	}

	fmt.Println("\n🎉 Seeding completed successfully!")
	fmt.Println("📊 Summary:")
	fmt.Printf("   - User: %s %s (%s)\n", firstName, lastName, email)
	fmt.Printf("   - Events created: %d\n", len(events))
	fmt.Printf("   - Sample orders created for revenue tracking\n")
	fmt.Println("\n🔗 You can now:")
	fmt.Println("   1. Login with denistakeprofit@gmail.com / SecurePassword123!")
	fmt.Println("   2. Access organizer dashboard at /organizer/dashboard")
	fmt.Println("   3. View events at /organizer/events")
	fmt.Println("   4. Check withdrawals at /organizer/withdrawals")
	fmt.Println("   5. Browse public events at /events")
}