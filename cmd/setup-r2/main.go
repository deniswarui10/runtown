package main

import (
	"fmt"
	"log"
	"os"

	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create storage factory
	factory := services.NewStorageFactory(cfg)

	// Validate R2 configuration
	if err := factory.ValidateR2Configuration(); err != nil {
		log.Fatalf("R2 configuration validation failed: %v", err)
	}

	fmt.Println("R2 configuration is valid")

	// Get storage info
	info := factory.GetStorageInfo()
	fmt.Printf("Storage Information:\n")
	fmt.Printf("  R2 Configured: %v\n", info["r2_configured"])
	fmt.Printf("  R2 Available: %v\n", info["r2_available"])
	fmt.Printf("  Bucket Name: %s\n", info["bucket_name"])
	fmt.Printf("  Public URL: %s\n", info["public_url"])
	fmt.Printf("  Fallback Path: %s\n", info["fallback_path"])

	// Check if we should set up the bucket
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		fmt.Println("\nSetting up R2 bucket...")
		
		if err := factory.SetupR2Bucket(); err != nil {
			log.Fatalf("Failed to set up R2 bucket: %v", err)
		}
		
		fmt.Println("R2 bucket setup completed successfully!")
	} else {
		fmt.Println("\nTo set up the R2 bucket, run: go run cmd/setup-r2/main.go setup")
	}
}