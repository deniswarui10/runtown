package config

import (
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Session  SessionConfig
	Email    EmailConfig
	Resend   ResendConfig
	Pesapal  PesapalConfig
	Paystack PaystackConfig
	R2       R2Config
}

type ServerConfig struct {
	Port string
	Host string
	Env  string
}

type DatabaseConfig struct {
	URL      string // Full database URL
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type SessionConfig struct {
	Secret string
}

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
}

type ResendConfig struct {
	APIKey    string
	FromEmail string
	FromName  string
}

type PesapalConfig struct {
	ConsumerKey    string
	ConsumerSecret string
	Environment    string
	CallbackURL    string
	IPNURL         string
}

type PaystackConfig struct {
	SecretKey   string
	PublicKey   string
	Environment string
	WebhookURL  string
	CallbackURL string
}

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	PublicURL       string
	Region          string
	Endpoint        string
}

func Load() (*Config, error) {
	// Load .env files if they exist (try .env.local first, then .env)
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load(".env")

	config := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Host: getEnv("HOST", "localhost"),
			Env:  getEnv("ENV", "development"),
		},
		Database: parseDatabaseConfig(),
		Session: SessionConfig{
			Secret: getEnv("SESSION_SECRET", "your-secret-key-change-in-production"),
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", "localhost"),
			SMTPPort:     getEnvAsInt("SMTP_PORT", 587),
			SMTPUser:     getEnv("SMTP_USER", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("FROM_EMAIL", "noreply@eventtickets.com"),
		},
		Resend: ResendConfig{
			APIKey:    getEnv("RESEND_API_KEY", ""),
			FromEmail: getEnv("RESEND_FROM_EMAIL", "noreply@eventtickets.com"),
			FromName:  getEnv("RESEND_FROM_NAME", "Event Ticketing Platform"),
		},
		Pesapal: PesapalConfig{
			ConsumerKey:    getEnv("PESAPAL_CONSUMER_KEY", ""),
			ConsumerSecret: getEnv("PESAPAL_CONSUMER_SECRET", ""),
			Environment:    getEnv("PESAPAL_ENVIRONMENT", "sandbox"),
			CallbackURL:    getEnv("PESAPAL_CALLBACK_URL", "http://localhost:8080/payment/callback"),
			IPNURL:         getEnv("PESAPAL_IPN_URL", "http://localhost:8080/payment/ipn"),
		},
		Paystack: PaystackConfig{
			SecretKey:   getEnv("PAYSTACK_SECRET_KEY", ""),
			PublicKey:   getEnv("PAYSTACK_PUBLIC_KEY", ""),
			Environment: getEnv("PAYSTACK_ENVIRONMENT", "test"),
			WebhookURL:  getEnv("PAYSTACK_WEBHOOK_URL", "http://localhost:8080/payment/paystack/webhook"),
			CallbackURL: getEnv("PAYSTACK_CALLBACK_URL", "http://localhost:8080/payment/paystack/callback"),
		},
		R2: R2Config{
			AccountID:       getEnv("R2_ACCOUNT_ID", ""),
			AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
			BucketName:      getEnv("R2_BUCKET_NAME", "event-images"),
			PublicURL:       getEnv("R2_PUBLIC_URL", ""),
			Region:          getEnv("R2_REGION", "auto"),
			Endpoint:        getEnv("R2_ENDPOINT", ""),
		},
	}

	return config, nil
}

func parseDatabaseConfig() DatabaseConfig {
	// Check if DATABASE_URL is provided
	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL != "" {
		return parseDatabaseURL(databaseURL)
	}

	// Fall back to individual environment variables
	return DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvAsInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", "event_ticketing"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

func parseDatabaseURL(databaseURL string) DatabaseConfig {
	config := DatabaseConfig{
		URL: databaseURL,
	}

	// Parse the URL
	u, err := url.Parse(databaseURL)
	if err != nil {
		// If parsing fails, return the URL as-is
		return config
	}

	// Extract components
	config.Host = u.Hostname()
	if u.Port() != "" {
		config.Port, _ = strconv.Atoi(u.Port())
	} else {
		config.Port = 5432 // Default PostgreSQL port
	}

	if u.User != nil {
		config.User = u.User.Username()
		config.Password, _ = u.User.Password()
	}

	// Remove leading slash from path to get database name
	config.DBName = strings.TrimPrefix(u.Path, "/")

	// Parse query parameters for SSL mode
	query := u.Query()
	config.SSLMode = query.Get("sslmode")
	if config.SSLMode == "" {
		config.SSLMode = "disable"
	}

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}