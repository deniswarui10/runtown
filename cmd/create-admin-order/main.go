package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"event-ticketing-platform/internal/database"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load(".env.local")
	if err != nil {
		log.Printf("Warning: Could not load .env.local file: %v", err)
	}

	// Get database configuration from environment
	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	if port == 0 {
		port = 5432 // Default PostgreSQL port
	}
	if os.Getenv("DB_HOST") == "" {
		os.Setenv("DB_HOST", "localhost")
	}
	if os.Getenv("DB_USER") == "" {
		os.Setenv("DB_USER", "runtown_user")
	}
	if os.Getenv("DB_PASSWORD") == "" {
		os.Setenv("DB_PASSWORD", "your_password")
	}
	if os.Getenv("DB_NAME") == "" {
		os.Setenv("DB_NAME", "runtown_db")
	}
	if os.Getenv("DB_SSLMODE") == "" {
		os.Setenv("DB_SSLMODE", "disable")
	}

	fmt.Printf("Connecting to database: %s:%d/%s\n", os.Getenv("DB_HOST"), port, os.Getenv("DB_NAME"))

	dbConfig := database.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     port,
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize repositories
	orderRepo := repositories.NewOrderRepository(db.DB)

	// Create orders for the main admin user (Denis - ID 232)
	userID := 232
	eventID := 1 // Use first event

	// Create first completed order
	orderReq1 := &models.OrderCreateRequest{
		UserID:       userID,
		EventID:      eventID,
		TotalAmount:  7500, // $75.00 in cents
		BillingEmail: "denistakeprofit@gmail.com",
		BillingName:  "Denis Warui",
		Status:       models.OrderPending,
	}

	order1, err := orderRepo.Create(orderReq1)
	if err != nil {
		log.Fatal("Failed to create first order:", err)
	}

	fmt.Printf("Created order: %s (ID: %d)\n", order1.OrderNumber, order1.ID)

	// Complete the order with tickets
	ticketData1 := []struct {
		TicketTypeID int
		QRCode       string
	}{
		{TicketTypeID: 1, QRCode: fmt.Sprintf("TKT-%d-1-%d-abc123", order1.ID, time.Now().Unix())},
		{TicketTypeID: 1, QRCode: fmt.Sprintf("TKT-%d-1-%d-def456", order1.ID, time.Now().Unix())},
	}

	err = orderRepo.ProcessOrderCompletion(order1.ID, "test-payment-admin-1", ticketData1)
	if err != nil {
		log.Fatal("Failed to complete first order:", err)
	}

	fmt.Printf("✅ Order 1 completed successfully with %d tickets\n", len(ticketData1))

	// Create second completed order
	orderReq2 := &models.OrderCreateRequest{
		UserID:       userID,
		EventID:      2,     // Different event
		TotalAmount:  12000, // $120.00 in cents
		BillingEmail: "denistakeprofit@gmail.com",
		BillingName:  "Denis Warui",
		Status:       models.OrderPending,
	}

	order2, err := orderRepo.Create(orderReq2)
	if err != nil {
		log.Fatal("Failed to create second order:", err)
	}

	fmt.Printf("Created order: %s (ID: %d)\n", order2.OrderNumber, order2.ID)

	// Complete the order with tickets
	ticketData2 := []struct {
		TicketTypeID int
		QRCode       string
	}{
		{TicketTypeID: 1, QRCode: fmt.Sprintf("TKT-%d-1-%d-xyz789", order2.ID, time.Now().Unix())},
		{TicketTypeID: 1, QRCode: fmt.Sprintf("TKT-%d-1-%d-uvw101", order2.ID, time.Now().Unix())},
		{TicketTypeID: 1, QRCode: fmt.Sprintf("TKT-%d-1-%d-rst112", order2.ID, time.Now().Unix())},
	}

	err = orderRepo.ProcessOrderCompletion(order2.ID, "test-payment-admin-2", ticketData2)
	if err != nil {
		log.Fatal("Failed to complete second order:", err)
	}

	fmt.Printf("✅ Order 2 completed successfully with %d tickets\n", len(ticketData2))

	// Create one pending order
	orderReq3 := &models.OrderCreateRequest{
		UserID:       userID,
		EventID:      3,
		TotalAmount:  4500, // $45.00 in cents
		BillingEmail: "denistakeprofit@gmail.com",
		BillingName:  "Denis Warui",
		Status:       models.OrderPending,
	}

	order3, err := orderRepo.Create(orderReq3)
	if err != nil {
		log.Fatal("Failed to create third order:", err)
	}

	fmt.Printf("Created pending order: %s (ID: %d)\n", order3.OrderNumber, order3.ID)

	fmt.Println("\n🎉 Admin test orders created successfully!")
	fmt.Println("📊 You should now see orders in your dashboard!")
}
