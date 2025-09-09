package services

import (
	"testing"

	appconfig "event-ticketing-platform/internal/config"
)

func TestR2Service_GetOptimizedURL(t *testing.T) {
	tests := []struct {
		name     string
		config   appconfig.R2Config
		key      string
		options  *ImageURLOptions
		expected string
	}{
		{
			name: "basic URL without options",
			config: appconfig.R2Config{
				AccountID: "test-account",
			},
			key:      "events/2024/01/01/image-12345678/original.jpeg",
			options:  nil,
			expected: "https://pub-test-account.r2.dev/events/2024/01/01/image-12345678/original.jpeg",
		},
		{
			name: "URL with custom public URL",
			config: appconfig.R2Config{
				PublicURL: "https://cdn.example.com",
			},
			key:      "events/2024/01/01/image-12345678/original.jpeg",
			options:  nil,
			expected: "https://cdn.example.com/events/2024/01/01/image-12345678/original.jpeg",
		},
		{
			name: "URL with optimization parameters",
			config: appconfig.R2Config{
				AccountID: "test-account",
			},
			key: "events/2024/01/01/image-12345678/original.jpeg",
			options: &ImageURLOptions{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "webp",
			},
			expected: "https://pub-test-account.r2.dev/events/2024/01/01/image-12345678/original.jpeg?w=800&h=600&q=85&f=webp",
		},
		{
			name: "Cloudflare Images URL",
			config: appconfig.R2Config{
				PublicURL: "https://imagedelivery.net/test-hash",
			},
			key: "events/2024/01/01/image-12345678/original.jpeg",
			options: &ImageURLOptions{
				Width:  150,
				Height: 150,
			},
			expected: "https://imagedelivery.net/test-hash/events-2024-01-01-image-12345678-original.jpeg/thumbnail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &R2Service{config: tt.config}
			result := service.GetOptimizedURL(tt.key, tt.options)
			if result != tt.expected {
				t.Errorf("GetOptimizedURL() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestR2Service_buildOptimizedURL(t *testing.T) {
	service := &R2Service{}
	
	tests := []struct {
		name     string
		baseURL  string
		options  *ImageURLOptions
		expected string
	}{
		{
			name:     "no optimization options",
			baseURL:  "https://example.com/image.jpg",
			options:  &ImageURLOptions{},
			expected: "https://example.com/image.jpg",
		},
		{
			name:    "width only",
			baseURL: "https://example.com/image.jpg",
			options: &ImageURLOptions{
				Width: 800,
			},
			expected: "https://example.com/image.jpg?w=800",
		},
		{
			name:    "width and height",
			baseURL: "https://example.com/image.jpg",
			options: &ImageURLOptions{
				Width:  800,
				Height: 600,
			},
			expected: "https://example.com/image.jpg?w=800&h=600",
		},
		{
			name:    "all options",
			baseURL: "https://example.com/image.jpg",
			options: &ImageURLOptions{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "webp",
			},
			expected: "https://example.com/image.jpg?w=800&h=600&q=85&f=webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.buildOptimizedURL(tt.baseURL, tt.options)
			if result != tt.expected {
				t.Errorf("buildOptimizedURL() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestR2Service_buildCloudflareImagesURL(t *testing.T) {
	service := &R2Service{
		config: appconfig.R2Config{
			PublicURL: "https://imagedelivery.net/test-hash",
		},
	}
	
	tests := []struct {
		name     string
		key      string
		options  *ImageURLOptions
		expected string
	}{
		{
			name:    "default variant",
			key:     "events/2024/01/01/image-12345678",
			options: &ImageURLOptions{},
			expected: "https://imagedelivery.net/test-hash/events-2024-01-01-image-12345678/public",
		},
		{
			name: "thumbnail variant",
			key:  "events/2024/01/01/image-12345678",
			options: &ImageURLOptions{
				Width:  150,
				Height: 150,
			},
			expected: "https://imagedelivery.net/test-hash/events-2024-01-01-image-12345678/thumbnail",
		},
		{
			name: "medium variant",
			key:  "events/2024/01/01/image-12345678",
			options: &ImageURLOptions{
				Width:  400,
				Height: 300,
			},
			expected: "https://imagedelivery.net/test-hash/events-2024-01-01-image-12345678/medium",
		},
		{
			name: "large variant",
			key:  "events/2024/01/01/image-12345678",
			options: &ImageURLOptions{
				Width:  800,
				Height: 600,
			},
			expected: "https://imagedelivery.net/test-hash/events-2024-01-01-image-12345678/large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.buildCloudflareImagesURL(tt.key, tt.options)
			if result != tt.expected {
				t.Errorf("buildCloudflareImagesURL() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestR2Service_GetResponsiveImageURLs(t *testing.T) {
	service := &R2Service{
		config: appconfig.R2Config{
			AccountID: "test-account",
		},
	}
	
	key := "events/2024/01/01/image-12345678/original.jpeg"
	urls := service.GetResponsiveImageURLs(key)
	
	expectedSizes := []string{"thumbnail", "small", "medium", "large", "xlarge", "original"}
	
	for _, size := range expectedSizes {
		if _, exists := urls[size]; !exists {
			t.Errorf("GetResponsiveImageURLs() missing size %s", size)
		}
	}
	
	// Check that original URL is correct
	expectedOriginal := "https://pub-test-account.r2.dev/events/2024/01/01/image-12345678/original.jpeg"
	if urls["original"] != expectedOriginal {
		t.Errorf("GetResponsiveImageURLs() original = %v, expected %v", urls["original"], expectedOriginal)
	}
	
	// Check that optimized URLs have parameters
	if urls["thumbnail"] == urls["original"] {
		t.Error("GetResponsiveImageURLs() thumbnail should be different from original")
	}
}

func TestImageURLOptions_Structure(t *testing.T) {
	// Test that ImageURLOptions struct has expected fields
	options := ImageURLOptions{
		Width:   800,
		Height:  600,
		Quality: 85,
		Format:  "webp",
	}
	
	if options.Width != 800 {
		t.Errorf("ImageURLOptions.Width = %v, expected 800", options.Width)
	}
	if options.Height != 600 {
		t.Errorf("ImageURLOptions.Height = %v, expected 600", options.Height)
	}
	if options.Quality != 85 {
		t.Errorf("ImageURLOptions.Quality = %v, expected 85", options.Quality)
	}
	if options.Format != "webp" {
		t.Errorf("ImageURLOptions.Format = %v, expected webp", options.Format)
	}
}

func TestR2Service_GetURL_WithTrailingSlash(t *testing.T) {
	service := &R2Service{
		config: appconfig.R2Config{
			PublicURL: "https://cdn.example.com/",
		},
	}
	
	// Test that trailing slash in PublicURL is handled correctly
	result := service.GetURL("events/image.jpg")
	expected := "https://cdn.example.com/events/image.jpg"
	
	if result != expected {
		t.Errorf("GetURL() = %v, expected %v", result, expected)
	}
}

func TestR2Service_GetURL_WithLeadingSlash(t *testing.T) {
	service := &R2Service{
		config: appconfig.R2Config{
			AccountID: "test-account",
		},
	}
	
	// Test that leading slash in key is handled correctly
	result := service.GetURL("/events/image.jpg")
	expected := "https://pub-test-account.r2.dev/events/image.jpg"
	
	if result != expected {
		t.Errorf("GetURL() = %v, expected %v", result, expected)
	}
}