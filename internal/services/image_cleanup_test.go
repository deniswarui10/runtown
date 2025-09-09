package services

import (
	"context"
	"database/sql"
	"io"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"
)

// MockCleanupStorageService for testing image cleanup
type MockCleanupStorageService struct {
	files map[string]bool
	deleteErrors map[string]error
}

func NewMockCleanupStorageService() *MockCleanupStorageService {
	return &MockCleanupStorageService{
		files: make(map[string]bool),
		deleteErrors: make(map[string]error),
	}
}

func (m *MockCleanupStorageService) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error) {
	m.files[key] = true
	return "https://example.com/" + key, nil
}

func (m *MockCleanupStorageService) Delete(ctx context.Context, key string) error {
	if err, exists := m.deleteErrors[key]; exists {
		return err
	}
	delete(m.files, key)
	return nil
}

func (m *MockCleanupStorageService) GetURL(key string) string {
	return "https://example.com/" + key
}

func (m *MockCleanupStorageService) GeneratePresignedURL(ctx context.Context, key string, contentType string, expiration time.Duration) (string, error) {
	return "https://example.com/presigned/" + key, nil
}

func (m *MockCleanupStorageService) Exists(ctx context.Context, key string) (bool, error) {
	return m.files[key], nil
}

func (m *MockCleanupStorageService) AddFile(key string) {
	m.files[key] = true
}

func (m *MockCleanupStorageService) SetDeleteError(key string, err error) {
	m.deleteErrors[key] = err
}

// MockEventRepository for testing
type MockEventRepository struct {
	events map[int]*models.Event
}

func NewMockEventRepository() *MockEventRepository {
	return &MockEventRepository{
		events: make(map[int]*models.Event),
	}
}

func (m *MockEventRepository) GetByID(id int) (*models.Event, error) {
	if event, exists := m.events[id]; exists {
		return event, nil
	}
	return nil, sql.ErrNoRows
}

func (m *MockEventRepository) AddEvent(event *models.Event) {
	m.events[event.ID] = event
}

func TestImageCleanupService_CleanupEventImages(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	storage := NewMockCleanupStorageService()
	eventRepo := NewMockEventRepository()
	
	// Add test files to storage
	storage.AddFile("events/2024/01/01/test-image-12345678/original.jpeg")
	storage.AddFile("events/2024/01/01/test-image-12345678/thumbnail.jpeg")
	storage.AddFile("events/2024/01/01/test-image-12345678/medium.jpeg")
	storage.AddFile("events/2024/01/01/test-image-12345678/large.jpeg")
	
	// Add test event
	uploadTime := time.Now()
	event := &models.Event{
		ID:              1,
		Title:           "Test Event",
		ImageURL:        "https://example.com/events/2024/01/01/test-image-12345678/original.jpeg",
		ImageKey:        "events/2024/01/01/test-image-12345678",
		ImageSize:       1024000,
		ImageFormat:     "jpeg",
		ImageWidth:      800,
		ImageHeight:     600,
		ImageUploadedAt: &uploadTime,
	}
	eventRepo.AddEvent(event)
	
	// Create cleanup service
	service := NewImageCleanupService(storage, eventRepo, nil)
	
	// Test cleanup
	err := service.CleanupEventImages(ctx, 1)
	if err != nil {
		t.Errorf("CleanupEventImages() error = %v", err)
	}
	
	// Verify files were deleted
	expectedDeletedFiles := []string{
		"events/2024/01/01/test-image-12345678/original",
		"events/2024/01/01/test-image-12345678/thumbnail",
		"events/2024/01/01/test-image-12345678/medium",
		"events/2024/01/01/test-image-12345678/large",
		"events/2024/01/01/test-image-12345678/thumbnail-webp",
		"events/2024/01/01/test-image-12345678/medium-webp",
		"events/2024/01/01/test-image-12345678/large-webp",
	}
	
	for _, file := range expectedDeletedFiles {
		if storage.files[file] {
			t.Errorf("File %s was not deleted", file)
		}
	}
}

func TestImageCleanupService_CleanupEventImages_NoImage(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	storage := NewMockCleanupStorageService()
	eventRepo := NewMockEventRepository()
	
	// Add test event without image
	event := &models.Event{
		ID:       1,
		Title:    "Test Event",
		ImageURL: "",
		ImageKey: "",
	}
	eventRepo.AddEvent(event)
	
	// Create cleanup service
	service := NewImageCleanupService(storage, eventRepo, nil)
	
	// Test cleanup - should not error
	err := service.CleanupEventImages(ctx, 1)
	if err != nil {
		t.Errorf("CleanupEventImages() error = %v", err)
	}
}

func TestImageCleanupService_CleanupEventImages_EventNotFound(t *testing.T) {
	ctx := context.Background()
	
	// Setup mocks
	storage := NewMockCleanupStorageService()
	eventRepo := NewMockEventRepository()
	
	// Create cleanup service
	service := NewImageCleanupService(storage, eventRepo, nil)
	
	// Test cleanup for non-existent event
	err := service.CleanupEventImages(ctx, 999)
	if err == nil {
		t.Error("CleanupEventImages() expected error for non-existent event")
	}
}

func TestImageCleanupService_findOrphanedImages(t *testing.T) {
	storage := NewMockCleanupStorageService()
	service := NewImageCleanupService(storage, nil, nil)
	
	// Database has these image keys
	dbKeys := map[string]bool{
		"events/2024/01/01/image1-12345678": true,
		"events/2024/01/01/image1-12345678/original": true,
		"events/2024/01/01/image1-12345678/thumbnail": true,
		"events/2024/01/01/image1-12345678/medium": true,
		"events/2024/01/01/image1-12345678/large": true,
		"events/2024/01/02/image2-87654321": true,
		"events/2024/01/02/image2-87654321/original": true,
		"events/2024/01/02/image2-87654321/thumbnail": true,
		"events/2024/01/02/image2-87654321/medium": true,
		"events/2024/01/02/image2-87654321/large": true,
	}
	
	// Storage has these files
	storageKeys := []string{
		"events/2024/01/01/image1-12345678/original",     // Referenced
		"events/2024/01/01/image1-12345678/thumbnail",    // Referenced
		"events/2024/01/01/image1-12345678/medium",       // Referenced
		"events/2024/01/02/image2-87654321/original",     // Referenced
		"events/2024/01/03/orphaned-image-11111111/original", // Orphaned
		"events/2024/01/03/orphaned-image-11111111/thumbnail", // Orphaned
		"events/2024/01/04/another-orphan-22222222/medium",    // Orphaned
	}
	
	orphaned := service.findOrphanedImages(dbKeys, storageKeys)
	
	expectedOrphaned := 3 // 3 orphaned files
	if len(orphaned) != expectedOrphaned {
		t.Errorf("findOrphanedImages() found %d orphaned images, expected %d", len(orphaned), expectedOrphaned)
	}
	
	// Check that the orphaned images are the expected ones
	orphanedKeys := make(map[string]bool)
	for _, img := range orphaned {
		orphanedKeys[img.Key] = true
	}
	
	expectedOrphanedKeys := []string{
		"events/2024/01/03/orphaned-image-11111111/original",
		"events/2024/01/03/orphaned-image-11111111/thumbnail",
		"events/2024/01/04/another-orphan-22222222/medium",
	}
	
	for _, key := range expectedOrphanedKeys {
		if !orphanedKeys[key] {
			t.Errorf("Expected orphaned key %s not found", key)
		}
	}
}

func TestImageCleanupResult_GetSummary(t *testing.T) {
	tests := []struct {
		name     string
		result   ImageCleanupResult
		expected string
	}{
		{
			name: "dry run with orphaned images",
			result: ImageCleanupResult{
				TotalImagesInStorage: 10,
				OrphanedImages:       make([]OrphanedImage, 3),
				DryRun:              true,
			},
			expected: "Dry run completed. Found 3 orphaned images out of 10 total images in storage.",
		},
		{
			name: "actual cleanup with success",
			result: ImageCleanupResult{
				CleanedUp: []string{"image1", "image2"},
				Errors:    []string{},
				DryRun:    false,
			},
			expected: "Cleanup completed. Cleaned up 2 images, 0 errors occurred.",
		},
		{
			name: "actual cleanup with errors",
			result: ImageCleanupResult{
				CleanedUp: []string{"image1"},
				Errors:    []string{"Failed to delete image2", "Failed to delete image3"},
				DryRun:    false,
			},
			expected: "Cleanup completed. Cleaned up 1 images, 2 errors occurred.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result.GetSummary()
			if result != tt.expected {
				t.Errorf("GetSummary() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestImageCleanupService_deleteImageAndVariants(t *testing.T) {
	ctx := context.Background()
	
	// Setup mock storage
	storage := NewMockCleanupStorageService()
	service := NewImageCleanupService(storage, nil, nil)
	
	// Add files to storage
	imageKey := "events/2024/01/01/test-image-12345678"
	filesToAdd := []string{
		imageKey + "/original",
		imageKey + "/thumbnail",
		imageKey + "/medium",
		imageKey + "/large",
		imageKey + "/thumbnail-webp",
		imageKey + "/medium-webp",
		imageKey + "/large-webp",
	}
	
	for _, file := range filesToAdd {
		storage.AddFile(file)
	}
	
	// Test deletion
	err := service.deleteImageAndVariants(ctx, imageKey)
	if err != nil {
		t.Errorf("deleteImageAndVariants() error = %v", err)
	}
	
	// Verify all files were deleted
	for _, file := range filesToAdd {
		if storage.files[file] {
			t.Errorf("File %s was not deleted", file)
		}
	}
}

func TestOrphanedImage_Structure(t *testing.T) {
	// Test that OrphanedImage struct has expected fields
	lastModified := time.Now()
	
	orphaned := OrphanedImage{
		Key:          "test-key",
		URL:          "https://example.com/test-key",
		Size:         1024,
		LastModified: lastModified,
	}
	
	if orphaned.Key != "test-key" {
		t.Errorf("OrphanedImage.Key = %v, expected test-key", orphaned.Key)
	}
	if orphaned.URL != "https://example.com/test-key" {
		t.Errorf("OrphanedImage.URL = %v, expected https://example.com/test-key", orphaned.URL)
	}
	if orphaned.Size != 1024 {
		t.Errorf("OrphanedImage.Size = %v, expected 1024", orphaned.Size)
	}
	if orphaned.LastModified != lastModified {
		t.Errorf("OrphanedImage.LastModified = %v, expected %v", orphaned.LastModified, lastModified)
	}
}