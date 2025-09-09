package services

import (
	"fmt"
	"log"
	"time"
	"event-ticketing-platform/internal/config"
)

// MockPaymentService provides a mock payment service that can optionally use Paystack or Pesapal
type MockPaymentService struct {
	pesapalService *PesapalPaymentService
	paystackService *PaystackService
	usePesapal     bool
	usePaystack    bool
}

// NewMockPaymentService creates a new mock payment service with Paystack support
func NewMockPaymentService(pesapalConfig *config.PesapalConfig, paystackConfig *config.PaystackConfig) *MockPaymentService {
	service := &MockPaymentService{
		usePesapal:  false,
		usePaystack: false,
	}
	
	// Prefer Paystack over Pesapal if both are configured
	if paystackConfig != nil && paystackConfig.SecretKey != "" && paystackConfig.PublicKey != "" {
		// Convert config types
		paystackServiceConfig := PaystackConfig{
			SecretKey:   paystackConfig.SecretKey,
			PublicKey:   paystackConfig.PublicKey,
			Environment: paystackConfig.Environment,
			WebhookURL:  paystackConfig.WebhookURL,
			CallbackURL: paystackConfig.CallbackURL,
		}
		service.paystackService = NewPaystackService(paystackServiceConfig)
		service.usePaystack = true
		log.Printf("Payment service: Using Paystack API (%s environment)", paystackConfig.Environment)
	} else if pesapalConfig != nil && pesapalConfig.ConsumerKey != "" && pesapalConfig.ConsumerSecret != "" {
		// Fallback to Pesapal if Paystack is not configured
		pesapalServiceConfig := PesapalConfig{
			ConsumerKey:    pesapalConfig.ConsumerKey,
			ConsumerSecret: pesapalConfig.ConsumerSecret,
			Environment:    pesapalConfig.Environment,
			CallbackURL:    pesapalConfig.CallbackURL,
			IPNURL:         pesapalConfig.IPNURL,
		}
		service.pesapalService = NewPesapalPaymentService(pesapalServiceConfig)
		service.usePesapal = true
		log.Printf("Payment service: Using Pesapal API (%s environment)", pesapalConfig.Environment)
	} else {
		log.Println("Payment service: Using mock (no Paystack or Pesapal credentials provided)")
	}
	
	return service
}

// ProcessPayment processes a payment
func (s *MockPaymentService) ProcessPayment(amount int, paymentMethod string, billingInfo PaymentBillingInfo) (*PaymentResult, error) {
	if s.usePaystack && s.paystackService != nil {
		return s.paystackService.ProcessPayment(amount, paymentMethod, billingInfo)
	} else if s.usePesapal && s.pesapalService != nil {
		return s.pesapalService.ProcessPayment(amount, paymentMethod, billingInfo)
	}
	
	// Mock implementation - simulate successful payment
	paymentID := fmt.Sprintf("mock_pay_%d_%d", time.Now().Unix(), amount)
	
	log.Printf("Mock Payment: Processing payment of $%.2f for %s", float64(amount)/100, billingInfo.Email)
	
	return &PaymentResult{
		PaymentID:     paymentID,
		Status:        "success",
		Amount:        amount,
		TransactionID: fmt.Sprintf("txn_%d", time.Now().Unix()),
		ProcessedAt:   time.Now(),
	}, nil
}

// RefundPayment processes a refund
func (s *MockPaymentService) RefundPayment(paymentID string, amount int) (*RefundResult, error) {
	if s.usePaystack && s.paystackService != nil {
		return s.paystackService.RefundPayment(paymentID, amount)
	} else if s.usePesapal && s.pesapalService != nil {
		return s.pesapalService.RefundPayment(paymentID, amount)
	}
	
	// Mock implementation - simulate successful refund
	refundID := fmt.Sprintf("mock_ref_%d_%d", time.Now().Unix(), amount)
	
	log.Printf("Mock Payment: Processing refund of $%.2f for payment %s", float64(amount)/100, paymentID)
	
	return &RefundResult{
		RefundID:    refundID,
		Status:      "success",
		Amount:      amount,
		ProcessedAt: time.Now(),
	}, nil
}

// GetPaymentStatus gets the status of a payment
func (s *MockPaymentService) GetPaymentStatus(paymentID string) (*PaymentStatus, error) {
	if s.usePaystack && s.paystackService != nil {
		return s.paystackService.GetPaymentStatus(paymentID)
	} else if s.usePesapal && s.pesapalService != nil {
		return s.pesapalService.GetPaymentStatus(paymentID)
	}
	
	// Mock implementation - return success status
	return &PaymentStatus{
		PaymentID:     paymentID,
		Status:        "success",
		Amount:        5000, // Mock amount
		TransactionID: fmt.Sprintf("txn_%s", paymentID),
		CreatedAt:     time.Now().Add(-1 * time.Hour),
		UpdatedAt:     time.Now(),
	}, nil
}

// TestConnection tests the payment service connection
func (s *MockPaymentService) TestConnection() error {
	if s.usePaystack && s.paystackService != nil {
		return s.paystackService.TestConnection()
	} else if s.usePesapal && s.pesapalService != nil {
		return s.pesapalService.TestConnection()
	}
	
	// Mock always works
	return nil
}