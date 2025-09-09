package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestImageProcessingIntegration tests the complete image processing pipeline
// with real file operations (using temporary storage)
func TestImageProcessingIntegration(t *testing.T) {
	// Skip if running in CI without proper setup
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration tests")
	}

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "image_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a mock storage that saves to temp directory
	mockStorage := &FileSystemMockStorage{baseDir: tempDir}
	service := NewImageService(mockStorage)

	ctx := context.Background()

	t.Run("ProcessLargeJPEGImage", func(t *testing.T) {
		// Create a large test JPEG
		largeImage := createTestJPEG(1920, 1080)
		filename := "large-photo.jpg"

		options := ImageProcessingOptions{
			Quality:         85,
			EnableWebP:      true,
			CompressionLevel: 6,
			SupportedFormats: []string{"image/webp", "image/jpeg"},
		}

		reader := bytes.NewReader(largeImage)
		result, err := service.UploadImageWithOptions(ctx, reader, filename, options)

		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify original dimensions
		assert.Equal(t, 1920, result.Original.Width)
		assert.Equal(t, 1080, result.Original.Height)

		// Verify all variants were created
		expectedVariants := []string{"thumbnail", "medium", "large"}
		for _, expectedVariant := range expectedVariants {
			found := false
			for _, variant := range result.Variants {
				if variant.Name == expectedVariant || variant.Name == expectedVariant+"-webp" {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected variant %s not found", expectedVariant)
		}

		// Verify files were actually created
		for _, variant := range result.Variants {
			filePath := filepath.Join(tempDir, variant.Key)
			_, err := os.Stat(filePath)
			assert.NoError(t, err, "Variant file should exist: %s", variant.Key)
		}
	})

	t.Run("ProcessPNGWithTransparency", func(t *testing.T) {
		// Create a PNG with transparency
		pngImage := createTestPNG(400, 400)
		filename := "transparent-logo.png"

		options := ImageProcessingOptions{
			Quality:         90,
			EnableWebP:      false, // Disable WebP to test PNG preservation
			CompressionLevel: 9,
		}

		reader := bytes.NewReader(pngImage)
		result, err := service.UploadImageWithOptions(ctx, reader, filename, options)

		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify PNG format is preserved for original
		assert.Contains(t, result.Original.ContentType, "png")

		// Verify variants maintain reasonable quality
		for _, variant := range result.Variants {
			assert.True(t, variant.Width > 0)
			assert.True(t, variant.Height > 0)
			assert.NotEmpty(t, variant.Key)
		}
	})

	t.Run("OptimalFormatSelection", func(t *testing.T) {
		jpegImage := createTestJPEG(600, 400)
		filename := "photo.jpg"

		// Test with WebP support
		optionsWebP := ImageProcessingOptions{
			Quality:         80,
			EnableWebP:      true,
			SupportedFormats: []string{"image/webp", "image/jpeg"},
		}

		reader := bytes.NewReader(jpegImage)
		resultWebP, err := service.UploadImageWithOptions(ctx, reader, filename, optionsWebP)
		require.NoError(t, err)

		// Should have WebP variants
		webpVariants := 0
		for _, variant := range resultWebP.Variants {
			if strings.Contains(variant.Name, "webp") {
				webpVariants++
			}
		}
		assert.True(t, webpVariants > 0, "Should have WebP variants when enabled")

		// Test without WebP support
		optionsNoWebP := ImageProcessingOptions{
			Quality:    80,
			EnableWebP: false,
		}

		reader2 := bytes.NewReader(jpegImage)
		resultNoWebP, err := service.UploadImageWithOptions(ctx, reader2, filename+"_no_webp", optionsNoWebP)
		require.NoError(t, err)

		// Should not have WebP variants
		webpVariants = 0
		for _, variant := range resultNoWebP.Variants {
			if strings.Contains(variant.Name, "webp") {
				webpVariants++
			}
		}
		assert.Equal(t, 0, webpVariants, "Should not have WebP variants when disabled")
	})

	t.Run("ImageMetadataExtraction", func(t *testing.T) {
		// Test with different image types
		testCases := []struct {
			name     string
			imageGen func() []byte
			format   string
		}{
			{"JPEG", func() []byte { return createTestJPEG(300, 200) }, "jpeg"},
			{"PNG", func() []byte { return createTestPNG(300, 200) }, "png"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				imageData := tc.imageGen()
				reader := bytes.NewReader(imageData)

				result, err := service.UploadImage(ctx, reader, "test."+tc.format)
				require.NoError(t, err)

				// Verify metadata
				assert.Equal(t, 300, result.Original.Width)
				assert.Equal(t, 200, result.Original.Height)
				assert.True(t, result.Original.Size > 0)
				assert.Contains(t, result.Original.ContentType, tc.format)
				assert.False(t, result.Original.UploadedAt.IsZero())
			})
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with invalid image data
		invalidData := []byte("this is not an image")
		reader := bytes.NewReader(invalidData)

		result, err := service.UploadImage(ctx, reader, "invalid.jpg")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to decode image")
	})

	t.Run("ImageValidation", func(t *testing.T) {
		validImage := createTestJPEG(100, 100)
		
		// Test size validation
		err := service.ValidateImage(bytes.NewReader(validImage), 50) // Very small limit
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")

		// Test valid image
		err = service.ValidateImage(bytes.NewReader(validImage), 1024*1024) // 1MB limit
		assert.NoError(t, err)
	})

	t.Run("ImageCropping", func(t *testing.T) {
		// Create a larger test image for cropping
		largeImage := createTestJPEG(400, 300)
		filename := "crop-test.jpg"

		// Test valid crop
		options := ImageProcessingOptions{
			Quality:         85,
			EnableWebP:      false,
			CompressionLevel: 6,
			CropData: &CropData{
				X:      50,
				Y:      50,
				Width:  200,
				Height: 150,
			},
		}

		reader := bytes.NewReader(largeImage)
		result, err := service.UploadImageWithOptions(ctx, reader, filename, options)

		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify cropped dimensions
		assert.Equal(t, 200, result.Original.Width)
		assert.Equal(t, 150, result.Original.Height)

		// Verify variants are also properly sized
		for _, variant := range result.Variants {
			assert.True(t, variant.Width > 0)
			assert.True(t, variant.Height > 0)
			// Variants should maintain aspect ratio of cropped image
			aspectRatio := float64(variant.Width) / float64(variant.Height)
			expectedRatio := float64(200) / float64(150) // 4:3
			assert.InDelta(t, expectedRatio, aspectRatio, 0.1, "Variant should maintain aspect ratio")
		}
	})

	t.Run("InvalidCropData", func(t *testing.T) {
		testImage := createTestJPEG(100, 100)
		
		testCases := []struct {
			name     string
			cropData *CropData
			errorMsg string
		}{
			{
				name: "NegativeCoordinates",
				cropData: &CropData{
					X: -10, Y: 0, Width: 50, Height: 50,
				},
				errorMsg: "negative",
			},
			{
				name: "ExceedsBounds",
				cropData: &CropData{
					X: 0, Y: 0, Width: 200, Height: 200,
				},
				errorMsg: "exceeds",
			},
			{
				name: "ZeroDimensions",
				cropData: &CropData{
					X: 10, Y: 10, Width: 0, Height: 50,
				},
				errorMsg: "positive",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				options := ImageProcessingOptions{
					Quality:  85,
					CropData: tc.cropData,
				}

				reader := bytes.NewReader(testImage)
				result, err := service.UploadImageWithOptions(ctx, reader, "invalid-crop.jpg", options)

				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, strings.ToLower(err.Error()), tc.errorMsg)
			})
		}
	})

	t.Run("CropWithDifferentFormats", func(t *testing.T) {
		cropData := &CropData{
			X: 25, Y: 25, Width: 50, Height: 50,
		}

		testCases := []struct {
			name     string
			imageGen func() []byte
			format   string
		}{
			{"JPEG", func() []byte { return createTestJPEG(100, 100) }, "jpeg"},
			{"PNG", func() []byte { return createTestPNG(100, 100) }, "png"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				options := ImageProcessingOptions{
					Quality:  90,
					CropData: cropData,
				}

				imageData := tc.imageGen()
				reader := bytes.NewReader(imageData)

				result, err := service.UploadImageWithOptions(ctx, reader, "crop-test."+tc.format, options)
				require.NoError(t, err)

				// Verify cropped dimensions
				assert.Equal(t, 50, result.Original.Width)
				assert.Equal(t, 50, result.Original.Height)
			})
		}
	})
}

// FileSystemMockStorage implements StorageService for integration testing
type FileSystemMockStorage struct {
	baseDir string
}

func (fs *FileSystemMockStorage) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error) {
	// Create directory structure
	filePath := filepath.Join(fs.baseDir, key)
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Write file
	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return "", err
	}

	return "file://" + filePath, nil
}

func (fs *FileSystemMockStorage) Delete(ctx context.Context, key string) error {
	filePath := filepath.Join(fs.baseDir, key)
	return os.Remove(filePath)
}

func (fs *FileSystemMockStorage) GetURL(key string) string {
	return "file://" + filepath.Join(fs.baseDir, key)
}

func (fs *FileSystemMockStorage) GeneratePresignedURL(ctx context.Context, key string, contentType string, expiration time.Duration) (string, error) {
	return fs.GetURL(key), nil
}

func (fs *FileSystemMockStorage) Exists(ctx context.Context, key string) (bool, error) {
	filePath := filepath.Join(fs.baseDir, key)
	_, err := os.Stat(filePath)
	return err == nil, nil
}

// Benchmark tests for performance
func BenchmarkImageProcessing(b *testing.B) {
	mockStorage := &MockStorageService{}
	service := NewImageService(mockStorage)
	ctx := context.Background()

	// Mock storage calls
	mockStorage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int64")).
		Return("https://example.com/image.jpg", nil).Maybe()

	b.Run("ProcessSmallImage", func(b *testing.B) {
		smallImage := createTestJPEG(200, 200)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			reader := bytes.NewReader(smallImage)
			_, err := service.UploadImage(ctx, reader, "small.jpg")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ProcessLargeImage", func(b *testing.B) {
		largeImage := createTestJPEG(1920, 1080)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			reader := bytes.NewReader(largeImage)
			_, err := service.UploadImage(ctx, reader, "large.jpg")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ProcessWithWebP", func(b *testing.B) {
		image := createTestJPEG(800, 600)
		options := ImageProcessingOptions{
			Quality:    85,
			EnableWebP: true,
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			reader := bytes.NewReader(image)
			_, err := service.UploadImageWithOptions(ctx, reader, "webp.jpg", options)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Test concurrent image processing
func TestConcurrentImageProcessing(t *testing.T) {
	mockStorage := &MockStorageService{}
	service := NewImageService(mockStorage)
	ctx := context.Background()

	// Mock storage calls
	mockStorage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int64")).
		Return("https://example.com/image.jpg", nil).Maybe()

	const numGoroutines = 10
	const imagesPerGoroutine = 5

	results := make(chan error, numGoroutines*imagesPerGoroutine)

	// Start concurrent processing
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < imagesPerGoroutine; j++ {
				testImage := createTestJPEG(300, 300)
				reader := bytes.NewReader(testImage)
				filename := fmt.Sprintf("concurrent-%d-%d.jpg", goroutineID, j)
				
				_, err := service.UploadImage(ctx, reader, filename)
				results <- err
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines*imagesPerGoroutine; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent image processing should not fail")
	}
}