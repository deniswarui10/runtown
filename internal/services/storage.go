package services

import (
	"context"
	"io"
	"time"
)

// StorageService defines the interface for file storage operations
type StorageService interface {
	// Upload uploads a file to storage and returns the public URL
	Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error)
	
	// Delete removes a file from storage
	Delete(ctx context.Context, key string) error
	
	// GetURL returns the public URL for a file
	GetURL(key string) string
	
	// GeneratePresignedURL generates a presigned URL for direct upload
	GeneratePresignedURL(ctx context.Context, key string, contentType string, expiration time.Duration) (string, error)
	
	// Exists checks if a file exists in storage
	Exists(ctx context.Context, key string) (bool, error)
}

// ImageMetadata contains metadata about uploaded images
type ImageMetadata struct {
	Key         string    `json:"key"`
	URL         string    `json:"url"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// ImageVariant represents different sizes of the same image
type ImageVariant struct {
	Name   string `json:"name"`   // thumbnail, medium, large
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Key    string `json:"key"`
	URL    string `json:"url"`
}

// ImageUploadResult contains the result of an image upload operation
type ImageUploadResult struct {
	Original ImageMetadata   `json:"original"`
	Variants []ImageVariant  `json:"variants"`
}

// ImageProcessingOptions defines options for image processing
type ImageProcessingOptions struct {
	Quality         int      // JPEG quality (1-100)
	SupportedFormats []string // Supported formats by client (from Accept header)
	EnableWebP      bool     // Whether to generate WebP variants
	CompressionLevel int     // PNG compression level (0-9)
	CropData        *CropData // Optional crop information
}

// CropData defines crop coordinates and dimensions
type CropData struct {
	X      int `json:"x"`      // X coordinate of crop area
	Y      int `json:"y"`      // Y coordinate of crop area
	Width  int `json:"width"`  // Width of crop area
	Height int `json:"height"` // Height of crop area
}

// ImageMetadataExtended contains extended metadata about images
type ImageMetadataExtended struct {
	ImageMetadata
	Format       string            `json:"format"`
	ColorSpace   string            `json:"color_space"`
	HasAlpha     bool              `json:"has_alpha"`
	Orientation  int               `json:"orientation"`
	ExifData     map[string]string `json:"exif_data,omitempty"`
}