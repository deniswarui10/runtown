package utils

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// PasswordHashConfig holds the configuration for password hashing
type PasswordHashConfig struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultPasswordHashConfig returns the default configuration for password hashing
func DefaultPasswordHashConfig() *PasswordHashConfig {
	return &PasswordHashConfig{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// HashPassword hashes a password using Argon2id
func HashPassword(password string) (string, error) {
	config := DefaultPasswordHashConfig()
	
	// Generate a random salt
	salt := make([]byte, config.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}
	
	// Hash the password
	hash := argon2.IDKey([]byte(password), salt, config.Iterations, config.Memory, config.Parallelism, config.KeyLength)
	
	// Encode the hash and salt
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)
	
	// Format: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		config.Memory, config.Iterations, config.Parallelism, encodedSalt, encodedHash), nil
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(password, hash string) (bool, error) {
	// Parse the hash
	config, salt, hashBytes, err := parseHash(hash)
	if err != nil {
		return false, fmt.Errorf("failed to parse hash: %w", err)
	}
	
	// Hash the provided password with the same parameters
	providedHash := argon2.IDKey([]byte(password), salt, config.Iterations, config.Memory, config.Parallelism, config.KeyLength)
	
	// Compare the hashes using constant-time comparison
	return subtle.ConstantTimeCompare(hashBytes, providedHash) == 1, nil
}

// parseHash parses an Argon2id hash string
func parseHash(hash string) (*PasswordHashConfig, []byte, []byte, error) {
	// Split the hash into parts
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, fmt.Errorf("invalid hash format: expected 6 parts, got %d", len(parts))
	}
	
	// Check format: ["", "argon2id", "v=19", "m=memory,t=iterations,p=parallelism", "salt", "hash"]
	if parts[1] != "argon2id" || parts[2] != "v=19" {
		return nil, nil, nil, fmt.Errorf("invalid hash format: incorrect prefix")
	}
	
	// Parse parameters
	var memory, iterations uint32
	var parallelism uint8
	n, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil || n != 3 {
		return nil, nil, nil, fmt.Errorf("invalid hash format: failed to parse parameters")
	}
	
	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode salt: %w", err)
	}
	
	hashBytes, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode hash: %w", err)
	}
	
	config := &PasswordHashConfig{
		Memory:      memory,
		Iterations:  iterations,
		Parallelism: parallelism,
		SaltLength:  uint32(len(salt)),
		KeyLength:   uint32(len(hashBytes)),
	}
	
	return config, salt, hashBytes, nil
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}