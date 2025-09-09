package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFallbackStorageService(t *testing.T) {
	tempDir := t.TempDir()
	baseURL := "http://localhost:8080"
	
	service := NewFallbackStorageService(tempDir, baseURL)
	
	assert.NotNil(t, service)
	assert.Equal(t, tempDir, service.basePath)
	assert.Equal(t, baseURL, service.baseURL)
	assert.Equal(t, "uploads", service.publicDir)
	
	// Check that directory was created
	_, err := os.Stat(tempDir)
	assert.NoError(t, err)
}

func TestFallbackStorageService_Upload(t *testing.T) {
	tempDir := t.TempDir()
	service := NewFallbackStorageService(tempDir, "http://localhost:8080")
	
	ctx := context.Background()
	testContent := "test file content"
	reader := strings.NewReader(testContent)
	
	url, err := service.Upload(ctx, "test/file.txt", reader, "text/plain", int64(len(testContent)))
	
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/uploads/test/file.txt", url)
	
	// Check that file was created
	filePath := filepath.Join(tempDir, "test", "file.txt")
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestFallbackStorageService_Upload_WithLeadingSlash(t *testing.T) {
	tempDir := t.TempDir()
	service := NewFallbackStorageService(tempDir, "http://localhost:8080")
	
	ctx := context.Background()
	testContent := "test content"
	reader := strings.NewReader(testContent)
	
	url, err := service.Upload(ctx, "/test/file.txt", reader, "text/plain", int64(len(testContent)))
	
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/uploads/test/file.txt", url)
}

func TestFallbackStorageService_Delete(t *testing.T) {
	tempDir := t.TempDir()
	service := NewFallbackStorageService(tempDir, "http://localhost:8080")
	
	ctx := context.Background()
	
	// First upload a file
	testContent := "test content"
	reader := strings.NewReader(testContent)
	_, err := service.Upload(ctx, "test/file.txt", reader, "text/plain", int64(len(testContent)))
	require.NoError(t, err)
	
	// Verify file exists
	filePath := filepath.Join(tempDir, "test", "file.txt")
	_, err = os.Stat(filePath)
	require.NoError(t, err)
	
	// Delete the file
	err = service.Delete(ctx, "test/file.txt")
	require.NoError(t, err)
	
	// Verify file is gone
	_, err = os.Stat(filePath)
	assert.True(t, os.IsNotExist(err))
}

func TestFallbackStorageService_Delete_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	service := NewFallbackStorageService(tempDir, "http://localhost:8080")
	
	ctx := context.Background()
	
	// Delete non-existent file should not error
	err := service.Delete(ctx, "nonexistent/file.txt")
	assert.NoError(t, err)
}

func TestFallbackStorageService_GetURL(t *testing.T) {
	service := NewFallbackStorageService("/tmp", "https://example.com")
	
	tests := []struct {
		key      string
		expected string
	}{
		{"test/file.txt", "https://example.com/uploads/test/file.txt"},
		{"/test/file.txt", "https://example.com/uploads/test/file.txt"},
		{"file.txt", "https://example.com/uploads/file.txt"},
	}
	
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			url := service.GetURL(tt.key)
			assert.Equal(t, tt.expected, url)
		})
	}
}

func TestFallbackStorageService_Exists(t *testing.T) {
	tempDir := t.TempDir()
	service := NewFallbackStorageService(tempDir, "http://localhost:8080")
	
	ctx := context.Background()
	
	// Check non-existent file
	exists, err := service.Exists(ctx, "nonexistent.txt")
	require.NoError(t, err)
	assert.False(t, exists)
	
	// Upload a file
	testContent := "test content"
	reader := strings.NewReader(testContent)
	_, err = service.Upload(ctx, "test/file.txt", reader, "text/plain", int64(len(testContent)))
	require.NoError(t, err)
	
	// Check existing file
	exists, err = service.Exists(ctx, "test/file.txt")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestFallbackStorageService_GeneratePresignedURL(t *testing.T) {
	service := NewFallbackStorageService("/tmp", "http://localhost:8080")
	
	ctx := context.Background()
	
	_, err := service.GeneratePresignedURL(ctx, "test.txt", "text/plain", time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "presigned URLs not supported")
}

func TestStorageServiceWithFallback_Upload_PrimarySuccess(t *testing.T) {
	mockPrimary := &MockStorageService{}
	mockFallback := &MockStorageService{}
	
	service := NewStorageServiceWithFallback(mockPrimary, mockFallback)
	
	ctx := context.Background()
	reader := strings.NewReader("test content")
	
	// Primary succeeds
	mockPrimary.On("Upload", ctx, "test.txt", reader, "text/plain", int64(12)).
		Return("https://primary.com/test.txt", nil)
	
	url, err := service.Upload(ctx, "test.txt", reader, "text/plain", 12)
	
	require.NoError(t, err)
	assert.Equal(t, "https://primary.com/test.txt", url)
	
	mockPrimary.AssertExpectations(t)
	mockFallback.AssertNotCalled(t, "Upload")
}

func TestStorageServiceWithFallback_Upload_PrimaryFailsFallbackSucceeds(t *testing.T) {
	mockPrimary := &MockStorageService{}
	mockFallback := &MockStorageService{}
	
	service := NewStorageServiceWithFallback(mockPrimary, mockFallback)
	
	ctx := context.Background()
	reader := strings.NewReader("test content")
	
	// Primary fails
	mockPrimary.On("Upload", ctx, "test.txt", reader, "text/plain", int64(12)).
		Return("", assert.AnError)
	
	// Fallback succeeds
	mockFallback.On("Upload", ctx, "test.txt", reader, "text/plain", int64(12)).
		Return("https://fallback.com/test.txt", nil)
	
	url, err := service.Upload(ctx, "test.txt", reader, "text/plain", 12)
	
	require.NoError(t, err)
	assert.Equal(t, "https://fallback.com/test.txt", url)
	
	mockPrimary.AssertExpectations(t)
	mockFallback.AssertExpectations(t)
}

func TestStorageServiceWithFallback_Delete(t *testing.T) {
	mockPrimary := &MockStorageService{}
	mockFallback := &MockStorageService{}
	
	service := NewStorageServiceWithFallback(mockPrimary, mockFallback)
	
	ctx := context.Background()
	
	// Both succeed
	mockPrimary.On("Delete", ctx, "test.txt").Return(nil)
	mockFallback.On("Delete", ctx, "test.txt").Return(nil)
	
	err := service.Delete(ctx, "test.txt")
	
	require.NoError(t, err)
	
	mockPrimary.AssertExpectations(t)
	mockFallback.AssertExpectations(t)
}

func TestStorageServiceWithFallback_Delete_BothFail(t *testing.T) {
	mockPrimary := &MockStorageService{}
	mockFallback := &MockStorageService{}
	
	service := NewStorageServiceWithFallback(mockPrimary, mockFallback)
	
	ctx := context.Background()
	
	// Both fail
	mockPrimary.On("Delete", ctx, "test.txt").Return(assert.AnError)
	mockFallback.On("Delete", ctx, "test.txt").Return(assert.AnError)
	
	err := service.Delete(ctx, "test.txt")
	
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both storages failed")
	
	mockPrimary.AssertExpectations(t)
	mockFallback.AssertExpectations(t)
}

func TestStorageServiceWithFallback_GetURL(t *testing.T) {
	mockPrimary := &MockStorageService{}
	mockFallback := &MockStorageService{}
	
	service := NewStorageServiceWithFallback(mockPrimary, mockFallback)
	
	mockPrimary.On("GetURL", "test.txt").Return("https://primary.com/test.txt")
	
	url := service.GetURL("test.txt")
	
	assert.Equal(t, "https://primary.com/test.txt", url)
	
	mockPrimary.AssertExpectations(t)
	mockFallback.AssertNotCalled(t, "GetURL")
}

func TestStorageServiceWithFallback_Exists_PrimaryExists(t *testing.T) {
	mockPrimary := &MockStorageService{}
	mockFallback := &MockStorageService{}
	
	service := NewStorageServiceWithFallback(mockPrimary, mockFallback)
	
	ctx := context.Background()
	
	// Primary exists
	mockPrimary.On("Exists", ctx, "test.txt").Return(true, nil)
	
	exists, err := service.Exists(ctx, "test.txt")
	
	require.NoError(t, err)
	assert.True(t, exists)
	
	mockPrimary.AssertExpectations(t)
	mockFallback.AssertNotCalled(t, "Exists")
}

func TestStorageServiceWithFallback_Exists_FallbackExists(t *testing.T) {
	mockPrimary := &MockStorageService{}
	mockFallback := &MockStorageService{}
	
	service := NewStorageServiceWithFallback(mockPrimary, mockFallback)
	
	ctx := context.Background()
	
	// Primary doesn't exist
	mockPrimary.On("Exists", ctx, "test.txt").Return(false, nil)
	
	// Fallback exists
	mockFallback.On("Exists", ctx, "test.txt").Return(true, nil)
	
	exists, err := service.Exists(ctx, "test.txt")
	
	require.NoError(t, err)
	assert.True(t, exists)
	
	mockPrimary.AssertExpectations(t)
	mockFallback.AssertExpectations(t)
}