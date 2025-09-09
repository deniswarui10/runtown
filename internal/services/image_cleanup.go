package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"event-ticketing-platform/internal/models"
)

// EventRepositoryInterface defines the interface for event repository operations needed by cleanup service
type EventRepositoryInterface interface {
	GetByID(id int) (*models.Event, error)
}

// ImageCleanupService handles cleanup of orphaned images in R2 storage
type ImageCleanupService struct {
	storage     StorageService
	eventRepo   EventRepositoryInterface
	db          *sql.DB
}

// NewImageCleanupService creates a new image cleanup service
func NewImageCleanupService(storage StorageService, eventRepo EventRepositoryInterface, db *sql.DB) *ImageCleanupService {
	return &ImageCleanupService{
		storage:   storage,
		eventRepo: eventRepo,
		db:        db,
	}
}

// OrphanedImage represents an image that exists in storage but not in database
type OrphanedImage struct {
	Key        string    `json:"key"`
	URL        string    `json:"url"`
	Size       int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
}

// CleanupOrphanedImages removes images from R2 that are no longer referenced by any events
func (s *ImageCleanupService) CleanupOrphanedImages(ctx context.Context, dryRun bool) (*ImageCleanupResult, error) {
	log.Printf("Starting image cleanup (dry run: %v)", dryRun)
	
	// Get all image keys from database
	dbImageKeys, err := s.getImageKeysFromDatabase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get image keys from database: %w", err)
	}
	
	log.Printf("Found %d image keys in database", len(dbImageKeys))
	
	// Get all image keys from R2 storage
	storageImageKeys, err := s.getImageKeysFromStorage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get image keys from storage: %w", err)
	}
	
	log.Printf("Found %d image keys in storage", len(storageImageKeys))
	
	// Find orphaned images
	orphanedImages := s.findOrphanedImages(dbImageKeys, storageImageKeys)
	
	log.Printf("Found %d orphaned images", len(orphanedImages))
	
	result := &ImageCleanupResult{
		TotalImagesInStorage: len(storageImageKeys),
		TotalImagesInDB:      len(dbImageKeys),
		OrphanedImages:       orphanedImages,
		DryRun:              dryRun,
		CleanedUp:           []string{},
		Errors:              []string{},
	}
	
	// If not a dry run, delete orphaned images
	if !dryRun {
		for _, orphaned := range orphanedImages {
			if err := s.storage.Delete(ctx, orphaned.Key); err != nil {
				errorMsg := fmt.Sprintf("Failed to delete %s: %v", orphaned.Key, err)
				result.Errors = append(result.Errors, errorMsg)
				log.Printf("Error: %s", errorMsg)
			} else {
				result.CleanedUp = append(result.CleanedUp, orphaned.Key)
				log.Printf("Deleted orphaned image: %s", orphaned.Key)
			}
		}
	}
	
	log.Printf("Image cleanup completed. Cleaned up: %d, Errors: %d", len(result.CleanedUp), len(result.Errors))
	
	return result, nil
}

// CleanupExpiredImages removes images that were uploaded but never associated with an event (older than 24 hours)
func (s *ImageCleanupService) CleanupExpiredImages(ctx context.Context, dryRun bool) (*ImageCleanupResult, error) {
	log.Printf("Starting expired image cleanup (dry run: %v)", dryRun)
	
	// This would require tracking temporary uploads, which we'll implement later
	// For now, return empty result
	return &ImageCleanupResult{
		TotalImagesInStorage: 0,
		TotalImagesInDB:      0,
		OrphanedImages:       []OrphanedImage{},
		DryRun:              dryRun,
		CleanedUp:           []string{},
		Errors:              []string{},
	}, nil
}

// CleanupEventImages removes all images associated with a specific event
func (s *ImageCleanupService) CleanupEventImages(ctx context.Context, eventID int) error {
	// Get event to find its image key
	event, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}
	
	if !event.HasImage() {
		return nil // No image to clean up
	}
	
	// Delete the main image and all its variants
	if err := s.deleteImageAndVariants(ctx, event.ImageKey); err != nil {
		return fmt.Errorf("failed to delete event images: %w", err)
	}
	
	log.Printf("Cleaned up images for event %d (key: %s)", eventID, event.ImageKey)
	return nil
}

// getImageKeysFromDatabase retrieves all image keys currently referenced in the database
func (s *ImageCleanupService) getImageKeysFromDatabase(ctx context.Context) (map[string]bool, error) {
	query := `
		SELECT DISTINCT image_key 
		FROM events 
		WHERE image_key IS NOT NULL AND image_key != ''
	`
	
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()
	
	imageKeys := make(map[string]bool)
	
	for rows.Next() {
		var imageKey string
		if err := rows.Scan(&imageKey); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		
		// Add the main key and all possible variant keys
		imageKeys[imageKey] = true
		
		// Add variant keys (thumbnail, medium, large, etc.)
		variants := []string{"thumbnail", "medium", "large", "thumbnail-webp", "medium-webp", "large-webp"}
		for _, variant := range variants {
			variantKey := fmt.Sprintf("%s/%s", imageKey, variant)
			imageKeys[variantKey] = true
		}
		
		// Add original key
		originalKey := fmt.Sprintf("%s/original", imageKey)
		imageKeys[originalKey] = true
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	
	return imageKeys, nil
}

// getImageKeysFromStorage retrieves all image keys from R2 storage
func (s *ImageCleanupService) getImageKeysFromStorage(ctx context.Context) ([]string, error) {
	// This is a simplified implementation
	// In a real implementation, you would list all objects in the R2 bucket
	// For now, we'll return an empty slice since the R2Service doesn't have a List method
	
	// TODO: Implement R2 object listing
	// This would require adding a ListObjects method to the StorageService interface
	
	return []string{}, nil
}

// findOrphanedImages compares database keys with storage keys to find orphans
func (s *ImageCleanupService) findOrphanedImages(dbKeys map[string]bool, storageKeys []string) []OrphanedImage {
	var orphaned []OrphanedImage
	
	for _, storageKey := range storageKeys {
		// Skip if this key is referenced in the database
		if dbKeys[storageKey] {
			continue
		}
		
		// Check if this might be a variant of a referenced image
		isVariant := false
		for dbKey := range dbKeys {
			if strings.HasPrefix(storageKey, dbKey+"/") {
				isVariant = true
				break
			}
		}
		
		if !isVariant {
			orphaned = append(orphaned, OrphanedImage{
				Key: storageKey,
				URL: s.storage.GetURL(storageKey),
				// Size and LastModified would need to be retrieved from storage metadata
			})
		}
	}
	
	return orphaned
}

// deleteImageAndVariants deletes an image and all its variants from storage
func (s *ImageCleanupService) deleteImageAndVariants(ctx context.Context, imageKey string) error {
	// Delete original
	originalKey := fmt.Sprintf("%s/original", imageKey)
	if err := s.storage.Delete(ctx, originalKey); err != nil {
		log.Printf("Warning: Failed to delete original image %s: %v", originalKey, err)
	}
	
	// Delete variants
	variants := []string{"thumbnail", "medium", "large", "thumbnail-webp", "medium-webp", "large-webp"}
	for _, variant := range variants {
		variantKey := fmt.Sprintf("%s/%s", imageKey, variant)
		if err := s.storage.Delete(ctx, variantKey); err != nil {
			log.Printf("Warning: Failed to delete variant %s: %v", variantKey, err)
		}
	}
	
	return nil
}

// ImageCleanupResult represents the result of an image cleanup operation
type ImageCleanupResult struct {
	TotalImagesInStorage int             `json:"total_images_in_storage"`
	TotalImagesInDB      int             `json:"total_images_in_db"`
	OrphanedImages       []OrphanedImage `json:"orphaned_images"`
	DryRun              bool            `json:"dry_run"`
	CleanedUp           []string        `json:"cleaned_up"`
	Errors              []string        `json:"errors"`
}

// GetSummary returns a summary of the cleanup result
func (r *ImageCleanupResult) GetSummary() string {
	if r.DryRun {
		return fmt.Sprintf("Dry run completed. Found %d orphaned images out of %d total images in storage.", 
			len(r.OrphanedImages), r.TotalImagesInStorage)
	}
	
	return fmt.Sprintf("Cleanup completed. Cleaned up %d images, %d errors occurred.", 
		len(r.CleanedUp), len(r.Errors))
}