package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PaystackConfig represents Paystack payment service configuration
type PaystackConfig struct {
	SecretKey   string
	PublicKey   string
	Environment string // "test" or "live"
	WebhookURL  string
	CallbackURL string
}

// PaystackService handles payments via Paystack API
type PaystackService struct {
	config  PaystackConfig
	client  *http.Client
	baseURL string
}

// NewPaystackService creates a new Paystack payment service
func NewPaystackService(config PaystackConfig) *PaystackService {
	baseURL := "https://api.paystack.co"

	return &PaystackService{
		config:  config,
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: baseURL,
	}
}

// TransactionRequest represents a payment initialization request
type TransactionRequest struct {
	Email       string            `json:"email"`
	Amount      int               `json:"amount"`    // Amount in kobo (NGN) or cents
	Currency    string            `json:"currency"`  // NGN, GHS, KES, ZAR
	Reference   string            `json:"reference"` // Unique transaction reference
	CallbackURL string            `json:"callback_url"`
	Metadata    map[string]string `json:"metadata"`
	Channels    []string          `json:"channels"` // card, bank, ussd, mobile_money
}

// TransactionResponse represents the response from transaction initialization
type TransactionResponse struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    TransactionData `json:"data"`
}

// TransactionData contains the transaction initialization data
type TransactionData struct {
	AuthorizationURL string `json:"authorization_url"`
	AccessCode       string `json:"access_code"`
	Reference        string `json:"reference"`
}

// TransactionVerification represents transaction verification response
type TransactionVerification struct {
	Status  bool               `json:"status"`
	Message string             `json:"message"`
	Data    TransactionDetails `json:"data"`
}

// TransactionDetails contains detailed transaction information
type TransactionDetails struct {
	ID            int               `json:"id"`
	Domain        string            `json:"domain"`
	Status        string            `json:"status"`
	Reference     string            `json:"reference"`
	Amount        int               `json:"amount"`
	Currency      string            `json:"currency"`
	PaidAt        string            `json:"paid_at"`
	CreatedAt     string            `json:"created_at"`
	Channel       string            `json:"channel"`
	IPAddress     string            `json:"ip_address"`
	Authorization AuthorizationData `json:"authorization"`
	Customer      CustomerData      `json:"customer"`
	Metadata      map[string]string `json:"metadata"`
}

// AuthorizationData contains payment authorization details
type AuthorizationData struct {
	AuthorizationCode string `json:"authorization_code"`
	Bin               string `json:"bin"`
	Last4             string `json:"last4"`
	ExpMonth          string `json:"exp_month"`
	ExpYear           string `json:"exp_year"`
	Channel           string `json:"channel"`
	CardType          string `json:"card_type"`
	Bank              string `json:"bank"`
	CountryCode       string `json:"country_code"`
	Brand             string `json:"brand"`
	Reusable          bool   `json:"reusable"`
}

// CustomerData contains customer information
type CustomerData struct {
	ID           int    `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Email        string `json:"email"`
	CustomerCode string `json:"customer_code"`
	Phone        string `json:"phone"`
}

// PaystackError represents an error response from Paystack
type PaystackError struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func (e *PaystackError) Error() string {
	return fmt.Sprintf("Paystack Error: %s", e.Message)
}

// InitializeTransaction initializes a payment transaction with Paystack
func (s *PaystackService) InitializeTransaction(req *TransactionRequest) (*TransactionResponse, error) {
	fmt.Printf("💳 Paystack Transaction Debug:\n")
	fmt.Printf("   Environment: %s\n", s.config.Environment)
	fmt.Printf("   Base URL: %s\n", s.baseURL)
	fmt.Printf("   Email: %s\n", req.Email)
	fmt.Printf("   Amount: %d (%s)\n", req.Amount, req.Currency)
	fmt.Printf("   Reference: %s\n", req.Reference)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction request: %w", err)
	}

	fmt.Printf("   Request JSON: %s\n", string(jsonData))

	initURL := s.baseURL + "/transaction/initialize"
	httpReq, err := http.NewRequest("POST", initURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", "Bearer "+s.config.SecretKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	fmt.Printf("   Request Headers: %v\n", httpReq.Header)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		fmt.Printf("   ❌ Request failed: %v\n", err)
		return nil, fmt.Errorf("failed to send transaction request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("   Response Status: %d %s\n", resp.StatusCode, resp.Status)

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	fmt.Printf("   Response Body: %s\n", string(bodyBytes))

	// Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		return nil, s.handleAPIError(resp.StatusCode, bodyBytes)
	}

	// Parse successful response
	var transactionResp TransactionResponse
	if err := json.Unmarshal(bodyBytes, &transactionResp); err != nil {
		fmt.Printf("   ❌ Failed to parse JSON response: %v\n", err)
		return nil, fmt.Errorf("failed to decode transaction response: %w", err)
	}

	if !transactionResp.Status {
		fmt.Printf("   ❌ Transaction initialization failed: %s\n", transactionResp.Message)
		return nil, fmt.Errorf("transaction initialization failed: %s", transactionResp.Message)
	}

	fmt.Printf("   ✅ Transaction initialized successfully\n")
	fmt.Printf("   Authorization URL: %s\n", transactionResp.Data.AuthorizationURL)

	return &transactionResp, nil
}

// VerifyTransaction verifies a transaction with Paystack
func (s *PaystackService) VerifyTransaction(reference string) (*TransactionVerification, error) {
	fmt.Printf("🔍 Paystack Verification Debug:\n")
	fmt.Printf("   Reference: %s\n", reference)

	verifyURL := fmt.Sprintf("%s/transaction/verify/%s", s.baseURL, reference)
	httpReq, err := http.NewRequest("GET", verifyURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create verification request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", "Bearer "+s.config.SecretKey)
	httpReq.Header.Set("Accept", "application/json")

	fmt.Printf("   Verify URL: %s\n", verifyURL)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		fmt.Printf("   ❌ Verification request failed: %v\n", err)
		return nil, fmt.Errorf("failed to send verification request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("   Response Status: %d %s\n", resp.StatusCode, resp.Status)

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read verification response body: %w", err)
	}

	fmt.Printf("   Response Body: %s\n", string(bodyBytes))

	// Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		return nil, s.handleAPIError(resp.StatusCode, bodyBytes)
	}

	// Parse verification response
	var verification TransactionVerification
	if err := json.Unmarshal(bodyBytes, &verification); err != nil {
		fmt.Printf("   ❌ Failed to parse verification response: %v\n", err)
		return nil, fmt.Errorf("failed to decode verification response: %w", err)
	}

	if !verification.Status {
		fmt.Printf("   ❌ Transaction verification failed: %s\n", verification.Message)
		return nil, fmt.Errorf("transaction verification failed: %s", verification.Message)
	}

	fmt.Printf("   ✅ Transaction verified: %s\n", verification.Data.Status)
	fmt.Printf("   Amount: %d %s\n", verification.Data.Amount, verification.Data.Currency)
	fmt.Printf("   Channel: %s\n", verification.Data.Channel)

	return &verification, nil
}

// ProcessPayment implements the PaymentService interface
func (s *PaystackService) ProcessPayment(amount int, paymentMethod string, billingInfo PaymentBillingInfo) (*PaymentResult, error) {
	// Generate unique reference
	reference := s.generateReference()

	// Determine supported channels based on payment method
	channels := []string{"card", "bank", "ussd", "mobile_money"}
	if paymentMethod == "card" {
		channels = []string{"card"}
	} else if paymentMethod == "mobile_money" {
		channels = []string{"mobile_money"}
	}

	// Try to initialize transaction with supported currencies
	resp, err := s.initializeTransactionWithFallback(billingInfo.Email, amount, reference, channels, map[string]string{
		"customer_name": billingInfo.Name,
		"payment_type":  billingInfo.PaymentType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Paystack transaction: %w", err)
	}

	// Return payment result
	return &PaymentResult{
		PaymentID:        resp.Data.Reference,
		Status:           "pending", // Paystack payments start as pending
		Amount:           amount,
		TransactionID:    resp.Data.AccessCode,
		ProcessedAt:      time.Now(),
		AuthorizationURL: resp.Data.AuthorizationURL, // Include the authorization URL for redirect
	}, nil
}

// RefundPayment processes a refund (placeholder implementation)
func (s *PaystackService) RefundPayment(paymentID string, amount int) (*RefundResult, error) {
	// TODO: Implement Paystack refund API
	return &RefundResult{
		RefundID:    fmt.Sprintf("REF-%s-%d", paymentID, time.Now().Unix()),
		Status:      "pending",
		Amount:      amount,
		ProcessedAt: time.Now(),
	}, nil
}

// GetPaymentStatus gets the status of a payment
func (s *PaystackService) GetPaymentStatus(paymentID string) (*PaymentStatus, error) {
	verification, err := s.VerifyTransaction(paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify transaction: %w", err)
	}

	// Map Paystack status to our status
	var status string
	switch verification.Data.Status {
	case "success":
		status = "success"
	case "failed":
		status = "failed"
	case "abandoned":
		status = "failed"
	default:
		status = "pending"
	}

	return &PaymentStatus{
		PaymentID:     paymentID,
		Status:        status,
		Amount:        verification.Data.Amount,
		TransactionID: fmt.Sprintf("%d", verification.Data.ID),
		CreatedAt:     parsePaystackTime(verification.Data.CreatedAt),
		UpdatedAt:     time.Now(),
	}, nil
}

// TestConnection tests the Paystack API connection
func (s *PaystackService) TestConnection() error {
	// Test by trying to initialize a transaction with currency fallback
	reference := "test-" + fmt.Sprintf("%d", time.Now().Unix())

	_, err := s.initializeTransactionWithFallback(
		"test@example.com",
		100, // 1 unit of currency
		reference,
		nil, // no specific channels
		map[string]string{"test": "connection"},
	)

	if err != nil {
		return fmt.Errorf("failed to connect to Paystack: %w", err)
	}

	return nil
}

// Helper methods

// generateReference generates a unique transaction reference
func (s *PaystackService) generateReference() string {
	return fmt.Sprintf("TXN-%d-%d", time.Now().Unix(), time.Now().Nanosecond()%1000000)
}

// handleAPIError handles Paystack API errors
func (s *PaystackService) handleAPIError(statusCode int, body []byte) error {
	var paystackErr PaystackError
	if err := json.Unmarshal(body, &paystackErr); err != nil {
		return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
	}

	switch statusCode {
	case 400:
		return fmt.Errorf("bad request: %s", paystackErr.Message)
	case 401:
		return fmt.Errorf("unauthorized: check API keys - %s", paystackErr.Message)
	case 404:
		return fmt.Errorf("not found: %s", paystackErr.Message)
	case 422:
		return fmt.Errorf("validation error: %s", paystackErr.Message)
	default:
		return &paystackErr
	}
}

// parsePaystackTime parses Paystack timestamp format
func parsePaystackTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Now()
	}

	// Try different time formats that Paystack might use
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t
		}
	}

	// If parsing fails, return current time
	return time.Now()
}

// VerifyWebhookSignature verifies Paystack webhook signature
func (s *PaystackService) VerifyWebhookSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha512.New, []byte(s.config.SecretKey))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// getSupportedCurrency returns a currency that's likely to be supported
func (s *PaystackService) getSupportedCurrency() string {
	// For test environment, try currencies in order of likelihood to be enabled
	if s.config.Environment == "test" {
		// Most Paystack test accounts have KES enabled for Kenya
		// But if that fails, we can try others
		return "KES"
	}

	// For production, default to KES (Kenyan Shilling) as our primary currency
	return "KES"
}

// getSupportedCurrencies returns a list of currencies to try in order
func (s *PaystackService) getSupportedCurrencies() []string {
	return []string{
		"KES", // Kenyan Shilling (primary currency)
		"NGN", // Nigerian Naira
		"GHS", // Ghanaian Cedi
		"ZAR", // South African Rand
		"USD", // US Dollar (fallback for international accounts)
	}
}

// initializeTransactionWithFallback tries multiple currencies until one works
func (s *PaystackService) initializeTransactionWithFallback(email string, amount int, reference string, channels []string, metadata map[string]string) (*TransactionResponse, error) {
	currencies := s.getSupportedCurrencies()

	var lastErr error
	for _, currency := range currencies {
		fmt.Printf("   Trying currency: %s\n", currency)

		req := &TransactionRequest{
			Email:       email,
			Amount:      amount,
			Currency:    currency,
			Reference:   reference,
			CallbackURL: s.config.CallbackURL,
			Channels:    channels,
			Metadata:    metadata,
		}

		resp, err := s.InitializeTransaction(req)
		if err != nil {
			// If it's a currency error, try the next currency
			if strings.Contains(err.Error(), "Currency not supported") || strings.Contains(err.Error(), "unsupported_currency") {
				fmt.Printf("   Currency %s not supported, trying next...\n", currency)
				lastErr = err
				continue
			}
			// If it's a different error, return it immediately
			return nil, err
		}

		// Success with this currency
		fmt.Printf("   ✅ Successfully initialized with currency: %s\n", currency)
		return resp, nil
	}

	// If we get here, all currencies failed
	return nil, fmt.Errorf("no supported currency found. Last error: %w", lastErr)
}

// GetAuthorizationURL gets the authorization URL for a payment reference
func (s *PaystackService) GetAuthorizationURL(reference string) (string, error) {
	// For Paystack, we need to initialize the transaction to get the authorization URL
	// This is a helper method for cases where we need the URL separately
	return "", fmt.Errorf("authorization URL should be obtained during transaction initialization")
}
