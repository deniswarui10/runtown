package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "simple password",
			password: "password123",
		},
		{
			name:     "complex password",
			password: "MyC0mpl3x!P@ssw0rd",
		},
		{
			name:     "long password",
			password: strings.Repeat("a", 100),
		},
		{
			name:     "password with special characters",
			password: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
		{
			name:     "unicode password",
			password: "пароль123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			
			require.NoError(t, err)
			assert.NotEmpty(t, hash)
			
			// Check that hash starts with expected format
			assert.True(t, strings.HasPrefix(hash, "$argon2id$v=19$"))
			
			// Check that hash is different from password
			assert.NotEqual(t, tt.password, hash)
			
			// Check that hashing the same password twice produces different hashes
			hash2, err := HashPassword(tt.password)
			require.NoError(t, err)
			assert.NotEqual(t, hash, hash2)
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	tests := []struct {
		name           string
		password       string
		hash           string
		expectedResult bool
		expectedError  bool
	}{
		{
			name:           "correct password",
			password:       password,
			hash:           hash,
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:           "incorrect password",
			password:       "wrongpassword",
			hash:           hash,
			expectedResult: false,
			expectedError:  false,
		},
		{
			name:           "empty password",
			password:       "",
			hash:           hash,
			expectedResult: false,
			expectedError:  false,
		},
		{
			name:          "invalid hash format",
			password:      password,
			hash:          "invalid-hash",
			expectedError: true,
		},
		{
			name:          "empty hash",
			password:      password,
			hash:          "",
			expectedError: true,
		},
		{
			name:          "malformed hash - missing parts",
			password:      password,
			hash:          "$argon2id$v=19$m=65536",
			expectedError: true,
		},
		{
			name:          "malformed hash - invalid base64",
			password:      password,
			hash:          "$argon2id$v=19$m=65536,t=3,p=2$invalid-base64$invalid-base64",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := VerifyPassword(tt.password, tt.hash)
			
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestPasswordHashAndVerifyIntegration(t *testing.T) {
	passwords := []string{
		"simple",
		"password123",
		"MyC0mpl3x!P@ssw0rd",
		"!@#$%^&*()_+-=[]{}|;':\",./<>?",
		"пароль123",
		strings.Repeat("a", 128), // max length password
	}

	for _, password := range passwords {
		t.Run("password_"+password[:min(len(password), 10)], func(t *testing.T) {
			// Hash the password
			hash, err := HashPassword(password)
			require.NoError(t, err)
			
			// Verify correct password
			valid, err := VerifyPassword(password, hash)
			require.NoError(t, err)
			assert.True(t, valid)
			
			// Verify incorrect password
			valid, err = VerifyPassword(password+"wrong", hash)
			require.NoError(t, err)
			assert.False(t, valid)
		})
	}
}

func TestGenerateSecureToken(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "small token",
			length: 8,
		},
		{
			name:   "medium token",
			length: 32,
		},
		{
			name:   "large token",
			length: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateSecureToken(tt.length)
			
			require.NoError(t, err)
			assert.NotEmpty(t, token)
			
			// Generate another token and ensure they're different
			token2, err := GenerateSecureToken(tt.length)
			require.NoError(t, err)
			assert.NotEqual(t, token, token2)
			
			// Check that token is base64 URL encoded
			assert.NotContains(t, token, "+")
			assert.NotContains(t, token, "/")
		})
	}
}

func TestGenerateSecureTokenZeroLength(t *testing.T) {
	token, err := GenerateSecureToken(0)
	require.NoError(t, err)
	// Zero length bytes will produce empty base64 string
	assert.Empty(t, token)
}

func TestParseHash(t *testing.T) {
	// Create a valid hash for testing
	validHash := "$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$aGFzaA"
	
	tests := []struct {
		name          string
		hash          string
		expectedError bool
	}{
		{
			name:          "valid hash",
			hash:          validHash,
			expectedError: false,
		},
		{
			name:          "invalid format - not argon2id",
			hash:          "$bcrypt$v=19$m=65536,t=3,p=2$c2FsdA$aGFzaA",
			expectedError: true,
		},
		{
			name:          "invalid format - missing parts",
			hash:          "$argon2id$v=19$m=65536",
			expectedError: true,
		},
		{
			name:          "invalid format - wrong version",
			hash:          "$argon2id$v=18$m=65536,t=3,p=2$c2FsdA$aGFzaA",
			expectedError: true,
		},
		{
			name:          "invalid base64 salt",
			hash:          "$argon2id$v=19$m=65536,t=3,p=2$invalid-base64$aGFzaA",
			expectedError: true,
		},
		{
			name:          "invalid base64 hash",
			hash:          "$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$invalid-base64",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, salt, hashBytes, err := parseHash(tt.hash)
			
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, config)
				assert.Nil(t, salt)
				assert.Nil(t, hashBytes)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.NotNil(t, salt)
				assert.NotNil(t, hashBytes)
				assert.Equal(t, uint32(65536), config.Memory)
				assert.Equal(t, uint32(3), config.Iterations)
				assert.Equal(t, uint8(2), config.Parallelism)
			}
		})
	}
}

func TestDefaultPasswordHashConfig(t *testing.T) {
	config := DefaultPasswordHashConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, uint32(64*1024), config.Memory)
	assert.Equal(t, uint32(3), config.Iterations)
	assert.Equal(t, uint8(2), config.Parallelism)
	assert.Equal(t, uint32(16), config.SaltLength)
	assert.Equal(t, uint32(32), config.KeyLength)
}

// Helper function for Go versions that don't have min built-in
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}