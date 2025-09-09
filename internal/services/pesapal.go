package services

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// PesapalConfig represents Pesapal payment service configuration
type PesapalConfig struct {
	ConsumerKey    string
	ConsumerSecret string
	Environment    string // "sandbox" or "production"
	CallbackURL    string
	IPNURL         string
}

// PesapalPaymentService handles payments via Pesapal API
type PesapalPaymentService struct {
	config PesapalConfig
	client *http.Client
	baseURL string
}

// NewPesapalPaymentService creates a new Pesapal payment service
func NewPesapalPaymentService(config PesapalConfig) *PesapalPaymentService {
	baseURL := "https://pay.pesapal.com/v3"
	if config.Environment == "sandbox" {
		baseURL = "https://cybqa.pesapal.com/pesapalv3"
	}

	return &PesapalPaymentService{
		config:  config,
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: baseURL,
	}
}

// PesapalAuthRequest represents authentication request
type PesapalAuthRequest struct {
	ConsumerKey    string `json:"consumer_key"`
	ConsumerSecret string `json:"consumer_secret"`
}

// PesapalAuthResponse represents authentication response
type PesapalAuthResponse struct {
	Token     string      `json:"token"`
	ExpiresIn int         `json:"expiresIn"`
	Error     interface{} `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
}

// PesapalSubmitOrderRequest represents order submission request
type PesapalSubmitOrderRequest struct {
	ID                string                    `json:"id"`
	Currency          string                    `json:"currency"`
	Amount            float64                   `json:"amount"`
	Description       string                    `json:"description"`
	CallbackURL       string                    `json:"callback_url"`
	NotificationID    string                    `json:"notification_id"`
	BillingAddress    PesapalBillingAddress     `json:"billing_address"`
}

// PesapalBillingAddress represents billing address
type PesapalBillingAddress struct {
	EmailAddress string `json:"email_address"`
	PhoneNumber  string `json:"phone_number"`
	CountryCode  string `json:"country_code"`
	FirstName    string `json:"first_name"`
	MiddleName   string `json:"middle_name"`
	LastName     string `json:"last_name"`
	Line1        string `json:"line_1"`
	Line2        string `json:"line_2"`
	City         string `json:"city"`
	State        string `json:"state"`
	PostalCode   string `json:"postal_code"`
}

// PesapalSubmitOrderResponse represents order submission response
type PesapalSubmitOrderResponse struct {
	OrderTrackingID string `json:"order_tracking_id"`
	MerchantReference string `json:"merchant_reference"`
	RedirectURL     string `json:"redirect_url"`
	Error           interface{} `json:"error,omitempty"`
	Message         string `json:"message,omitempty"`
}

// PesapalTransactionStatusResponse represents transaction status response
type PesapalTransactionStatusResponse struct {
	PaymentMethod     string    `json:"payment_method"`
	Amount            float64   `json:"amount"`
	CreatedDate       time.Time `json:"created_date"`
	ConfirmationCode  string    `json:"confirmation_code"`
	PaymentStatusDescription string `json:"payment_status_description"`
	Description       string    `json:"description"`
	MerchantReference string    `json:"merchant_reference"`
	PaymentAccount    string    `json:"payment_account"`
	CallbackURL       string    `json:"call_back_url"`
	StatusCode        int       `json:"status_code"`
	Currency          string    `json:"currency"`
	Error             interface{} `json:"error,omitempty"`
	Message           string    `json:"message,omitempty"`
}

// PesapalIPN represents Instant Payment Notification
type PesapalIPN struct {
	OrderTrackingID   string `json:"OrderTrackingId"`
	OrderMerchantReference string `json:"OrderMerchantReference"`
}

// ProcessPayment processes a payment via Pesapal
func (s *PesapalPaymentService) ProcessPayment(amount int, paymentMethod string, billingInfo PaymentBillingInfo) (*PaymentResult, error) {
	// Get authentication token
	token, err := s.authenticate()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Generate unique merchant reference
	merchantRef := fmt.Sprintf("ORD-%d-%d", time.Now().Unix(), amount)

	// Submit order to Pesapal
	orderRequest := PesapalSubmitOrderRequest{
		ID:          merchantRef,
		Currency:    "KES", // Kenyan Shillings - adjust as needed
		Amount:      float64(amount) / 100, // Convert from cents
		Description: "Event Ticket Purchase",
		CallbackURL: s.config.CallbackURL,
		BillingAddress: PesapalBillingAddress{
			EmailAddress: billingInfo.Email,
			PhoneNumber:  "+254700000000", // Default - should be from billing info
			CountryCode:  "KE",
			FirstName:    strings.Split(billingInfo.Name, " ")[0],
			LastName:     billingInfo.Name,
			Line1:        billingInfo.Address,
			City:         billingInfo.City,
			State:        billingInfo.State,
			PostalCode:   billingInfo.ZipCode,
		},
	}

	orderResponse, err := s.submitOrder(token, orderRequest)
	if err != nil {
		return nil, fmt.Errorf("order submission failed: %w", err)
	}

	// Return payment result with redirect URL
	return &PaymentResult{
		PaymentID:     orderResponse.OrderTrackingID,
		Status:        "pending", // Pesapal payments start as pending
		Amount:        amount,
		TransactionID: orderResponse.MerchantReference,
		ProcessedAt:   time.Now(),
	}, nil
}

// RefundPayment processes a refund (Note: Pesapal doesn't support automatic refunds via API)
func (s *PesapalPaymentService) RefundPayment(paymentID string, amount int) (*RefundResult, error) {
	// Pesapal doesn't support automatic refunds via API
	// This would typically require manual processing
	return &RefundResult{
		RefundID:     fmt.Sprintf("REF-%s-%d", paymentID, time.Now().Unix()),
		Status:       "pending",
		Amount:       amount,
		ProcessedAt:  time.Now(),
		ErrorMessage: "Refund initiated - manual processing required",
	}, nil
}

// GetPaymentStatus gets the status of a payment
func (s *PesapalPaymentService) GetPaymentStatus(paymentID string) (*PaymentStatus, error) {
	token, err := s.authenticate()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	status, err := s.getTransactionStatus(token, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction status: %w", err)
	}

	// Map Pesapal status to our status
	var paymentStatus string
	switch status.StatusCode {
	case 1:
		paymentStatus = "success"
	case 2:
		paymentStatus = "failed"
	case 0:
		paymentStatus = "pending"
	default:
		paymentStatus = "unknown"
	}

	return &PaymentStatus{
		PaymentID:     paymentID,
		Status:        paymentStatus,
		Amount:        int(status.Amount * 100), // Convert to cents
		TransactionID: status.ConfirmationCode,
		CreatedAt:     status.CreatedDate,
		UpdatedAt:     time.Now(),
	}, nil
}

// authenticate gets an authentication token from Pesapal with enhanced debugging
func (s *PesapalPaymentService) authenticate() (string, error) {
	fmt.Printf("🔐 Pesapal Authentication Debug:\n")
	fmt.Printf("   Environment: %s\n", s.config.Environment)
	fmt.Printf("   Base URL: %s\n", s.baseURL)
	fmt.Printf("   Consumer Key: %s (length: %d)\n", s.config.ConsumerKey, len(s.config.ConsumerKey))
	fmt.Printf("   Consumer Secret: %s... (length: %d)\n", s.config.ConsumerSecret[:min(10, len(s.config.ConsumerSecret))], len(s.config.ConsumerSecret))

	authRequest := PesapalAuthRequest{
		ConsumerKey:    s.config.ConsumerKey,
		ConsumerSecret: s.config.ConsumerSecret,
	}

	jsonData, err := json.Marshal(authRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal auth request: %w", err)
	}

	fmt.Printf("   Request JSON: %s\n", string(jsonData))

	authURL := s.baseURL + "/api/Auth/RequestToken"
	fmt.Printf("   Auth URL: %s\n", authURL)

	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	fmt.Printf("   Request Headers: %v\n", req.Header)

	resp, err := s.client.Do(req)
	if err != nil {
		fmt.Printf("   ❌ Request failed: %v\n", err)
		return "", fmt.Errorf("failed to send auth request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("   Response Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("   Response Headers: %v\n", resp.Header)

	// Read response body for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	fmt.Printf("   Response Body: %s\n", string(bodyBytes))

	// Parse the response
	var authResponse PesapalAuthResponse
	if err := json.Unmarshal(bodyBytes, &authResponse); err != nil {
		fmt.Printf("   ❌ Failed to parse JSON response: %v\n", err)
		return "", fmt.Errorf("failed to decode auth response: %w", err)
	}

	if authResponse.Error != nil {
		errorMsg := "unknown error"
		
		// Handle structured error response
		if errorMap, ok := authResponse.Error.(map[string]interface{}); ok {
			if code, exists := errorMap["code"]; exists {
				errorMsg = fmt.Sprintf("%v", code)
			}
			if errorType, exists := errorMap["error_type"]; exists {
				errorMsg = fmt.Sprintf("%v: %s", errorType, errorMsg)
			}
			if message, exists := errorMap["message"]; exists && message != "" {
				errorMsg = fmt.Sprintf("%s - %v", errorMsg, message)
			}
		} else if errStr, ok := authResponse.Error.(string); ok && errStr != "" {
			errorMsg = errStr
		} else if authResponse.Message != "" {
			errorMsg = authResponse.Message
		}
		
		fmt.Printf("   ❌ Authentication error: %s\n", errorMsg)
		return "", fmt.Errorf("authentication error: %s", errorMsg)
	}

	if authResponse.Token == "" {
		fmt.Printf("   ❌ Empty token received\n")
		return "", fmt.Errorf("received empty authentication token")
	}

	fmt.Printf("   ✅ Authentication successful, token length: %d\n", len(authResponse.Token))
	return authResponse.Token, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// submitOrder submits an order to Pesapal
func (s *PesapalPaymentService) submitOrder(token string, orderRequest PesapalSubmitOrderRequest) (*PesapalSubmitOrderResponse, error) {
	jsonData, err := json.Marshal(orderRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order request: %w", err)
	}

	req, err := http.NewRequest("POST", s.baseURL+"/api/Transactions/SubmitOrderRequest", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create order request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send order request: %w", err)
	}
	defer resp.Body.Close()

	var orderResponse PesapalSubmitOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResponse); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %w", err)
	}

	if orderResponse.Error != nil {
		errorMsg := "unknown error"
		if errStr, ok := orderResponse.Error.(string); ok && errStr != "" {
			errorMsg = errStr
		} else if orderResponse.Message != "" {
			errorMsg = orderResponse.Message
		}
		return nil, fmt.Errorf("order submission error: %s", errorMsg)
	}

	return &orderResponse, nil
}

// getTransactionStatus gets the status of a transaction
func (s *PesapalPaymentService) getTransactionStatus(token, orderTrackingID string) (*PesapalTransactionStatusResponse, error) {
	url := fmt.Sprintf("%s/api/Transactions/GetTransactionStatus?orderTrackingId=%s", s.baseURL, orderTrackingID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create status request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send status request: %w", err)
	}
	defer resp.Body.Close()

	var statusResponse PesapalTransactionStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResponse); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	if statusResponse.Error != nil {
		errorMsg := "unknown error"
		if errStr, ok := statusResponse.Error.(string); ok && errStr != "" {
			errorMsg = errStr
		} else if statusResponse.Message != "" {
			errorMsg = statusResponse.Message
		}
		return nil, fmt.Errorf("status request error: %s", errorMsg)
	}

	return &statusResponse, nil
}

// HandleIPN handles Instant Payment Notifications from Pesapal
func (s *PesapalPaymentService) HandleIPN(ipnData PesapalIPN) error {
	// Verify the IPN (in production, you should verify the signature)
	if ipnData.OrderTrackingID == "" {
		return fmt.Errorf("invalid IPN: missing order tracking ID")
	}

	// Get the current status of the transaction
	_, err := s.GetPaymentStatus(ipnData.OrderTrackingID)
	if err != nil {
		return fmt.Errorf("failed to get payment status for IPN: %w", err)
	}

	// Here you would typically update your database with the payment status
	// and trigger any necessary business logic (send confirmation emails, etc.)

	return nil
}

// GetRedirectURL returns the redirect URL for a payment
func (s *PesapalPaymentService) GetRedirectURL(orderTrackingID string) string {
	return fmt.Sprintf("%s/api/URLGeneration/SubmitOrderRequest?OrderTrackingId=%s", s.baseURL, orderTrackingID)
}

// TestConnection tests the Pesapal API connection
func (s *PesapalPaymentService) TestConnection() error {
	_, err := s.authenticate()
	if err != nil {
		return fmt.Errorf("failed to authenticate with Pesapal: %w", err)
	}
	return nil
}

// TestConnectionWithDiagnostics provides detailed diagnostic information
func (s *PesapalPaymentService) TestConnectionWithDiagnostics() (*ConnectionDiagnostics, error) {
	diagnostics := &ConnectionDiagnostics{
		Environment: s.config.Environment,
		Timestamp:   time.Now(),
	}

	// Test endpoint reachability
	fmt.Printf("🔍 Testing Pesapal endpoint reachability...\n")
	start := time.Now()
	resp, err := http.Get(s.baseURL)
	diagnostics.ResponseTime = time.Since(start)
	
	if err != nil {
		diagnostics.EndpointReachable = false
		diagnostics.LastError = fmt.Sprintf("Endpoint unreachable: %v", err)
		fmt.Printf("   ❌ Endpoint unreachable: %v\n", err)
	} else {
		diagnostics.EndpointReachable = true
		diagnostics.APIVersion = resp.Header.Get("X-API-Version")
		resp.Body.Close()
		fmt.Printf("   ✅ Endpoint reachable (status: %d, response time: %v)\n", resp.StatusCode, diagnostics.ResponseTime)
	}

	// Validate credentials format
	fmt.Printf("🔍 Validating credential format...\n")
	if s.config.ConsumerKey == "" {
		diagnostics.CredentialsValid = false
		diagnostics.LastError = "Consumer key is empty"
		fmt.Printf("   ❌ Consumer key is empty\n")
	} else if s.config.ConsumerSecret == "" {
		diagnostics.CredentialsValid = false
		diagnostics.LastError = "Consumer secret is empty"
		fmt.Printf("   ❌ Consumer secret is empty\n")
	} else {
		diagnostics.CredentialsValid = true
		fmt.Printf("   ✅ Credentials format valid\n")
	}

	// Test authentication
	fmt.Printf("🔍 Testing authentication...\n")
	token, err := s.authenticate()
	if err != nil {
		diagnostics.AuthenticationTest = false
		diagnostics.LastError = fmt.Sprintf("Authentication failed: %v", err)
		fmt.Printf("   ❌ Authentication failed: %v\n", err)
	} else {
		diagnostics.AuthenticationTest = true
		fmt.Printf("   ✅ Authentication successful (token length: %d)\n", len(token))
	}

	return diagnostics, nil
}

// ConnectionDiagnostics represents diagnostic information
type ConnectionDiagnostics struct {
	EndpointReachable  bool          `json:"endpoint_reachable"`
	AuthenticationTest bool          `json:"authentication_test"`
	CredentialsValid   bool          `json:"credentials_valid"`
	ResponseTime       time.Duration `json:"response_time"`
	LastError          string        `json:"last_error,omitempty"`
	APIVersion         string        `json:"api_version,omitempty"`
	Environment        string        `json:"environment"`
	Timestamp          time.Time     `json:"timestamp"`
}

// generateSignature generates a signature for Pesapal requests (if needed for older API versions)
func (s *PesapalPaymentService) generateSignature(params map[string]string) string {
	// Sort parameters
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build query string
	var queryParts []string
	for _, k := range keys {
		queryParts = append(queryParts, fmt.Sprintf("%s=%s", k, url.QueryEscape(params[k])))
	}
	queryString := strings.Join(queryParts, "&")

	// Create signature base string
	baseString := fmt.Sprintf("GET&%s&%s", 
		url.QueryEscape(s.baseURL), 
		url.QueryEscape(queryString))

	// Generate signature
	key := fmt.Sprintf("%s&", s.config.ConsumerSecret)
	h := sha1.New()
	h.Write([]byte(key + baseString))
	
	return fmt.Sprintf("%x", h.Sum(nil))
}