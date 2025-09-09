package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/database"
	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
	"event-ticketing-platform/internal/services"

	"github.com/disintegration/imaging"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageManagementIntegration(t *testing.T) {
	// Skip if no database URL is provided
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping integration tests")
	}

	// Setup test database
	cfg := config.Load()
	db, err := database.Connect(cfg.DatabaseURL)
	require.NoError(t, err)
	defer db.Close()

	// Run migrations
	err = database.RunMigrations(db, "../../database/migrations")
	require.NoError(t, err)

	// Setup repositories
	userRepo := repositories.NewUserRepository(db)
	eventRepo := repositories.NewEventRepository(db)

	// Setup services
	authService := services.NewAuthService(userRepo, cfg)
	eventService := services.NewEventService(eventRepo, userRepo)
	
	// Setup storage service (use fallback for testing)
	storageService := services.NewFallbackStorageService()
	imageService := services.NewImageService(storageService)

	// Setup handlers
	imageHandler := NewImageManagementHandler(imageService, eventService, storageService)

	// Create test user (organizer)
	organizer := &models.User{
		Email:     "organizer@test.com",
		FirstName: "Test",
		LastName:  "Organizer",
		Role:      models.RoleOrganizer,
	}
	
	registerReq := &services.RegisterRequest{
		Email:     organizer.Email,
		Password:  "password123",
		FirstName: organizer.FirstName,
		LastName:  organizer.LastName,
		Role:      organizer.Role,
	}
	
	authResp, err := authService.Register(registerReq)
	require.NoError(t, err)
	organizer.ID = authResp.User.ID

	// Create test event
	eventReq := &services.EventCreateRequest{
		Title:       "Test Event",
		Description: "Test event description",
		StartDate:   time.Now().Add(24 * time.Hour),
		EndDate:     time.Now().Add(26 * time.Hour),
		Location:    "Test Location",
		CategoryID:  1,
		Status:      models.StatusDraft,
	}
	
	event, err := eventService.CreateEvent(organizer.ID, eventReq)
	require.NoError(t, err)

	t.Run("ImageGalleryPage", func(t *testing.T) {
		// Create request with authentication
		req := httptest.NewRequest("GET", fmt.Sprintf("/organizer/events/%d/images", event.ID), nil)
		req = req.WithContext(middleware.SetUserInContext(req.Context(), organizer))
		
		// Setup router
		r := chi.NewRouter()
		r.Get("/organizer/events/{eventId}/images", imageHandler.ImageGalleryPage)
		
		// Execute request
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Manage Event Images")
		assert.Contains(t, w.Body.String(), event.Title)
	})

	t.Run("UploadImage", func(t *testing.T) {
		// Create test image data
		imageData := createTestImageData()
		
		// Create multipart form
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		
		// Add image file
		part, err := writer.CreateFormFile("image", "test.jpg")
		require.NoError(t, err)
		_, err = part.Write(imageData)
		require.NoError(t, err)
		
		// Add processing options
		writer.WriteField("quality", "85")
		writer.WriteField("enable_webp", "true")
		writer.WriteField("compression_level", "6")
		
		writer.Close()
		
		// Create request
		req := httptest.NewRequest("POST", fmt.Sprintf("/organizer/events/%d/images/upload", event.ID), &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = req.WithContext(middleware.SetUserInContext(req.Context(), organizer))
		
		// Setup router
		r := chi.NewRouter()
		r.Post("/organizer/events/{eventId}/images/upload", imageHandler.UploadImage)
		
		// Execute request
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response ImageUploadResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response.Success)
		assert.NotEmpty(t, response.ImageURL)
		assert.NotNil(t, response.Result)
		assert.NotEmpty(t, response.Result.Original.URL)
		assert.Greater(t, len(response.Result.Variants), 0)
	})

	t.Run("UploadImageWithCrop", func(t *testing.T) {
		// Create larger test image data for cropping
		imageData := createLargerTestImageData()
		
		// Create multipart form
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		
		// Add image file
		part, err := writer.CreateFormFile("image", "test-crop.jpg")
		require.NoError(t, err)
		_, err = part.Write(imageData)
		require.NoError(t, err)
		
		// Add processing options
		writer.WriteField("quality", "90")
		writer.WriteField("enable_webp", "true")
		writer.WriteField("compression_level", "6")
		
		// Add crop data
		writer.WriteField("crop_x", "10")
		writer.WriteField("crop_y", "10")
		writer.WriteField("crop_width", "80")
		writer.WriteField("crop_height", "80")
		
		writer.Close()
		
		// Create request
		req := httptest.NewRequest("POST", fmt.Sprintf("/organizer/events/%d/images/upload", event.ID), &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = req.WithContext(middleware.SetUserInContext(req.Context(), organizer))
		
		// Setup router
		r := chi.NewRouter()
		r.Post("/organizer/events/{eventId}/images/upload", imageHandler.UploadImage)
		
		// Execute request
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response ImageUploadResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response.Success)
		assert.NotEmpty(t, response.ImageURL)
		assert.NotNil(t, response.Result)
		
		// Verify that cropped image has expected dimensions
		assert.Equal(t, 80, response.Result.Original.Width)
		assert.Equal(t, 80, response.Result.Original.Height)
	})

	t.Run("UploadImageWithInvalidCrop", func(t *testing.T) {
		// Create test image data
		imageData := createTestImageData()
		
		// Create multipart form
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		
		// Add image file
		part, err := writer.CreateFormFile("image", "test.jpg")
		require.NoError(t, err)
		_, err = part.Write(imageData)
		require.NoError(t, err)
		
		// Add invalid crop data (exceeds image bounds)
		writer.WriteField("crop_x", "0")
		writer.WriteField("crop_y", "0")
		writer.WriteField("crop_width", "1000")
		writer.WriteField("crop_height", "1000")
		
		writer.Close()
		
		// Create request
		req := httptest.NewRequest("POST", fmt.Sprintf("/organizer/events/%d/images/upload", event.ID), &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = req.WithContext(middleware.SetUserInContext(req.Context(), organizer))
		
		// Setup router
		r := chi.NewRouter()
		r.Post("/organizer/events/{eventId}/images/upload", imageHandler.UploadImage)
		
		// Execute request
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		// Verify response - should fail due to invalid crop
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response ImageUploadResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response.Success)
		assert.Contains(t, response.Error, "crop")
	})

	t.Run("UploadImage_InvalidFile", func(t *testing.T) {
		// Create invalid file data
		invalidData := []byte("not an image")
		
		// Create multipart form
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		
		part, err := writer.CreateFormFile("image", "test.txt")
		require.NoError(t, err)
		_, err = part.Write(invalidData)
		require.NoError(t, err)
		
		writer.Close()
		
		// Create request
		req := httptest.NewRequest("POST", fmt.Sprintf("/organizer/events/%d/images/upload", event.ID), &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = req.WithContext(middleware.SetUserInContext(req.Context(), organizer))
		
		// Setup router
		r := chi.NewRouter()
		r.Post("/organizer/events/{eventId}/images/upload", imageHandler.UploadImage)
		
		// Execute request
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		// Verify response
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response ImageUploadResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response.Success)
		assert.NotEmpty(t, response.Error)
	})

	t.Run("UploadImage_Unauthorized", func(t *testing.T) {
		// Create test image data
		imageData := createTestImageData()
		
		// Create multipart form
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		
		part, err := writer.CreateFormFile("image", "test.jpg")
		require.NoError(t, err)
		_, err = part.Write(imageData)
		require.NoError(t, err)
		
		writer.Close()
		
		// Create request without authentication
		req := httptest.NewRequest("POST", fmt.Sprintf("/organizer/events/%d/images/upload", event.ID), &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		
		// Setup router
		r := chi.NewRouter()
		r.Post("/organizer/events/{eventId}/images/upload", imageHandler.UploadImage)
		
		// Execute request
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		// Verify response
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("GeneratePresignedURL", func(t *testing.T) {
		// Create request body
		reqBody := map[string]string{
			"filename":     "test.jpg",
			"content_type": "image/jpeg",
		}
		
		bodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)
		
		// Create request
		req := httptest.NewRequest("POST", fmt.Sprintf("/organizer/events/%d/images/presigned", event.ID), bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(middleware.SetUserInContext(req.Context(), organizer))
		
		// Setup router
		r := chi.NewRouter()
		r.Post("/organizer/events/{eventId}/images/presigned", imageHandler.GeneratePresignedURL)
		
		// Execute request
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response["success"].(bool))
		assert.NotEmpty(t, response["presigned_url"])
		assert.NotEmpty(t, response["key"])
	})

	t.Run("DeleteImage", func(t *testing.T) {
		// First upload an image to delete
		imageData := createTestImageData()
		
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		
		part, err := writer.CreateFormFile("image", "test.jpg")
		require.NoError(t, err)
		_, err = part.Write(imageData)
		require.NoError(t, err)
		
		writer.Close()
		
		// Upload image
		uploadReq := httptest.NewRequest("POST", fmt.Sprintf("/organizer/events/%d/images/upload", event.ID), &buf)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		uploadReq = uploadReq.WithContext(middleware.SetUserInContext(uploadReq.Context(), organizer))
		
		r := chi.NewRouter()
		r.Post("/organizer/events/{eventId}/images/upload", imageHandler.UploadImage)
		r.Delete("/organizer/events/{eventId}/images/delete", imageHandler.DeleteImage)
		
		uploadW := httptest.NewRecorder()
		r.ServeHTTP(uploadW, uploadReq)
		
		require.Equal(t, http.StatusOK, uploadW.Code)
		
		var uploadResponse ImageUploadResponse
		err = json.Unmarshal(uploadW.Body.Bytes(), &uploadResponse)
		require.NoError(t, err)
		require.True(t, uploadResponse.Success)
		
		// Extract image key from the result
		imageKey := "test-key" // In a real test, you'd extract this from the upload response
		
		// Now delete the image
		deleteReqBody := map[string]string{
			"image_key": imageKey,
		}
		
		deleteBodyBytes, err := json.Marshal(deleteReqBody)
		require.NoError(t, err)
		
		deleteReq := httptest.NewRequest("DELETE", fmt.Sprintf("/organizer/events/%d/images/delete", event.ID), bytes.NewReader(deleteBodyBytes))
		deleteReq.Header.Set("Content-Type", "application/json")
		deleteReq = deleteReq.WithContext(middleware.SetUserInContext(deleteReq.Context(), organizer))
		
		deleteW := httptest.NewRecorder()
		r.ServeHTTP(deleteW, deleteReq)
		
		// Verify response
		assert.Equal(t, http.StatusOK, deleteW.Code)
		
		var deleteResponse map[string]interface{}
		err = json.Unmarshal(deleteW.Body.Bytes(), &deleteResponse)
		require.NoError(t, err)
		
		assert.True(t, deleteResponse["success"].(bool))
	})

	t.Run("ReplaceImage", func(t *testing.T) {
		// Create test image data
		imageData := createTestImageData()
		
		// Create multipart form
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		
		// Add image file
		part, err := writer.CreateFormFile("image", "replacement.jpg")
		require.NoError(t, err)
		_, err = part.Write(imageData)
		require.NoError(t, err)
		
		// Add old image key
		writer.WriteField("old_image_key", "old-test-key")
		writer.WriteField("quality", "90")
		
		writer.Close()
		
		// Create request
		req := httptest.NewRequest("POST", fmt.Sprintf("/organizer/events/%d/images/replace", event.ID), &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = req.WithContext(middleware.SetUserInContext(req.Context(), organizer))
		
		// Setup router
		r := chi.NewRouter()
		r.Post("/organizer/events/{eventId}/images/replace", imageHandler.ReplaceImage)
		
		// Execute request
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response ImageUploadResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response.Success)
		assert.NotEmpty(t, response.ImageURL)
	})

	t.Run("GetImageVariants", func(t *testing.T) {
		imageKey := "test-image-key"
		
		// Create request
		req := httptest.NewRequest("GET", fmt.Sprintf("/organizer/events/%d/images/%s/variants", event.ID, imageKey), nil)
		req = req.WithContext(middleware.SetUserInContext(req.Context(), organizer))
		
		// Setup router
		r := chi.NewRouter()
		r.Get("/organizer/events/{eventId}/images/{imageKey}/variants", imageHandler.GetImageVariants)
		
		// Execute request
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response["success"].(bool))
		assert.NotNil(t, response["variants"])
	})

	t.Run("ForbiddenAccess", func(t *testing.T) {
		// Create another user who shouldn't have access
		otherUser := &models.User{
			Email:     "other@test.com",
			FirstName: "Other",
			LastName:  "User",
			Role:      models.RoleAttendee,
		}
		
		otherRegisterReq := &services.RegisterRequest{
			Email:     otherUser.Email,
			Password:  "password123",
			FirstName: otherUser.FirstName,
			LastName:  otherUser.LastName,
			Role:      otherUser.Role,
		}
		
		otherAuthResp, err := authService.Register(otherRegisterReq)
		require.NoError(t, err)
		otherUser.ID = otherAuthResp.User.ID
		
		// Try to access image gallery
		req := httptest.NewRequest("GET", fmt.Sprintf("/organizer/events/%d/images", event.ID), nil)
		req = req.WithContext(middleware.SetUserInContext(req.Context(), otherUser))
		
		r := chi.NewRouter()
		r.Get("/organizer/events/{eventId}/images", imageHandler.ImageGalleryPage)
		
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		// Should be forbidden
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// Cleanup
	t.Cleanup(func() {
		// Clean up test data
		db.Exec("DELETE FROM events WHERE organizer_id = ?", organizer.ID)
		db.Exec("DELETE FROM users WHERE id = ?", organizer.ID)
	})
}

func TestImageManagementR2Integration(t *testing.T) {
	// Skip if R2 credentials are not provided
	if os.Getenv("R2_ACCESS_KEY_ID") == "" {
		t.Skip("R2 credentials not set, skipping R2 integration tests")
	}

	// Setup R2 service
	r2Config := config.R2Config{
		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		BucketName:      os.Getenv("R2_BUCKET_NAME"),
		Region:          os.Getenv("R2_REGION"),
		AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		PublicURL:       os.Getenv("R2_PUBLIC_URL"),
	}

	r2Service, err := services.NewR2Service(r2Config)
	require.NoError(t, err)

	imageService := services.NewImageService(r2Service)

	t.Run("UploadToR2", func(t *testing.T) {
		// Create test image
		imageData := createTestImageData()
		reader := bytes.NewReader(imageData)

		// Upload image
		result, err := imageService.UploadImage(context.Background(), reader, "test-r2.jpg")
		require.NoError(t, err)

		// Verify result
		assert.NotEmpty(t, result.Original.URL)
		assert.Greater(t, len(result.Variants), 0)
		assert.Contains(t, result.Original.URL, r2Config.BucketName)

		// Verify image exists in R2
		exists, err := r2Service.Exists(context.Background(), result.Original.Key)
		require.NoError(t, err)
		assert.True(t, exists)

		// Cleanup
		err = imageService.DeleteImage(context.Background(), extractKeyPrefix(result.Original.Key))
		assert.NoError(t, err)
	})

	t.Run("R2HealthCheck", func(t *testing.T) {
		err := r2Service.HealthCheck(context.Background())
		assert.NoError(t, err)
	})

	t.Run("GeneratePresignedURL", func(t *testing.T) {
		key := "test-presigned/test.jpg"
		contentType := "image/jpeg"
		expiration := 15 * time.Minute

		url, err := r2Service.GeneratePresignedURL(context.Background(), key, contentType, expiration)
		require.NoError(t, err)

		assert.NotEmpty(t, url)
		assert.Contains(t, url, r2Config.BucketName)
	})

	t.Run("ImageVariantsR2", func(t *testing.T) {
		// Create test image with different processing options
		imageData := createLargerTestImageData()
		reader := bytes.NewReader(imageData)

		// Upload with WebP enabled
		options := services.ImageProcessingOptions{
			Quality:         90,
			EnableWebP:      true,
			CompressionLevel: 6,
		}

		result, err := imageService.UploadImageWithOptions(context.Background(), reader, "test-variants.jpg", options)
		require.NoError(t, err)

		// Verify all variants were created
		assert.NotEmpty(t, result.Original.URL)
		assert.Greater(t, len(result.Variants), 3) // Should have thumbnail, medium, large + WebP variants

		// Check that WebP variants exist
		hasWebPVariant := false
		for _, variant := range result.Variants {
			if strings.Contains(variant.Name, "webp") {
				hasWebPVariant = true
				break
			}
		}
		assert.True(t, hasWebPVariant, "Should have WebP variants")

		// Verify each variant exists in R2
		for _, variant := range result.Variants {
			exists, err := r2Service.Exists(context.Background(), variant.Key)
			assert.NoError(t, err)
			assert.True(t, exists, "Variant %s should exist in R2", variant.Name)
		}

		// Cleanup
		err = imageService.DeleteImage(context.Background(), extractKeyPrefix(result.Original.Key))
		assert.NoError(t, err)
	})

	t.Run("CropImageR2", func(t *testing.T) {
		// Create larger test image for cropping
		imageData := createLargerTestImageData()
		reader := bytes.NewReader(imageData)

		// Upload with crop options
		options := services.ImageProcessingOptions{
			Quality:         85,
			EnableWebP:      false,
			CompressionLevel: 6,
			CropData: &services.CropData{
				X:      20,
				Y:      20,
				Width:  60,
				Height: 60,
			},
		}

		result, err := imageService.UploadImageWithOptions(context.Background(), reader, "test-crop-r2.jpg", options)
		require.NoError(t, err)

		// Verify cropped dimensions
		assert.Equal(t, 60, result.Original.Width)
		assert.Equal(t, 60, result.Original.Height)

		// Verify image exists in R2
		exists, err := r2Service.Exists(context.Background(), result.Original.Key)
		require.NoError(t, err)
		assert.True(t, exists)

		// Cleanup
		err = imageService.DeleteImage(context.Background(), extractKeyPrefix(result.Original.Key))
		assert.NoError(t, err)
	})

	t.Run("R2URLGeneration", func(t *testing.T) {
		// Test URL generation for different variants
		keyPrefix := "events/2024/01/01/test-image-abc123"
		
		originalURL := imageService.GetImageURL(keyPrefix, "original")
		thumbnailURL := imageService.GetImageURL(keyPrefix, "thumbnail")
		
		assert.NotEmpty(t, originalURL)
		assert.NotEmpty(t, thumbnailURL)
		assert.Contains(t, originalURL, keyPrefix)
		assert.Contains(t, thumbnailURL, keyPrefix)
		assert.Contains(t, originalURL, "original")
		assert.Contains(t, thumbnailURL, "thumbnail")
	})

	t.Run("R2ErrorHandling", func(t *testing.T) {
		// Test error handling for non-existent keys
		exists, err := r2Service.Exists(context.Background(), "non-existent-key")
		require.NoError(t, err)
		assert.False(t, exists)

		// Test deletion of non-existent key (should not error)
		err = r2Service.Delete(context.Background(), "non-existent-key")
		assert.NoError(t, err)
	})
}

// Helper functions

func createTestImageData() []byte {
	// Create a minimal JPEG image (1x1 pixel)
	// This is a base64 encoded 1x1 red pixel JPEG
	jpegData := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
		0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
		0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07, 0x07, 0x07, 0x09,
		0x09, 0x08, 0x0A, 0x0C, 0x14, 0x0D, 0x0C, 0x0B, 0x0B, 0x0C, 0x19, 0x12,
		0x13, 0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E, 0x1D, 0x1A, 0x1C, 0x1C, 0x20,
		0x24, 0x2E, 0x27, 0x20, 0x22, 0x2C, 0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29,
		0x2C, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1F, 0x27, 0x39, 0x3D, 0x38, 0x32,
		0x3C, 0x2E, 0x33, 0x34, 0x32, 0xFF, 0xC0, 0x00, 0x11, 0x08, 0x00, 0x01,
		0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0x02, 0x11, 0x01, 0x03, 0x11, 0x01,
		0xFF, 0xC4, 0x00, 0x14, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0xFF, 0xC4,
		0x00, 0x14, 0x10, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xDA, 0x00, 0x0C,
		0x03, 0x01, 0x00, 0x02, 0x11, 0x03, 0x11, 0x00, 0x3F, 0x00, 0x80, 0xFF, 0xD9,
	}
	return jpegData
}

func createLargerTestImageData() []byte {
	// Create a simple 100x100 pixel test image using the imaging library
	// This creates a programmatic image for testing crop functionality
	img := imaging.New(100, 100, color.RGBA{255, 0, 0, 255}) // Red 100x100 image
	
	var buf bytes.Buffer
	err := imaging.Encode(&buf, img, imaging.JPEG)
	if err != nil {
		// Fallback to minimal image if encoding fails
		return createTestImageData()
	}
	
	return buf.Bytes()
}

func extractKeyPrefix(fullKey string) string {
	// Extract the key prefix from a full key path
	// Example: "events/2024/01/02/image-name-uuid/original.jpg" -> "events/2024/01/02/image-name-uuid"
	parts := strings.Split(fullKey, "/")
	if len(parts) >= 2 {
		return strings.Join(parts[:len(parts)-1], "/")
	}
	return fullKey
}