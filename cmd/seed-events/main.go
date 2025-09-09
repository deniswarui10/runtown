package main

import (
	"fmt"
	"log"
	"time"

	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/database"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
	"event-ticketing-platform/internal/utils"
)

func main() {
	fmt.Println("🌱 Seeding Events for denistakeprofit@gmail.com")
	
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

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db.DB)
	eventRepo := repositories.NewEventRepository(db.DB)
	ticketRepo := repositories.NewTicketRepository(db.DB)
	orderRepo := repositories.NewOrderRepository(db.DB)

	// Find or create the user
	user, err := userRepo.GetByEmail("denistakeprofit@gmail.com")
	if err != nil {
		fmt.Println("User not found, creating denistakeprofit@gmail.com...")
		
		// Create the user
		hashedPassword, _ := utils.HashPassword("SecurePassword123!")
		
		createReq := &models.UserCreateRequest{
			Email:     "denistakeprofit@gmail.com",
			Password:  hashedPassword,
			FirstName: "Denis",
			LastName:  "TakeProfit",
			Role:      models.UserRoleOrganizer,
		}
		
		user, err = userRepo.Create(createReq)
		if err != nil {
			log.Fatal("Failed to create user:", err)
		}
		
		// Verify the user's email
		err = userRepo.VerifyEmail(user.ID)
		if err != nil {
			log.Fatal("Failed to verify user email:", err)
		}
		
		fmt.Printf("✅ Created user: %s %s (%s)\n", user.FirstName, user.LastName, user.Email)
	} else {
		fmt.Printf("✅ Found existing user: %s %s (%s)\n", user.FirstName, user.LastName, user.Email)
		
		// Make sure user is organizer
		if user.Role != models.UserRoleOrganizer {
			fmt.Println("✅ User role is already organizer or admin")
		}
	}

	// Create sample events
	events := []struct {
		Title       string
		Description string
		Category    string
		Location    string
		StartTime   time.Time
		EndTime     time.Time
		TicketTypes []struct {
			Name        string
			Description string
			Price       int // in cents
			Quantity    int
		}
	}{
		{
			Title:       "Tech Innovation Summit 2024",
			Description: "Join industry leaders and innovators for a day of cutting-edge technology discussions, networking, and insights into the future of tech. Featuring keynote speakers from major tech companies, startup showcases, and hands-on workshops.",
			Category:    "technology",
			Location:    "San Francisco Convention Center, CA",
			StartTime:   time.Now().AddDate(0, 1, 15).Truncate(time.Hour).Add(9 * time.Hour),
			EndTime:     time.Now().AddDate(0, 1, 15).Truncate(time.Hour).Add(17 * time.Hour),
			TicketTypes: []struct {
				Name        string
				Description string
				Price       int
				Quantity    int
			}{
				{"Early Bird", "Limited time early bird pricing", 15000, 100}, // $150
				{"General Admission", "Standard conference access", 25000, 300}, // $250
				{"VIP Pass", "Includes networking dinner and premium seating", 50000, 50}, // $500
			},
		},
		{
			Title:       "Digital Marketing Masterclass",
			Description: "Learn the latest digital marketing strategies from industry experts. This comprehensive workshop covers SEO, social media marketing, content strategy, and conversion optimization techniques that drive real results.",
			Category:    "business",
			Location:    "New York Business Center, NY",
			StartTime:   time.Now().AddDate(0, 0, 20).Truncate(time.Hour).Add(10 * time.Hour),
			EndTime:     time.Now().AddDate(0, 0, 20).Truncate(time.Hour).Add(16 * time.Hour),
			TicketTypes: []struct {
				Name        string
				Description string
				Price       int
				Quantity    int
			}{
				{"Standard", "Workshop access and materials", 12000, 150}, // $120
				{"Premium", "Includes 1-on-1 consultation session", 20000, 75}, // $200
			},
		},
		{
			Title:       "Startup Pitch Competition",
			Description: "Watch promising startups pitch their innovative ideas to a panel of experienced investors and entrepreneurs. Network with founders, investors, and industry professionals in this exciting entrepreneurial event.",
			Category:    "business",
			Location:    "Austin Startup Hub, TX",
			StartTime:   time.Now().AddDate(0, 0, 30).Truncate(time.Hour).Add(18 * time.Hour),
			EndTime:     time.Now().AddDate(0, 0, 30).Truncate(time.Hour).Add(21 * time.Hour),
			TicketTypes: []struct {
				Name        string
				Description string
				Price       int
				Quantity    int
			}{
				{"General", "Event access and networking", 5000, 200}, // $50
				{"Investor", "Priority seating and exclusive networking", 15000, 50}, // $150
			},
		},
		{
			Title:       "AI & Machine Learning Conference",
			Description: "Explore the latest developments in artificial intelligence and machine learning. Sessions cover practical applications, ethical considerations, and future trends in AI technology across various industries.",
			Category:    "technology",
			Location:    "Seattle Tech Center, WA",
			StartTime:   time.Now().AddDate(0, 2, 5).Truncate(time.Hour).Add(8 * time.Hour),
			EndTime:     time.Now().AddDate(0, 2, 6).Truncate(time.Hour).Add(17 * time.Hour),
			TicketTypes: []struct {
				Name        string
				Description string
				Price       int
				Quantity    int
			}{
				{"Student", "Special pricing for students", 8000, 100}, // $80
				{"Professional", "Full conference access", 30000, 250}, // $300
				{"Corporate", "Team packages available", 45000, 100}, // $450
			},
		},
		{
			Title:       "Creative Design Workshop",
			Description: "Unleash your creativity in this hands-on design workshop. Learn modern design principles, typography, color theory, and digital design tools from award-winning designers and creative directors.",
			Category:    "arts",
			Location:    "Los Angeles Design Studio, CA",
			StartTime:   time.Now().AddDate(0, 0, 12).Truncate(time.Hour).Add(13 * time.Hour),
			EndTime:     time.Now().AddDate(0, 0, 12).Truncate(time.Hour).Add(18 * time.Hour),
			TicketTypes: []struct {
				Name        string
				Description string
				Price       int
				Quantity    int
			}{
				{"Basic", "Workshop access and materials", 8000, 80}, // $80
				{"Pro", "Includes design software licenses", 15000, 40}, // $150
			},
		},
	}

	fmt.Println("\n🎫 Creating events and ticket types...")
	
	for i, eventData := range events {
		// Get category ID
		categories, err := eventRepo.GetCategories()
		if err != nil {
			log.Printf("Failed to get categories: %v", err)
			continue
		}
		
		categoryID := 1 // Default to first category
		for _, cat := range categories {
			if cat.Slug == eventData.Category {
				categoryID = cat.ID
				break
			}
		}

		// Create event request (no image fields to satisfy constraint)
		eventReq := &models.EventCreateRequest{
			Title:       eventData.Title,
			Description: eventData.Description,
			Location:    eventData.Location,
			StartDate:   eventData.StartTime,
			EndDate:     eventData.EndTime,
			CategoryID:  categoryID,
			Status:      models.StatusPublished,
			// Leave image fields empty to satisfy the constraint
		}

		createdEvent, err := eventRepo.Create(eventReq, user.ID)
		if err != nil {
			log.Printf("Failed to create event %s: %v", eventData.Title, err)
			continue
		}

		fmt.Printf("✅ Created event: %s (ID: %d)\n", createdEvent.Title, createdEvent.ID)

		// Create ticket types for this event
		for _, ticketData := range eventData.TicketTypes {
			ticketTypeReq := &models.TicketTypeCreateRequest{
				EventID:     createdEvent.ID,
				Name:        ticketData.Name,
				Description: ticketData.Description,
				Price:       ticketData.Price,
				Quantity:    ticketData.Quantity,
				SaleStart:   time.Now().Add(-24 * time.Hour), // Sales started yesterday
				SaleEnd:     eventData.StartTime.Add(-1 * time.Hour), // Sales end 1 hour before event
			}

			createdTicketType, err := ticketRepo.CreateTicketType(ticketTypeReq)
			if err != nil {
				log.Printf("Failed to create ticket type %s: %v", ticketData.Name, err)
				continue
			}

			fmt.Printf("   ✅ Created ticket type: %s - $%.2f (%d available)\n", 
				createdTicketType.Name, 
				float64(createdTicketType.Price)/100, 
				createdTicketType.Quantity)
		}

		// Create some sample orders to give the organizer some earnings
		if i < 2 { // Only for first 2 events
			fmt.Printf("   💰 Creating sample orders for revenue...\n")
			
			// Create sample customer
			customerEmail := fmt.Sprintf("customer%d@example.com", i+1)
			customer, err := userRepo.GetByEmail(customerEmail)
			if err != nil {
				// Create customer
				hashedPassword, _ := utils.HashPassword("CustomerPass123!")
				
				createReq := &models.UserCreateRequest{
					Email:     customerEmail,
					Password:  hashedPassword,
					FirstName: "Sample",
					LastName:  fmt.Sprintf("Customer%d", i+1),
					Role:      models.UserRoleUser, // Use UserRoleUser instead of UserRoleCustomer
				}
				
				customer, err = userRepo.Create(createReq)
				if err != nil {
					log.Printf("Failed to create customer: %v", err)
					continue
				}
				
				userRepo.VerifyEmail(customer.ID)
			}

			// Get ticket types for this event
			ticketTypes, err := ticketRepo.GetTicketTypesByEvent(createdEvent.ID)
			if err != nil {
				log.Printf("Failed to get ticket types: %v", err)
				continue
			}

			// Create 2-3 sample orders
			for j := 0; j < 3; j++ {
				if len(ticketTypes) > 0 {
					ticketType := ticketTypes[j%len(ticketTypes)]
					quantity := 1 + j // 1, 2, 3 tickets
					
					orderReq := &models.OrderCreateRequest{
						UserID:       customer.ID,
						EventID:      createdEvent.ID,
						TotalAmount:  ticketType.Price * quantity,
						Status:       models.OrderCompleted,
						BillingEmail: customer.Email,
						BillingName:  fmt.Sprintf("%s %s", customer.FirstName, customer.LastName),
					}

					createdOrder, err := orderRepo.Create(orderReq)
					if err != nil {
						log.Printf("Failed to create order: %v", err)
						continue
					}

					fmt.Printf("   💳 Created order: %s - $%.2f (%d tickets)\n", 
						createdOrder.OrderNumber, 
						float64(createdOrder.TotalAmount)/100, 
						quantity)
				}
			}
		}
	}

	fmt.Println("\n🎉 Seeding completed successfully!")
	fmt.Println("📊 Summary:")
	fmt.Printf("   - User: %s %s (%s)\n", user.FirstName, user.LastName, user.Email)
	fmt.Printf("   - Events created: %d\n", len(events))
	fmt.Printf("   - Sample orders created for revenue tracking\n")
	fmt.Println("\n🔗 You can now:")
	fmt.Println("   1. Login with denistakeprofit@gmail.com / SecurePassword123!")
	fmt.Println("   2. Access organizer dashboard at /organizer/dashboard")
	fmt.Println("   3. View events at /organizer/events")
	fmt.Println("   4. Check withdrawals at /organizer/withdrawals")
	fmt.Println("   5. Browse public events at /events")
}