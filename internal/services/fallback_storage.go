package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FallbackStorageService provides local file storage as a fallback when R2 is unavailable
type FallbackStorageService struct {
	basePath  string
	baseURL   string
	publicDir string
}

// NewFallbackStorageService creates a new fallback storage service
func NewFallbackStorageService(basePath, baseURL string) *FallbackStorageService {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		fmt.Printf("Warning: Failed to create storage directory %s: %v\n", basePath, err)
	}

	return &FallbackStorageService{
		basePath:  basePath,
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		publicDir: "uploads",
	}
}

// Upload saves a file to local storage
func (f *FallbackStorageService) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error) {
	// Clean the key
	key = strings.TrimPrefix(key, "/")
	
	// Create full file path
	fullPath := filepath.Join(f.basePath, key)
	
	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	
	// Create the file
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file %s: %w", fullPath, err)
	}
	defer file.Close()
	
	// Copy data to file
	written, err := io.Copy(file, reader)
	if err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}
	
	if written != size {
		return "", fmt.Errorf("size mismatch: expected %d bytes, wrote %d bytes", size, written)
	}
	
	// Return public URL
	url := f.GetURL(key)
	fmt.Printf("Fallback storage: saved %s to %s\n", key, fullPath)
	
	return url, nil
}

// Delete removes a file from local storage
func (f *FallbackStorageService) Delete(ctx context.Context, key string) error {
	key = strings.TrimPrefix(key, "/")
	fullPath := filepath.Join(f.basePath, key)
	
	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file %s: %w", fullPath, err)
	}
	
	// Try to remove empty directories
	dir := filepath.Dir(fullPath)
	f.cleanupEmptyDirs(dir)
	
	return nil
}

// GetURL returns the public URL for a file
func (f *FallbackStorageService) GetURL(key string) string {
	key = strings.TrimPrefix(key, "/")
	return fmt.Sprintf("%s/%s", f.baseURL, key)
}

// GeneratePresignedURL is not supported for fallback storage
func (f *FallbackStorageService) GeneratePresignedURL(ctx context.Context, key string, contentType string, expiration time.Duration) (string, error) {
	return "", fmt.Errorf("presigned URLs not supported by fallback storage")
}

// Exists checks if a file exists in local storage
func (f *FallbackStorageService) Exists(ctx context.Context, key string) (bool, error) {
	key = strings.TrimPrefix(key, "/")
	fullPath := filepath.Join(f.basePath, key)
	
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if file exists: %w", err)
	}
	
	return true, nil
}

// cleanupEmptyDirs removes empty directories up to the base path
func (f *FallbackStorageService) cleanupEmptyDirs(dir string) {
	// Don't remove the base path itself
	if dir == f.basePath || dir == "." || dir == "/" {
		return
	}
	
	// Check if directory is empty
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) > 0 {
		return
	}
	
	// Remove empty directory
	if err := os.Remove(dir); err == nil {
		// Recursively clean parent directories
		parent := filepath.Dir(dir)
		f.cleanupEmptyDirs(parent)
	}
}

// StorageServiceWithFallback wraps a primary storage service with a fallback
type StorageServiceWithFallback struct {
	primary  StorageService
	fallback StorageService
}

// NewStorageServiceWithFallback creates a storage service with fallback capability
func NewStorageServiceWithFallback(primary, fallback StorageService) *StorageServiceWithFallback {
	return &StorageServiceWithFallback{
		primary:  primary,
		fallback: fallback,
	}
}

// Upload tries primary storage first, falls back to fallback storage on error
func (s *StorageServiceWithFallback) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error) {
	// Try primary storage first
	url, err := s.primary.Upload(ctx, key, reader, contentType, size)
	if err == nil {
		return url, nil
	}
	
	fmt.Printf("Primary storage failed, using fallback: %v\n", err)
	
	// Reset reader if possible
	if seeker, ok := reader.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	} else {
		return "", fmt.Errorf("primary storage failed and cannot reset reader for fallback: %w", err)
	}
	
	// Try fallback storage
	return s.fallback.Upload(ctx, key, reader, contentType, size)
}

// Delete tries to delete from both storages
func (s *StorageServiceWithFallback) Delete(ctx context.Context, key string) error {
	var primaryErr, fallbackErr error
	
	// Try to delete from primary
	primaryErr = s.primary.Delete(ctx, key)
	
	// Try to delete from fallback
	fallbackErr = s.fallback.Delete(ctx, key)
	
	// Return error only if both failed
	if primaryErr != nil && fallbackErr != nil {
		return fmt.Errorf("both storages failed - primary: %v, fallback: %v", primaryErr, fallbackErr)
	}
	
	return nil
}

// GetURL returns URL from primary storage
func (s *StorageServiceWithFallback) GetURL(key string) string {
	return s.primary.GetURL(key)
}

// GeneratePresignedURL uses primary storage
func (s *StorageServiceWithFallback) GeneratePresignedURL(ctx context.Context, key string, contentType string, expiration time.Duration) (string, error) {
	return s.primary.GeneratePresignedURL(ctx, key, contentType, expiration)
}

// Exists checks both storages
func (s *StorageServiceWithFallback) Exists(ctx context.Context, key string) (bool, error) {
	// Check primary first
	exists, err := s.primary.Exists(ctx, key)
	if err == nil && exists {
		return true, nil
	}
	
	// Check fallback
	return s.fallback.Exists(ctx, key)
}