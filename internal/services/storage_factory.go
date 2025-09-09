package services

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"event-ticketing-platform/internal/config"
)

// StorageFactory creates storage services with proper fallback configuration
type StorageFactory struct {
	config *config.Config
}

// NewStorageFactory creates a new storage factory
func NewStorageFactory(cfg *config.Config) *StorageFactory {
	return &StorageFactory{config: cfg}
}

// CreateStorageService creates a storage service with R2 primary and local fallback
func (f *StorageFactory) CreateStorageService() (StorageServiceInterface, error) {
	// Try to create R2 service as primary
	r2Service, r2Err := NewR2Service(f.config.R2)
	
	// Create fallback storage service
	fallbackPath := filepath.Join("web", "static", "uploads")
	fallbackURL := fmt.Sprintf("http://%s:%s", f.config.Server.Host, f.config.Server.Port)
	fallbackService := NewFallbackStorageService(fallbackPath, fallbackURL)
	
	if r2Err != nil {
		fmt.Printf("Warning: R2 service unavailable, using fallback storage only: %v\n", r2Err)
		return fallbackService, nil
	}
	
	// Test R2 connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := r2Service.HealthCheck(ctx); err != nil {
		fmt.Printf("Warning: R2 health check failed, using fallback storage only: %v\n", err)
		return fallbackService, nil
	}
	
	// R2 is available, use it with fallback
	fmt.Println("R2 storage service initialized successfully")
	return NewStorageServiceWithFallback(r2Service, fallbackService), nil
}

// CreateImageService creates an image service with the configured storage
func (f *StorageFactory) CreateImageService() (ImageServiceInterface, error) {
	storage, err := f.CreateStorageService()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage service: %w", err)
	}
	
	return NewImageService(storage), nil
}

// SetupR2Bucket initializes the R2 bucket with proper configuration
func (f *StorageFactory) SetupR2Bucket() error {
	r2Service, err := NewR2Service(f.config.R2)
	if err != nil {
		return fmt.Errorf("failed to create R2 service: %w", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Create bucket if it doesn't exist
	if err := r2Service.CreateBucket(ctx); err != nil {
		return fmt.Errorf("failed to create R2 bucket: %w", err)
	}
	
	// Set CORS configuration
	if err := r2Service.SetBucketCORS(ctx); err != nil {
		return fmt.Errorf("failed to set R2 bucket CORS: %w", err)
	}
	
	fmt.Printf("R2 bucket '%s' configured successfully\n", f.config.R2.BucketName)
	return nil
}

// ValidateR2Configuration validates the R2 configuration
func (f *StorageFactory) ValidateR2Configuration() error {
	cfg := f.config.R2
	
	if cfg.AccountID == "" {
		return fmt.Errorf("R2_ACCOUNT_ID is required")
	}
	
	if cfg.AccessKeyID == "" {
		return fmt.Errorf("R2_ACCESS_KEY_ID is required")
	}
	
	if cfg.SecretAccessKey == "" {
		return fmt.Errorf("R2_SECRET_ACCESS_KEY is required")
	}
	
	if cfg.BucketName == "" {
		return fmt.Errorf("R2_BUCKET_NAME is required")
	}
	
	return nil
}

// GetStorageInfo returns information about the configured storage
func (f *StorageFactory) GetStorageInfo() map[string]interface{} {
	info := map[string]interface{}{
		"r2_configured": f.config.R2.AccessKeyID != "" && f.config.R2.SecretAccessKey != "",
		"bucket_name":   f.config.R2.BucketName,
		"public_url":    f.config.R2.PublicURL,
		"fallback_path": filepath.Join("web", "static", "uploads"),
	}
	
	// Test R2 connectivity if configured
	if info["r2_configured"].(bool) {
		r2Service, err := NewR2Service(f.config.R2)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			info["r2_available"] = r2Service.HealthCheck(ctx) == nil
		} else {
			info["r2_available"] = false
		}
	} else {
		info["r2_available"] = false
	}
	
	return info
}