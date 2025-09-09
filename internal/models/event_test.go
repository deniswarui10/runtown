package models

import (
	"testing"
	"time"
)

func TestEvent_HasImage(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		expected bool
	}{
		{
			name: "event with both URL and key",
			event: Event{
				ImageURL: "https://example.com/image.jpg",
				ImageKey: "events/2024/01/01/test-image-12345678",
			},
			expected: true,
		},
		{
			name: "event with URL but no key",
			event: Event{
				ImageURL: "https://example.com/image.jpg",
				ImageKey: "",
			},
			expected: false,
		},
		{
			name: "event with key but no URL",
			event: Event{
				ImageURL: "",
				ImageKey: "events/2024/01/01/test-image-12345678",
			},
			expected: false,
		},
		{
			name: "event with neither URL nor key",
			event: Event{
				ImageURL: "",
				ImageKey: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.HasImage()
			if result != tt.expected {
				t.Errorf("HasImage() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEvent_GetImageMetadata(t *testing.T) {
	uploadTime := time.Now()
	
	tests := []struct {
		name     string
		event    Event
		expected *EventImageMetadata
	}{
		{
			name: "event with complete image metadata",
			event: Event{
				ImageURL:        "https://example.com/image.jpg",
				ImageKey:        "events/2024/01/01/test-image-12345678",
				ImageSize:       1024000,
				ImageFormat:     "jpeg",
				ImageWidth:      800,
				ImageHeight:     600,
				ImageUploadedAt: &uploadTime,
			},
			expected: &EventImageMetadata{
				Key:        "events/2024/01/01/test-image-12345678",
				URL:        "https://example.com/image.jpg",
				Size:       1024000,
				Format:     "jpeg",
				Width:      800,
				Height:     600,
				UploadedAt: &uploadTime,
			},
		},
		{
			name: "event without image",
			event: Event{
				ImageURL: "",
				ImageKey: "",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.GetImageMetadata()
			
			if tt.expected == nil {
				if result != nil {
					t.Errorf("GetImageMetadata() = %v, expected nil", result)
				}
				return
			}
			
			if result == nil {
				t.Errorf("GetImageMetadata() = nil, expected %v", tt.expected)
				return
			}
			
			if result.Key != tt.expected.Key {
				t.Errorf("GetImageMetadata().Key = %v, expected %v", result.Key, tt.expected.Key)
			}
			if result.URL != tt.expected.URL {
				t.Errorf("GetImageMetadata().URL = %v, expected %v", result.URL, tt.expected.URL)
			}
			if result.Size != tt.expected.Size {
				t.Errorf("GetImageMetadata().Size = %v, expected %v", result.Size, tt.expected.Size)
			}
			if result.Format != tt.expected.Format {
				t.Errorf("GetImageMetadata().Format = %v, expected %v", result.Format, tt.expected.Format)
			}
			if result.Width != tt.expected.Width {
				t.Errorf("GetImageMetadata().Width = %v, expected %v", result.Width, tt.expected.Width)
			}
			if result.Height != tt.expected.Height {
				t.Errorf("GetImageMetadata().Height = %v, expected %v", result.Height, tt.expected.Height)
			}
		})
	}
}

func TestEvent_SetImageMetadata(t *testing.T) {
	uploadTime := time.Now()
	
	tests := []struct {
		name     string
		event    Event
		metadata *EventImageMetadata
		expected Event
	}{
		{
			name:  "set complete image metadata",
			event: Event{},
			metadata: &EventImageMetadata{
				Key:        "events/2024/01/01/test-image-12345678",
				URL:        "https://example.com/image.jpg",
				Size:       1024000,
				Format:     "jpeg",
				Width:      800,
				Height:     600,
				UploadedAt: &uploadTime,
			},
			expected: Event{
				ImageKey:        "events/2024/01/01/test-image-12345678",
				ImageURL:        "https://example.com/image.jpg",
				ImageSize:       1024000,
				ImageFormat:     "jpeg",
				ImageWidth:      800,
				ImageHeight:     600,
				ImageUploadedAt: &uploadTime,
			},
		},
		{
			name: "clear image metadata with nil",
			event: Event{
				ImageKey:        "events/2024/01/01/test-image-12345678",
				ImageURL:        "https://example.com/image.jpg",
				ImageSize:       1024000,
				ImageFormat:     "jpeg",
				ImageWidth:      800,
				ImageHeight:     600,
				ImageUploadedAt: &uploadTime,
			},
			metadata: nil,
			expected: Event{
				ImageKey:        "",
				ImageURL:        "",
				ImageSize:       0,
				ImageFormat:     "",
				ImageWidth:      0,
				ImageHeight:     0,
				ImageUploadedAt: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.event.SetImageMetadata(tt.metadata)
			
			if tt.event.ImageKey != tt.expected.ImageKey {
				t.Errorf("SetImageMetadata() ImageKey = %v, expected %v", tt.event.ImageKey, tt.expected.ImageKey)
			}
			if tt.event.ImageURL != tt.expected.ImageURL {
				t.Errorf("SetImageMetadata() ImageURL = %v, expected %v", tt.event.ImageURL, tt.expected.ImageURL)
			}
			if tt.event.ImageSize != tt.expected.ImageSize {
				t.Errorf("SetImageMetadata() ImageSize = %v, expected %v", tt.event.ImageSize, tt.expected.ImageSize)
			}
			if tt.event.ImageFormat != tt.expected.ImageFormat {
				t.Errorf("SetImageMetadata() ImageFormat = %v, expected %v", tt.event.ImageFormat, tt.expected.ImageFormat)
			}
			if tt.event.ImageWidth != tt.expected.ImageWidth {
				t.Errorf("SetImageMetadata() ImageWidth = %v, expected %v", tt.event.ImageWidth, tt.expected.ImageWidth)
			}
			if tt.event.ImageHeight != tt.expected.ImageHeight {
				t.Errorf("SetImageMetadata() ImageHeight = %v, expected %v", tt.event.ImageHeight, tt.expected.ImageHeight)
			}
		})
	}
}

func TestEvent_ClearImageMetadata(t *testing.T) {
	uploadTime := time.Now()
	
	event := Event{
		ImageKey:        "events/2024/01/01/test-image-12345678",
		ImageURL:        "https://example.com/image.jpg",
		ImageSize:       1024000,
		ImageFormat:     "jpeg",
		ImageWidth:      800,
		ImageHeight:     600,
		ImageUploadedAt: &uploadTime,
	}
	
	event.ClearImageMetadata()
	
	if event.ImageKey != "" {
		t.Errorf("ClearImageMetadata() ImageKey = %v, expected empty string", event.ImageKey)
	}
	if event.ImageURL != "" {
		t.Errorf("ClearImageMetadata() ImageURL = %v, expected empty string", event.ImageURL)
	}
	if event.ImageSize != 0 {
		t.Errorf("ClearImageMetadata() ImageSize = %v, expected 0", event.ImageSize)
	}
	if event.ImageFormat != "" {
		t.Errorf("ClearImageMetadata() ImageFormat = %v, expected empty string", event.ImageFormat)
	}
	if event.ImageWidth != 0 {
		t.Errorf("ClearImageMetadata() ImageWidth = %v, expected 0", event.ImageWidth)
	}
	if event.ImageHeight != 0 {
		t.Errorf("ClearImageMetadata() ImageHeight = %v, expected 0", event.ImageHeight)
	}
	if event.ImageUploadedAt != nil {
		t.Errorf("ClearImageMetadata() ImageUploadedAt = %v, expected nil", event.ImageUploadedAt)
	}
}

func TestEvent_GetImageAspectRatio(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		expected float64
	}{
		{
			name: "landscape image",
			event: Event{
				ImageWidth:  800,
				ImageHeight: 600,
			},
			expected: 800.0 / 600.0,
		},
		{
			name: "portrait image",
			event: Event{
				ImageWidth:  600,
				ImageHeight: 800,
			},
			expected: 600.0 / 800.0,
		},
		{
			name: "square image",
			event: Event{
				ImageWidth:  600,
				ImageHeight: 600,
			},
			expected: 1.0,
		},
		{
			name: "no height",
			event: Event{
				ImageWidth:  800,
				ImageHeight: 0,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.GetImageAspectRatio()
			if result != tt.expected {
				t.Errorf("GetImageAspectRatio() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEvent_IsImageLandscape(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		expected bool
	}{
		{
			name: "landscape image",
			event: Event{
				ImageWidth:  800,
				ImageHeight: 600,
			},
			expected: true,
		},
		{
			name: "portrait image",
			event: Event{
				ImageWidth:  600,
				ImageHeight: 800,
			},
			expected: false,
		},
		{
			name: "square image",
			event: Event{
				ImageWidth:  600,
				ImageHeight: 600,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.IsImageLandscape()
			if result != tt.expected {
				t.Errorf("IsImageLandscape() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEvent_IsImagePortrait(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		expected bool
	}{
		{
			name: "portrait image",
			event: Event{
				ImageWidth:  600,
				ImageHeight: 800,
			},
			expected: true,
		},
		{
			name: "landscape image",
			event: Event{
				ImageWidth:  800,
				ImageHeight: 600,
			},
			expected: false,
		},
		{
			name: "square image",
			event: Event{
				ImageWidth:  600,
				ImageHeight: 600,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.IsImagePortrait()
			if result != tt.expected {
				t.Errorf("IsImagePortrait() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEvent_IsImageSquare(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		expected bool
	}{
		{
			name: "square image",
			event: Event{
				ImageWidth:  600,
				ImageHeight: 600,
			},
			expected: true,
		},
		{
			name: "landscape image",
			event: Event{
				ImageWidth:  800,
				ImageHeight: 600,
			},
			expected: false,
		},
		{
			name: "portrait image",
			event: Event{
				ImageWidth:  600,
				ImageHeight: 800,
			},
			expected: false,
		},
		{
			name: "no dimensions",
			event: Event{
				ImageWidth:  0,
				ImageHeight: 0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.IsImageSquare()
			if result != tt.expected {
				t.Errorf("IsImageSquare() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestValidateImageURLFormat(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid HTTPS image URL with extension",
			url:     "https://example.com/image.jpg",
			wantErr: false,
		},
		{
			name:    "valid HTTP image URL with extension",
			url:     "http://example.com/image.png",
			wantErr: false,
		},
		{
			name:    "valid R2 URL without extension",
			url:     "https://pub-123456.r2.dev/events/2024/01/01/image-12345678",
			wantErr: false,
		},
		{
			name:    "valid Cloudflare R2 storage URL",
			url:     "https://account.r2.cloudflarestorage.com/bucket/image.webp",
			wantErr: false,
		},
		{
			name:    "invalid protocol",
			url:     "ftp://example.com/image.jpg",
			wantErr: true,
		},
		{
			name:    "invalid URL format",
			url:     "not-a-url",
			wantErr: true,
		},
		{
			name:    "URL without image extension and not R2",
			url:     "https://example.com/not-an-image",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateImageURLFormat(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateImageURLFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateImageMetadata(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		format  string
		size    int64
		width   int
		height  int
		wantErr bool
	}{
		{
			name:    "valid metadata",
			key:     "events/2024/01/01/image-12345678",
			format:  "jpeg",
			size:    1024000,
			width:   800,
			height:  600,
			wantErr: false,
		},
		{
			name:    "empty metadata (valid for no image)",
			key:     "",
			format:  "",
			size:    0,
			width:   0,
			height:  0,
			wantErr: false,
		},
		{
			name:    "key too long",
			key:     string(make([]byte, 256)), // 256 characters
			format:  "jpeg",
			size:    1024000,
			width:   800,
			height:  600,
			wantErr: true,
		},
		{
			name:    "invalid format",
			key:     "events/2024/01/01/image-12345678",
			format:  "invalid",
			size:    1024000,
			width:   800,
			height:  600,
			wantErr: true,
		},
		{
			name:    "size too large (over 10MB)",
			key:     "events/2024/01/01/image-12345678",
			format:  "jpeg",
			size:    11 * 1024 * 1024, // 11MB
			width:   800,
			height:  600,
			wantErr: true,
		},
		{
			name:    "negative dimensions",
			key:     "events/2024/01/01/image-12345678",
			format:  "jpeg",
			size:    1024000,
			width:   -800,
			height:  600,
			wantErr: true,
		},
		{
			name:    "dimensions too large",
			key:     "events/2024/01/01/image-12345678",
			format:  "jpeg",
			size:    1024000,
			width:   15000,
			height:  600,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateImageMetadata(tt.key, tt.format, tt.size, tt.width, tt.height)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateImageMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEventCreateRequest_Validate_WithImageMetadata(t *testing.T) {
	validTime := time.Now().Add(24 * time.Hour)
	
	tests := []struct {
		name    string
		req     EventCreateRequest
		wantErr bool
	}{
		{
			name: "valid request with image metadata",
			req: EventCreateRequest{
				Title:       "Test Event",
				Description: "Test Description",
				StartDate:   validTime,
				EndDate:     validTime.Add(2 * time.Hour),
				Location:    "Test Location",
				CategoryID:  1,
				ImageURL:    "https://example.com/image.jpg",
				ImageKey:    "events/2024/01/01/image-12345678",
				ImageSize:   1024000,
				ImageFormat: "jpeg",
				ImageWidth:  800,
				ImageHeight: 600,
				Status:      StatusDraft,
			},
			wantErr: false,
		},
		{
			name: "valid request without image",
			req: EventCreateRequest{
				Title:       "Test Event",
				Description: "Test Description",
				StartDate:   validTime,
				EndDate:     validTime.Add(2 * time.Hour),
				Location:    "Test Location",
				CategoryID:  1,
				Status:      StatusDraft,
			},
			wantErr: false,
		},
		{
			name: "invalid image metadata",
			req: EventCreateRequest{
				Title:       "Test Event",
				Description: "Test Description",
				StartDate:   validTime,
				EndDate:     validTime.Add(2 * time.Hour),
				Location:    "Test Location",
				CategoryID:  1,
				ImageURL:    "https://example.com/image.jpg",
				ImageKey:    "events/2024/01/01/image-12345678",
				ImageSize:   11 * 1024 * 1024, // Too large
				ImageFormat: "jpeg",
				ImageWidth:  800,
				ImageHeight: 600,
				Status:      StatusDraft,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EventCreateRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}