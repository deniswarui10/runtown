package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

// ImageService handles image processing and storage operations
type ImageService struct {
	storage StorageService
}

// NewImageService creates a new image service
func NewImageService(storage StorageService) *ImageService {
	return &ImageService{
		storage: storage,
	}
}

// ImageVariantConfig defines the configuration for image variants
type ImageVariantConfig struct {
	Name   string
	Width  int
	Height int
	Fit    imaging.ResampleFilter
}



// Default image variants
var DefaultImageVariants = []ImageVariantConfig{
	{Name: "thumbnail", Width: 150, Height: 150, Fit: imaging.Lanczos},
	{Name: "medium", Width: 400, Height: 300, Fit: imaging.Lanczos},
	{Name: "large", Width: 800, Height: 600, Fit: imaging.Lanczos},
}

// UploadImage processes and uploads an image with multiple variants
func (s *ImageService) UploadImage(ctx context.Context, reader io.Reader, filename string) (*ImageUploadResult, error) {
	return s.UploadImageWithOptions(ctx, reader, filename, ImageProcessingOptions{
		Quality:         85,
		EnableWebP:      true,
		CompressionLevel: 6,
	})
}

// UploadImageWithOptions processes and uploads an image with custom options
func (s *ImageService) UploadImageWithOptions(ctx context.Context, reader io.Reader, filename string, options ImageProcessingOptions) (*ImageUploadResult, error) {
	// Read the image data
	imageData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Validate image format
	if !isValidImageFormat(format) {
		return nil, fmt.Errorf("unsupported image format: %s", format)
	}

	// Apply cropping if specified
	if options.CropData != nil {
		img, err = s.cropImage(img, options.CropData)
		if err != nil {
			return nil, fmt.Errorf("failed to crop image: %w", err)
		}
	}

	// Extract extended metadata (for future use)
	_ = s.extractImageMetadata(img, format, imageData)

	// Generate unique key prefix
	keyPrefix := generateImageKey(filename)
	
	// Get image dimensions
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()

	// For now, keep original format for storage and create WebP variants separately
	storageFormat := format

	// Process and upload original image with optimization
	originalData, err := s.processImageData(img, storageFormat, options)
	if err != nil {
		return nil, fmt.Errorf("failed to process original image: %w", err)
	}

	originalKey := fmt.Sprintf("%s/original.%s", keyPrefix, storageFormat)
	originalURL, err := s.uploadImageDataWithHeaders(ctx, originalKey, originalData, getContentType(storageFormat))
	if err != nil {
		return nil, fmt.Errorf("failed to upload original image: %w", err)
	}

	// Create original metadata
	original := ImageMetadata{
		Key:         originalKey,
		URL:         originalURL,
		Size:        int64(len(originalData)),
		ContentType: getContentType(storageFormat),
		Width:       originalWidth,
		Height:      originalHeight,
		UploadedAt:  time.Now(),
	}

	// Create variants with format optimization
	variants := make([]ImageVariant, 0, len(DefaultImageVariants)*2) // *2 for potential WebP variants
	
	for _, config := range DefaultImageVariants {
		// Create variant in original format
		variant, err := s.createImageVariantWithOptions(ctx, img, keyPrefix, config, storageFormat, options)
		if err != nil {
			// Log error but continue with other variants
			fmt.Printf("Failed to create variant %s: %v\n", config.Name, err)
			continue
		}
		variants = append(variants, *variant)

		// Create WebP variant if enabled and format is not already WebP
		if options.EnableWebP && storageFormat != "webp" {
			webpVariant, err := s.createImageVariantWithOptions(ctx, img, keyPrefix, config, "webp", options)
			if err != nil {
				fmt.Printf("Failed to create WebP variant %s: %v\n", config.Name, err)
				continue
			}
			webpVariant.Name = config.Name + "-webp"
			variants = append(variants, *webpVariant)
		}
	}

	return &ImageUploadResult{
		Original: original,
		Variants: variants,
	}, nil
}

// createImageVariant creates a resized variant of the image (legacy method)
func (s *ImageService) createImageVariant(ctx context.Context, img image.Image, keyPrefix string, config ImageVariantConfig, format string) (*ImageVariant, error) {
	return s.createImageVariantWithOptions(ctx, img, keyPrefix, config, format, ImageProcessingOptions{
		Quality:         85,
		CompressionLevel: 6,
	})
}

// createImageVariantWithOptions creates a resized variant of the image with custom options
func (s *ImageService) createImageVariantWithOptions(ctx context.Context, img image.Image, keyPrefix string, config ImageVariantConfig, format string, options ImageProcessingOptions) (*ImageVariant, error) {
	// Resize image maintaining aspect ratio
	resized := imaging.Fit(img, config.Width, config.Height, config.Fit)
	
	// Process the resized image with optimization
	imageData, err := s.processImageData(resized, format, options)
	if err != nil {
		return nil, fmt.Errorf("failed to process variant image: %w", err)
	}

	// Upload variant
	variantKey := fmt.Sprintf("%s/%s.%s", keyPrefix, config.Name, format)
	variantURL, err := s.uploadImageDataWithHeaders(ctx, variantKey, imageData, getContentType(format))
	if err != nil {
		return nil, fmt.Errorf("failed to upload variant: %w", err)
	}

	// Get variant dimensions
	bounds := resized.Bounds()
	
	return &ImageVariant{
		Name:   config.Name,
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Key:    variantKey,
		URL:    variantURL,
	}, nil
}

// extractImageMetadata extracts extended metadata from an image
func (s *ImageService) extractImageMetadata(img image.Image, format string, data []byte) ImageMetadataExtended {
	bounds := img.Bounds()
	
	// Determine if image has alpha channel
	hasAlpha := false
	switch img.(type) {
	case *image.NRGBA, *image.RGBA, *image.NRGBA64, *image.RGBA64:
		hasAlpha = true
	}

	// Basic color space detection
	colorSpace := "RGB"
	switch img.ColorModel() {
	case color.GrayModel, color.Gray16Model:
		colorSpace = "Grayscale"
	case color.CMYKModel:
		colorSpace = "CMYK"
	}

	return ImageMetadataExtended{
		ImageMetadata: ImageMetadata{
			Size:        int64(len(data)),
			ContentType: getContentType(format),
			Width:       bounds.Dx(),
			Height:      bounds.Dy(),
			UploadedAt:  time.Now(),
		},
		Format:      format,
		ColorSpace:  colorSpace,
		HasAlpha:    hasAlpha,
		Orientation: 1, // Default orientation
		ExifData:    make(map[string]string),
	}
}

// determineOptimalFormat determines the best format for storage based on client support
func (s *ImageService) determineOptimalFormat(originalFormat string, supportedFormats []string, enableWebP bool) string {
	// If WebP is enabled and supported by client, prefer WebP for better compression
	if enableWebP {
		for _, format := range supportedFormats {
			if strings.Contains(strings.ToLower(format), "webp") {
				return "webp"
			}
		}
	}

	// For images with transparency, prefer PNG
	if originalFormat == "png" {
		return "png"
	}

	// Default to JPEG for photos
	return "jpeg"
}

// processImageData processes and encodes image data with optimization
func (s *ImageService) processImageData(img image.Image, format string, options ImageProcessingOptions) ([]byte, error) {
	var buf bytes.Buffer
	
	switch format {
	case "jpeg", "jpg":
		quality := options.Quality
		if quality <= 0 || quality > 100 {
			quality = 85
		}
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		if err != nil {
			return nil, fmt.Errorf("failed to encode JPEG: %w", err)
		}
	case "png":
		// Use PNG encoder with compression
		encoder := &png.Encoder{
			CompressionLevel: png.CompressionLevel(options.CompressionLevel),
		}
		err := encoder.Encode(&buf, img)
		if err != nil {
			return nil, fmt.Errorf("failed to encode PNG: %w", err)
		}
	case "webp":
		// For WebP, we'll use a simple approach since golang.org/x/image/webp only supports decoding
		// Convert to JPEG with high quality as fallback
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
		if err != nil {
			return nil, fmt.Errorf("failed to encode WebP fallback: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format for processing: %s", format)
	}

	return buf.Bytes(), nil
}

// uploadImageData uploads image data to storage (legacy method)
func (s *ImageService) uploadImageData(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	reader := bytes.NewReader(data)
	return s.storage.Upload(ctx, key, reader, contentType, int64(len(data)))
}

// uploadImageDataWithHeaders uploads image data to storage with caching headers
func (s *ImageService) uploadImageDataWithHeaders(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	reader := bytes.NewReader(data)
	
	// Add caching metadata to context for storage implementations that support it
	ctx = s.addCachingContext(ctx, contentType)
	
	return s.storage.Upload(ctx, key, reader, contentType, int64(len(data)))
}

// addCachingContext adds caching-related metadata to the context
func (s *ImageService) addCachingContext(ctx context.Context, contentType string) context.Context {
	// Add cache control headers based on content type
	var cacheControl string
	var maxAge int
	
	switch {
	case strings.HasPrefix(contentType, "image/"):
		// Images can be cached for a long time since they're immutable
		cacheControl = "public, max-age=31536000, immutable" // 1 year
		maxAge = 31536000
	default:
		cacheControl = "public, max-age=86400" // 1 day
		maxAge = 86400
	}
	
	// Store caching metadata in context for storage implementations to use
	ctx = context.WithValue(ctx, "cache-control", cacheControl)
	ctx = context.WithValue(ctx, "max-age", maxAge)
	ctx = context.WithValue(ctx, "expires", time.Now().Add(time.Duration(maxAge)*time.Second))
	
	return ctx
}

// DeleteImage deletes an image and all its variants
func (s *ImageService) DeleteImage(ctx context.Context, keyPrefix string) error {
	// Delete original
	originalKey := fmt.Sprintf("%s/original", keyPrefix)
	if err := s.storage.Delete(ctx, originalKey); err != nil {
		fmt.Printf("Failed to delete original image %s: %v\n", originalKey, err)
	}

	// Delete variants
	for _, config := range DefaultImageVariants {
		variantKey := fmt.Sprintf("%s/%s", keyPrefix, config.Name)
		if err := s.storage.Delete(ctx, variantKey); err != nil {
			fmt.Printf("Failed to delete variant %s: %v\n", variantKey, err)
		}
	}

	return nil
}

// ValidateImage validates image file before processing
func (s *ImageService) ValidateImage(reader io.Reader, maxSize int64) error {
	// Check file size
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read image: %w", err)
	}

	if int64(len(data)) > maxSize {
		return fmt.Errorf("image size %d bytes exceeds maximum allowed size %d bytes", len(data), maxSize)
	}

	// Try to decode image
	_, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("invalid image format: %w", err)
	}

	if !isValidImageFormat(format) {
		return fmt.Errorf("unsupported image format: %s", format)
	}

	return nil
}

// generateImageKey generates a unique key for image storage
func generateImageKey(filename string) string {
	// Generate UUID for uniqueness
	id := uuid.New().String()
	
	// Extract base name without extension
	baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	
	// Clean the filename
	baseName = strings.ReplaceAll(baseName, " ", "-")
	baseName = strings.ToLower(baseName)
	
	// Create key with timestamp for organization
	timestamp := time.Now().Format("2006/01/02")
	
	return fmt.Sprintf("events/%s/%s-%s", timestamp, baseName, id[:8])
}

// isValidImageFormat checks if the image format is supported
func isValidImageFormat(format string) bool {
	switch format {
	case "jpeg", "jpg", "png", "webp":
		return true
	default:
		return false
	}
}

// getContentType returns the MIME type for the image format
func getContentType(format string) string {
	switch format {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// GetImageURL returns the URL for a specific image variant
func (s *ImageService) GetImageURL(keyPrefix, variant string) string {
	if variant == "original" {
		return s.storage.GetURL(fmt.Sprintf("%s/original", keyPrefix))
	}
	
	// Check if variant exists in our default variants
	for _, config := range DefaultImageVariants {
		if config.Name == variant {
			return s.storage.GetURL(fmt.Sprintf("%s/%s", keyPrefix, variant))
		}
	}
	
	// Default to original if variant not found
	return s.storage.GetURL(fmt.Sprintf("%s/original", keyPrefix))
}

// GetOptimalImageURL returns the best image URL based on browser support
func (s *ImageService) GetOptimalImageURL(keyPrefix, variant string, acceptHeader string) string {
	// Check if browser supports WebP
	supportsWebP := strings.Contains(strings.ToLower(acceptHeader), "image/webp")
	
	if supportsWebP {
		// Try to get WebP variant first
		webpVariant := variant + "-webp"
		webpURL := s.storage.GetURL(fmt.Sprintf("%s/%s", keyPrefix, webpVariant))
		
		// In a real implementation, we'd check if the WebP variant exists
		// For now, we'll assume it exists if WebP is supported
		return webpURL
	}
	
	// Fall back to regular variant
	return s.GetImageURL(keyPrefix, variant)
}

// GetImageVariants returns all available variants for an image
func (s *ImageService) GetImageVariants(keyPrefix string) []string {
	variants := []string{"original"}
	
	for _, config := range DefaultImageVariants {
		variants = append(variants, config.Name)
		// Add WebP variant if it exists
		variants = append(variants, config.Name+"-webp")
	}
	
	return variants
}

// cropImage crops an image to the specified dimensions
func (s *ImageService) cropImage(img image.Image, cropData *CropData) (image.Image, error) {
	bounds := img.Bounds()
	
	// Validate crop coordinates
	if cropData.X < 0 || cropData.Y < 0 {
		return nil, fmt.Errorf("crop coordinates cannot be negative")
	}
	
	if cropData.X+cropData.Width > bounds.Dx() || cropData.Y+cropData.Height > bounds.Dy() {
		return nil, fmt.Errorf("crop area exceeds image bounds")
	}
	
	if cropData.Width <= 0 || cropData.Height <= 0 {
		return nil, fmt.Errorf("crop dimensions must be positive")
	}
	
	// Create crop rectangle
	cropRect := image.Rect(
		cropData.X,
		cropData.Y,
		cropData.X+cropData.Width,
		cropData.Y+cropData.Height,
	)
	
	// Use imaging library to crop
	croppedImg := imaging.Crop(img, cropRect)
	
	return croppedImg, nil
}