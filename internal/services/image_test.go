package services

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockStorageService is a mock implementation of StorageService
type MockStorageService struct {
	mock.Mock
}

func (m *MockStorageService) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error) {
	args := m.Called(ctx, key, reader, contentType, size)
	return args.String(0), args.Error(1)
}

func (m *MockStorageService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorageService) GetURL(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func (m *MockStorageService) GeneratePresignedURL(ctx context.Context, key string, contentType string, expiration time.Duration) (string, error) {
	args := m.Called(ctx, key, contentType, expiration)
	return args.String(0), args.Error(1)
}

func (m *MockStorageService) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

// Helper function to create a test JPEG image
func createTestJPEG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fill with a simple pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, image.NewRGBA(image.Rect(0, 0, 1, 1)).At(0, 0))
		}
	}
	
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

// Helper function to create a test PNG image
func createTestPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func TestNewImageService(t *testing.T) {
	mockStorage := &MockStorageService{}
	service := NewImageService(mockStorage)
	
	assert.NotNil(t, service)
	assert.Equal(t, mockStorage, service.storage)
}

func TestImageService_ValidateImage(t *testing.T) {
	service := NewImageService(&MockStorageService{})
	
	tests := []struct {
		name      string
		imageData []byte
		maxSize   int64
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid JPEG",
			imageData: createTestJPEG(100, 100),
			maxSize:   1024 * 1024, // 1MB
			wantErr:   false,
		},
		{
			name:      "valid PNG",
			imageData: createTestPNG(100, 100),
			maxSize:   1024 * 1024, // 1MB
			wantErr:   false,
		},
		{
			name:      "file too large",
			imageData: createTestJPEG(100, 100),
			maxSize:   100, // Very small limit
			wantErr:   true,
			errMsg:    "exceeds maximum allowed size",
		},
		{
			name:      "invalid image data",
			imageData: []byte("not an image"),
			maxSize:   1024 * 1024,
			wantErr:   true,
			errMsg:    "invalid image format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.imageData)
			err := service.ValidateImage(reader, tt.maxSize)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestImageService_UploadImage(t *testing.T) {
	mockStorage := &MockStorageService{}
	service := NewImageService(mockStorage)
	
	ctx := context.Background()
	testImage := createTestJPEG(200, 200)
	filename := "test-image.jpg"

	// Mock storage calls - original + 3 variants + 3 WebP variants = 7 calls
	mockStorage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int64")).
		Return("https://example.com/image.jpg", nil).Times(7)

	reader := bytes.NewReader(testImage)
	result, err := service.UploadImage(ctx, reader, filename)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Check original metadata
	assert.NotEmpty(t, result.Original.Key)
	assert.NotEmpty(t, result.Original.URL)
	assert.Equal(t, "image/jpeg", result.Original.ContentType)
	assert.Equal(t, 200, result.Original.Width)
	assert.Equal(t, 200, result.Original.Height)
	assert.True(t, result.Original.Size > 0)
	
	// Check variants - should have both regular and WebP variants
	assert.True(t, len(result.Variants) >= 3) // At least 3 regular variants
	
	variantNames := make(map[string]bool)
	for _, variant := range result.Variants {
		variantNames[variant.Name] = true
		assert.NotEmpty(t, variant.Key)
		assert.NotEmpty(t, variant.URL)
		assert.True(t, variant.Width > 0)
		assert.True(t, variant.Height > 0)
	}
	
	// Check that basic variants exist
	assert.True(t, variantNames["thumbnail"])
	assert.True(t, variantNames["medium"])
	assert.True(t, variantNames["large"])
	
	// Check that WebP variants exist
	assert.True(t, variantNames["thumbnail-webp"])
	assert.True(t, variantNames["medium-webp"])
	assert.True(t, variantNames["large-webp"])
	
	mockStorage.AssertExpectations(t)
}

func TestImageService_UploadImage_InvalidFormat(t *testing.T) {
	mockStorage := &MockStorageService{}
	service := NewImageService(mockStorage)
	
	ctx := context.Background()
	invalidData := []byte("not an image")
	filename := "test.txt"

	reader := bytes.NewReader(invalidData)
	result, err := service.UploadImage(ctx, reader, filename)
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode image")
}

func TestImageService_DeleteImage(t *testing.T) {
	mockStorage := &MockStorageService{}
	service := NewImageService(mockStorage)
	
	ctx := context.Background()
	keyPrefix := "test/image-123"

	// Mock delete calls for original and variants
	mockStorage.On("Delete", ctx, keyPrefix+"/original").Return(nil)
	mockStorage.On("Delete", ctx, keyPrefix+"/thumbnail").Return(nil)
	mockStorage.On("Delete", ctx, keyPrefix+"/medium").Return(nil)
	mockStorage.On("Delete", ctx, keyPrefix+"/large").Return(nil)

	err := service.DeleteImage(ctx, keyPrefix)
	
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestImageService_GetImageURL(t *testing.T) {
	mockStorage := &MockStorageService{}
	service := NewImageService(mockStorage)
	
	keyPrefix := "test/image-123"
	
	tests := []struct {
		name     string
		variant  string
		expected string
	}{
		{
			name:     "original variant",
			variant:  "original",
			expected: keyPrefix + "/original",
		},
		{
			name:     "thumbnail variant",
			variant:  "thumbnail",
			expected: keyPrefix + "/thumbnail",
		},
		{
			name:     "medium variant",
			variant:  "medium",
			expected: keyPrefix + "/medium",
		},
		{
			name:     "large variant",
			variant:  "large",
			expected: keyPrefix + "/large",
		},
		{
			name:     "unknown variant defaults to original",
			variant:  "unknown",
			expected: keyPrefix + "/original",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage.On("GetURL", tt.expected).Return("https://example.com/"+tt.expected).Once()
			
			url := service.GetImageURL(keyPrefix, tt.variant)
			assert.Equal(t, "https://example.com/"+tt.expected, url)
		})
	}
	
	mockStorage.AssertExpectations(t)
}

func TestGenerateImageKey(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "simple filename",
			filename: "image.jpg",
		},
		{
			name:     "filename with spaces",
			filename: "my image file.png",
		},
		{
			name:     "filename with path",
			filename: "/path/to/image.jpeg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := generateImageKey(tt.filename)
			
			// Check that key starts with events/ and contains date
			assert.True(t, strings.HasPrefix(key, "events/"))
			assert.Contains(t, key, "/")
			
			// Check that key doesn't contain spaces
			assert.NotContains(t, key, " ")
			
			// Check that key is lowercase
			assert.Equal(t, strings.ToLower(key), key)
		})
	}
}

func TestIsValidImageFormat(t *testing.T) {
	tests := []struct {
		format string
		valid  bool
	}{
		{"jpeg", true},
		{"jpg", true},
		{"png", true},
		{"webp", true},
		{"gif", false},
		{"bmp", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := isValidImageFormat(tt.format)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		format      string
		contentType string
	}{
		{"jpeg", "image/jpeg"},
		{"jpg", "image/jpeg"},
		{"png", "image/png"},
		{"webp", "image/webp"},
		{"unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := getContentType(tt.format)
			assert.Equal(t, tt.contentType, result)
		})
	}
}
func
 TestImageService_UploadImageWithOptions(t *testing.T) {
	mockStorage := &MockStorageService{}
	service := NewImageService(mockStorage)
	
	ctx := context.Background()
	testImage := createTestJPEG(200, 200)
	filename := "test-image.jpg"

	options := ImageProcessingOptions{
		Quality:         90,
		EnableWebP:      true,
		CompressionLevel: 6,
		SupportedFormats: []string{"image/webp", "image/jpeg"},
	}

	// Mock storage calls - original + 3 variants + 3 WebP variants = 7 calls
	mockStorage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int64")).
		Return("https://example.com/image.jpg", nil).Times(7)

	reader := bytes.NewReader(testImage)
	result, err := service.UploadImageWithOptions(ctx, reader, filename, options)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Check original metadata
	assert.NotEmpty(t, result.Original.Key)
	assert.NotEmpty(t, result.Original.URL)
	assert.Equal(t, 200, result.Original.Width)
	assert.Equal(t, 200, result.Original.Height)
	
	// Check variants - should have both regular and WebP variants
	assert.True(t, len(result.Variants) >= 3) // At least 3 regular variants
	
	// Check that WebP variants are created
	webpVariants := 0
	for _, variant := range result.Variants {
		if strings.HasSuffix(variant.Name, "-webp") {
			webpVariants++
		}
	}
	assert.Equal(t, 3, webpVariants) // Should have 3 WebP variants
	
	mockStorage.AssertExpectations(t)
}

func TestImageService_GetOptimalImageURL(t *testing.T) {
	mockStorage := &MockStorageService{}
	service := NewImageService(mockStorage)
	
	keyPrefix := "test/image-123"
	variant := "medium"
	
	tests := []struct {
		name         string
		acceptHeader string
		expectedKey  string
	}{
		{
			name:         "WebP supported",
			acceptHeader: "image/webp,image/jpeg,*/*",
			expectedKey:  keyPrefix + "/medium-webp",
		},
		{
			name:         "WebP not supported",
			acceptHeader: "image/jpeg,image/png,*/*",
			expectedKey:  keyPrefix + "/medium",
		},
		{
			name:         "Empty accept header",
			acceptHeader: "",
			expectedKey:  keyPrefix + "/medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage.On("GetURL", tt.expectedKey).Return("https://example.com/"+tt.expectedKey).Once()
			
			url := service.GetOptimalImageURL(keyPrefix, variant, tt.acceptHeader)
			assert.Equal(t, "https://example.com/"+tt.expectedKey, url)
		})
	}
	
	mockStorage.AssertExpectations(t)
}

func TestImageService_GetImageVariants(t *testing.T) {
	mockStorage := &MockStorageService{}
	service := NewImageService(mockStorage)
	
	keyPrefix := "test/image-123"
	variants := service.GetImageVariants(keyPrefix)
	
	// Should include original, all default variants, and WebP variants
	expectedVariants := []string{
		"original",
		"thumbnail", "thumbnail-webp",
		"medium", "medium-webp",
		"large", "large-webp",
	}
	
	assert.Len(t, variants, len(expectedVariants))
	
	for _, expected := range expectedVariants {
		assert.Contains(t, variants, expected)
	}
}

func TestDetermineOptimalFormat(t *testing.T) {
	service := NewImageService(&MockStorageService{})
	
	tests := []struct {
		name             string
		originalFormat   string
		supportedFormats []string
		enableWebP       bool
		expected         string
	}{
		{
			name:             "WebP supported and enabled",
			originalFormat:   "jpeg",
			supportedFormats: []string{"image/webp", "image/jpeg"},
			enableWebP:       true,
			expected:         "webp",
		},
		{
			name:             "WebP not supported",
			originalFormat:   "jpeg",
			supportedFormats: []string{"image/jpeg", "image/png"},
			enableWebP:       true,
			expected:         "jpeg",
		},
		{
			name:             "WebP disabled",
			originalFormat:   "jpeg",
			supportedFormats: []string{"image/webp", "image/jpeg"},
			enableWebP:       false,
			expected:         "jpeg",
		},
		{
			name:             "PNG with transparency",
			originalFormat:   "png",
			supportedFormats: []string{"image/png", "image/jpeg"},
			enableWebP:       false,
			expected:         "png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.determineOptimalFormat(tt.originalFormat, tt.supportedFormats, tt.enableWebP)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProcessImageData(t *testing.T) {
	service := NewImageService(&MockStorageService{})
	
	// Create a test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	
	tests := []struct {
		name    string
		format  string
		options ImageProcessingOptions
		wantErr bool
	}{
		{
			name:   "JPEG with custom quality",
			format: "jpeg",
			options: ImageProcessingOptions{
				Quality: 95,
			},
			wantErr: false,
		},
		{
			name:   "PNG with compression",
			format: "png",
			options: ImageProcessingOptions{
				CompressionLevel: 9,
			},
			wantErr: false,
		},
		{
			name:   "WebP fallback to JPEG",
			format: "webp",
			options: ImageProcessingOptions{
				Quality: 90,
			},
			wantErr: false,
		},
		{
			name:    "Unsupported format",
			format:  "gif",
			options: ImageProcessingOptions{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := service.processImageData(img, tt.format, tt.options)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, data)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, data)
				assert.True(t, len(data) > 0)
			}
		})
	}
}

func TestExtractImageMetadata(t *testing.T) {
	service := NewImageService(&MockStorageService{})
	
	// Create test images with different characteristics
	rgbaImg := image.NewRGBA(image.Rect(0, 0, 100, 100))
	grayImg := image.NewGray(image.Rect(0, 0, 100, 100))
	
	tests := []struct {
		name           string
		img            image.Image
		format         string
		expectedAlpha  bool
		expectedColor  string
	}{
		{
			name:          "RGBA image",
			img:           rgbaImg,
			format:        "png",
			expectedAlpha: true,
			expectedColor: "RGB",
		},
		{
			name:          "Grayscale image",
			img:           grayImg,
			format:        "jpeg",
			expectedAlpha: false,
			expectedColor: "Grayscale",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte("test data")
			metadata := service.extractImageMetadata(tt.img, tt.format, data)
			
			assert.Equal(t, tt.format, metadata.Format)
			assert.Equal(t, tt.expectedAlpha, metadata.HasAlpha)
			assert.Equal(t, tt.expectedColor, metadata.ColorSpace)
			assert.Equal(t, 100, metadata.Width)
			assert.Equal(t, 100, metadata.Height)
			assert.Equal(t, int64(len(data)), metadata.Size)
			assert.Equal(t, 1, metadata.Orientation) // Default orientation
		})
	}
}

// Integration test for the complete image processing pipeline
func TestImageProcessingPipeline_Integration(t *testing.T) {
	mockStorage := &MockStorageService{}
	service := NewImageService(mockStorage)
	
	ctx := context.Background()
	
	// Create a larger test image to better test resizing
	testImage := createTestJPEG(800, 600)
	filename := "test-large-image.jpg"

	options := ImageProcessingOptions{
		Quality:         80,
		EnableWebP:      true,
		CompressionLevel: 6,
		SupportedFormats: []string{"image/webp", "image/jpeg"},
	}

	// Mock all expected storage calls
	mockStorage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int64")).
		Return("https://cdn.example.com/image.jpg", nil).Maybe()

	// Test the complete pipeline
	reader := bytes.NewReader(testImage)
	result, err := service.UploadImageWithOptions(ctx, reader, filename, options)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Verify original image metadata
	assert.Equal(t, 800, result.Original.Width)
	assert.Equal(t, 600, result.Original.Height)
	assert.Contains(t, result.Original.Key, "original")
	
	// Verify variants are properly sized
	for _, variant := range result.Variants {
		switch {
		case strings.Contains(variant.Name, "thumbnail"):
			// Thumbnail should be no larger than 150x150
			assert.True(t, variant.Width <= 150)
			assert.True(t, variant.Height <= 150)
		case strings.Contains(variant.Name, "medium"):
			// Medium should be no larger than 400x300
			assert.True(t, variant.Width <= 400)
			assert.True(t, variant.Height <= 300)
		case strings.Contains(variant.Name, "large"):
			// Large should be no larger than 800x600
			assert.True(t, variant.Width <= 800)
			assert.True(t, variant.Height <= 600)
		}
		
		// All variants should maintain aspect ratio
		originalRatio := float64(result.Original.Width) / float64(result.Original.Height)
		variantRatio := float64(variant.Width) / float64(variant.Height)
		assert.InDelta(t, originalRatio, variantRatio, 0.1) // Allow small difference due to rounding
	}
	
	// Test optimal URL selection
	keyPrefix := "test/image-123"
	
	// Mock different URLs for WebP and regular variants
	mockStorage.On("GetURL", keyPrefix+"/medium-webp").Return("https://cdn.example.com/medium.webp").Maybe()
	mockStorage.On("GetURL", keyPrefix+"/medium").Return("https://cdn.example.com/medium.jpg").Maybe()
	
	webpURL := service.GetOptimalImageURL(keyPrefix, "medium", "image/webp,image/jpeg,*/*")
	jpegURL := service.GetOptimalImageURL(keyPrefix, "medium", "image/jpeg,image/png,*/*")
	
	assert.NotEqual(t, webpURL, jpegURL) // Should return different URLs based on support
}

// Test caching behavior (simulated)
func TestImageCachingStrategy(t *testing.T) {
	mockStorage := &MockStorageService{}
	service := NewImageService(mockStorage)
	
	ctx := context.Background()
	testData := []byte("test image data")
	key := "test/image.jpg"
	contentType := "image/jpeg"

	// Mock upload with caching headers expectation
	mockStorage.On("Upload", mock.Anything, key, mock.Anything, contentType, int64(len(testData))).
		Return("https://cdn.example.com/cached-image.jpg", nil).Once()

	// Test upload with caching
	url, err := service.uploadImageDataWithHeaders(ctx, key, testData, contentType)
	
	assert.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/cached-image.jpg", url)
	
	mockStorage.AssertExpectations(t)
}