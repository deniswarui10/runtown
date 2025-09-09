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

	// Create a test user if not exists
	userID := 232 // Assuming this user exists (denistakeprofit@gmail.com)

	// Create a test completed order
	orderReq := &models.OrderCreateRequest{
		UserID:       userID,
		EventID:      1,    // Assuming event with ID 1 exists
		TotalAmount:  5000, // $50.00 in cents
		BillingEmail: "denistakeprofit@gmail.com",
		BillingName:  "Denis Wamu",
		Status:       models.OrderPending,
	}

	order, err := orderRepo.Create(orderReq)
	if err != nil {
		log.Fatal("Failed to create order:", err)
	}

	fmt.Printf("Created order: %s (ID: %d)\n", order.OrderNumber, order.ID)

	// Create test tickets for the order
	ticketData := []struct {
		TicketTypeID int
		QRCode       string
	}{
		{TicketTypeID: 1, QRCode: fmt.Sprintf("TKT-%d-1-%d-abc123", order.ID, time.Now().Unix())},
		{TicketTypeID: 1, QRCode: fmt.Sprintf("TKT-%d-1-%d-def456", order.ID, time.Now().Unix())},
	}

	// Complete the order with tickets
	err = orderRepo.ProcessOrderCompletion(order.ID, "test-payment-123", ticketData)
	if err != nil {
		log.Fatal("Failed to complete order:", err)
	}

	fmt.Printf("Order completed successfully with %d tickets\n", len(ticketData))

	// Create another pending order
	orderReq2 := &models.OrderCreateRequest{
		UserID:       userID,
		EventID:      1,
		TotalAmount:  3000, // $30.00 in cents
		BillingEmail: "denistakeprofit@gmail.com",
		BillingName:  "Denis Wamu",
		Status:       models.OrderPending,
	}

	order2, err := orderRepo.Create(orderReq2)
	if err != nil {
		log.Fatal("Failed to create second order:", err)
	}

	fmt.Printf("Created pending order: %s (ID: %d)\n", order2.OrderNumber, order2.ID)

	fmt.Println("Test orders created successfully!")
}
