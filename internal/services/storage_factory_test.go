package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"event-ticketing-platform/internal/config"
)

func TestNewStorageFactory(t *testing.T) {
	cfg := &config.Config{}
	factory := NewStorageFactory(cfg)
	
	assert.NotNil(t, factory)
	assert.Equal(t, cfg, factory.config)
}

func TestStorageFactory_ValidateR2Configuration(t *testing.T) {
	tests := []struct {
		name    string
		config  config.R2Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: config.R2Config{
				AccountID:       "test-account",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				BucketName:      "test-bucket",
			},
			wantErr: false,
		},
		{
			name: "missing account ID",
			config: config.R2Config{
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				BucketName:      "test-bucket",
			},
			wantErr: true,
			errMsg:  "R2_ACCOUNT_ID is required",
		},
		{
			name: "missing access key",
			config: config.R2Config{
				AccountID:       "test-account",
				SecretAccessKey: "test-secret",
				BucketName:      "test-bucket",
			},
			wantErr: true,
			errMsg:  "R2_ACCESS_KEY_ID is required",
		},
		{
			name: "missing secret key",
			config: config.R2Config{
				AccountID:   "test-account",
				AccessKeyID: "test-key",
				BucketName:  "test-bucket",
			},
			wantErr: true,
			errMsg:  "R2_SECRET_ACCESS_KEY is required",
		},
		{
			name: "missing bucket name",
			config: config.R2Config{
				AccountID:       "test-account",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
			},
			wantErr: true,
			errMsg:  "R2_BUCKET_NAME is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{R2: tt.config}
			factory := NewStorageFactory(cfg)
			
			err := factory.ValidateR2Configuration()
			
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

func TestStorageFactory_CreateStorageService_FallbackOnly(t *testing.T) {
	// Test with invalid R2 config - should fall back to local storage
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: "8080",
		},
		R2: config.R2Config{
			// Invalid/empty config
		},
	}
	
	factory := NewStorageFactory(cfg)
	service, err := factory.CreateStorageService()
	
	require.NoError(t, err)
	assert.NotNil(t, service)
	
	// Should be fallback service since R2 config is invalid
	_, isFallback := service.(*FallbackStorageService)
	assert.True(t, isFallback, "Should use fallback storage when R2 is not configured")
}

func TestStorageFactory_CreateImageService(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: "8080",
		},
		R2: config.R2Config{
			// Invalid config - will use fallback
		},
	}
	
	factory := NewStorageFactory(cfg)
	service, err := factory.CreateImageService()
	
	require.NoError(t, err)
	assert.NotNil(t, service)
	
	// Should be ImageService
	_, isImageService := service.(*ImageService)
	assert.True(t, isImageService)
}

func TestStorageFactory_GetStorageInfo(t *testing.T) {
	tests := []struct {
		name   string
		config config.R2Config
	}{
		{
			name: "R2 configured",
			config: config.R2Config{
				AccountID:       "test-account",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				BucketName:      "test-bucket",
				PublicURL:       "https://cdn.example.com",
			},
		},
		{
			name: "R2 not configured",
			config: config.R2Config{
				BucketName: "test-bucket",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{R2: tt.config}
			factory := NewStorageFactory(cfg)
			
			info := factory.GetStorageInfo()
			
			assert.NotNil(t, info)
			assert.Contains(t, info, "r2_configured")
			assert.Contains(t, info, "bucket_name")
			assert.Contains(t, info, "public_url")
			assert.Contains(t, info, "fallback_path")
			assert.Contains(t, info, "r2_available")
			
			// Check r2_configured value
			expectedConfigured := tt.config.AccessKeyID != "" && tt.config.SecretAccessKey != ""
			assert.Equal(t, expectedConfigured, info["r2_configured"])
			
			// Check bucket name
			assert.Equal(t, tt.config.BucketName, info["bucket_name"])
			
			// Check public URL
			assert.Equal(t, tt.config.PublicURL, info["public_url"])
			
			// Check fallback path (handle Windows path separators)
			fallbackPath := info["fallback_path"].(string)
			assert.True(t, strings.Contains(fallbackPath, "web") && strings.Contains(fallbackPath, "static") && strings.Contains(fallbackPath, "uploads"))
		})
	}
}

func TestStorageFactory_SetupR2Bucket_InvalidConfig(t *testing.T) {
	cfg := &config.Config{
		R2: config.R2Config{
			// Invalid config
		},
	}
	
	factory := NewStorageFactory(cfg)
	err := factory.SetupR2Bucket()
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create R2 service")
}