package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/web/templates/pages"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
)

// PaymentHandler handles payment-related operations
type PaymentHandler struct {
	paymentService services.PaymentService
	orderService   services.OrderServiceInterface
	ticketService  services.TicketServiceInterface
	store          sessions.Store
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(paymentService services.PaymentService, orderService services.OrderServiceInterface, ticketService services.TicketServiceInterface, store sessions.Store) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		orderService:   orderService,
		ticketService:  ticketService,
		store:          store,
	}
}

// PaymentCallback handles payment callback from Pesapal
func (h *PaymentHandler) PaymentCallback(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	orderTrackingID := r.URL.Query().Get("OrderTrackingId")
	merchantReference := r.URL.Query().Get("OrderMerchantReference")

	if orderTrackingID == "" {
		log.Printf("Payment callback: missing OrderTrackingId")
		http.Error(w, "Missing order tracking ID", http.StatusBadRequest)
		return
	}

	log.Printf("Payment callback received: OrderTrackingId=%s, MerchantReference=%s", orderTrackingID, merchantReference)

	// Get payment status from Pesapal
	paymentStatus, err := h.paymentService.GetPaymentStatus(orderTrackingID)
	if err != nil {
		log.Printf("Payment callback: failed to get payment status: %v", err)
		http.Error(w, "Failed to verify payment status", http.StatusInternalServerError)
		return
	}

	log.Printf("Payment status: %s for OrderTrackingId=%s", paymentStatus.Status, orderTrackingID)

	// For successful payments, complete the order creation process
	if paymentStatus.Status == "success" {
		// Get session to retrieve pending order info
		session, err := h.store.Get(r, "session")
		if err != nil {
			log.Printf("Payment callback: failed to get session: %v", err)
			http.Redirect(w, r, "/payment/success?payment_id="+orderTrackingID, http.StatusSeeOther)
			return
		}

		// Check if we have pending payment info
		if pendingPaymentID, ok := session.Values["pending_payment_id"].(string); ok && pendingPaymentID == orderTrackingID {
			// We have matching pending payment, complete the order
			if err := h.completePendingOrder(session, orderTrackingID, paymentStatus); err != nil {
				log.Printf("Payment callback: failed to complete pending order: %v", err)
				http.Redirect(w, r, "/payment/failed?payment_id="+orderTrackingID, http.StatusSeeOther)
				return
			}

			// Clear pending payment info from session
			delete(session.Values, "pending_payment_id")
			delete(session.Values, "pending_cart")
			delete(session.Values, "pending_billing_email")
			delete(session.Values, "pending_billing_name")
			session.Save(r, w)

			// Redirect to success page
			http.Redirect(w, r, "/payment/success?payment_id="+orderTrackingID, http.StatusSeeOther)
			return
		}
	}

	// Determine redirect URL based on payment status
	var redirectURL string
	switch paymentStatus.Status {
	case "success":
		redirectURL = "/payment/success?payment_id=" + orderTrackingID
	case "failed":
		redirectURL = "/payment/failed?payment_id=" + orderTrackingID
	case "pending":
		redirectURL = "/payment/pending?payment_id=" + orderTrackingID
	default:
		redirectURL = "/payment/unknown?payment_id=" + orderTrackingID
	}

	// Redirect user to appropriate page
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// PaymentIPN handles Instant Payment Notifications from Pesapal
func (h *PaymentHandler) PaymentIPN(w http.ResponseWriter, r *http.Request) {
	// Parse IPN data
	var ipnData services.PesapalIPN
	if err := json.NewDecoder(r.Body).Decode(&ipnData); err != nil {
		log.Printf("Payment IPN: failed to decode IPN data: %v", err)
		http.Error(w, "Invalid IPN data", http.StatusBadRequest)
		return
	}

	log.Printf("Payment IPN received: OrderTrackingId=%s, MerchantReference=%s",
		ipnData.OrderTrackingID, ipnData.OrderMerchantReference)

	// Handle IPN through payment service
	if pesapalService, ok := h.paymentService.(*services.PesapalPaymentService); ok {
		if err := pesapalService.HandleIPN(ipnData); err != nil {
			log.Printf("Payment IPN: failed to handle IPN: %v", err)
			http.Error(w, "Failed to process IPN", http.StatusInternalServerError)
			return
		}
	}

	// Get updated payment status
	paymentStatus, err := h.paymentService.GetPaymentStatus(ipnData.OrderTrackingID)
	if err != nil {
		log.Printf("Payment IPN: failed to get payment status: %v", err)
		http.Error(w, "Failed to get payment status", http.StatusInternalServerError)
		return
	}

	// TODO: Find and update the corresponding order
	// This requires implementing order lookup by payment ID

	log.Printf("Payment IPN processed successfully for OrderTrackingId=%s, Status=%s",
		ipnData.OrderTrackingID, paymentStatus.Status)

	// Respond with success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("IPN processed successfully"))
}

// PaymentStatus displays payment status pages
func (h *PaymentHandler) PaymentStatus(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status") // success, failed, pending, unknown
	paymentID := r.URL.Query().Get("payment_id")

	var paymentStatus *services.PaymentStatus
	if paymentID != "" {
		var err error
		paymentStatus, err = h.paymentService.GetPaymentStatus(paymentID)
		if err != nil {
			log.Printf("Failed to get payment status: %v", err)
		}
	}

	// Render payment status page
	component := pages.PaymentStatusPage(status, paymentID, paymentStatus)
	component.Render(r.Context(), w)
}

// DiagnosePesapal provides diagnostic information about Pesapal integration
func (h *PaymentHandler) DiagnosePesapal(w http.ResponseWriter, r *http.Request) {
	// Check if payment service is Pesapal
	pesapalService, ok := h.paymentService.(*services.PesapalPaymentService)
	if !ok {
		http.Error(w, "Payment service is not Pesapal", http.StatusBadRequest)
		return
	}

	// Run diagnostics
	diagnostics, err := pesapalService.TestConnectionWithDiagnostics()
	if err != nil {
		http.Error(w, fmt.Sprintf("Diagnostic failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return diagnostic information as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(diagnostics)
}

// PaymentRedirect handles redirecting users to Paystack payment page
func (h *PaymentHandler) PaymentRedirect(w http.ResponseWriter, r *http.Request) {
	paymentID := r.URL.Query().Get("payment_id")
	if paymentID == "" {
		http.Error(w, "Missing payment ID", http.StatusBadRequest)
		return
	}

	fmt.Printf("🔄 Payment Redirect Debug:\n")
	fmt.Printf("   Payment ID: %s\n", paymentID)

	// Get session to retrieve payment details
	session, err := h.store.Get(r, "session")
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	// Check if we have pending payment info
	pendingPaymentID, ok := session.Values["pending_payment_id"].(string)
	if !ok {
		fmt.Printf("   ❌ No pending payment ID in session\n")
		fmt.Printf("   Session values: %v\n", session.Values)
		http.Error(w, "No pending payment found in session", http.StatusBadRequest)
		return
	}

	if pendingPaymentID != paymentID {
		fmt.Printf("   ❌ Payment ID mismatch: expected %s, got %s\n", pendingPaymentID, paymentID)
		http.Error(w, "Payment ID mismatch", http.StatusBadRequest)
		return
	}

	fmt.Printf("   ✅ Payment ID matches: %s\n", paymentID)

	// For Paystack, we need to re-initialize the transaction to get the authorization URL
	// This is because our current PaymentResult doesn't include the URL
	// In a production system, we'd store this URL properly

	// Get the billing info from session
	billingEmail, _ := session.Values["pending_billing_email"].(string)

	if billingEmail == "" {
		http.Error(w, "Missing billing information", http.StatusBadRequest)
		return
	}

	// Get the cart information from session to get the correct amount
	pendingCart, ok := session.Values["pending_cart"].(*models.Cart)
	if !ok {
		fmt.Printf("   ❌ No pending cart found in session\n")
		http.Error(w, "No cart information found", http.StatusBadRequest)
		return
	}

	// Calculate total amount from cart
	totalAmount := 0
	for _, item := range pendingCart.Items {
		totalAmount += item.Price * item.Quantity
	}

	fmt.Printf("   💰 Cart total amount: %d\n", totalAmount)

	// Debug: Check payment service type
	fmt.Printf("   🔍 Payment service type: %T\n", h.paymentService)

	// Check if we have the authorization URL stored in session
	if authURL, ok := session.Values["pending_authorization_url"].(string); ok && authURL != "" {
		fmt.Printf("   ✅ Found stored authorization URL: %s\n", authURL)
		http.Redirect(w, r, authURL, http.StatusSeeOther)
		return
	}

	// Fallback: Check if payment service is Paystack and try to re-initialize
	fmt.Printf("   🔍 No stored authorization URL, checking if payment service is Paystack...\n")
	if paystackService, ok := h.paymentService.(*services.PaystackService); ok {
		fmt.Printf("   ✅ Payment service is Paystack, attempting fallback re-initialization\n")
		// Generate a new reference to avoid duplicate reference error
		newReference := fmt.Sprintf("%s-retry-%d", paymentID, time.Now().Unix())

		req := &services.TransactionRequest{
			Email:       billingEmail,
			Amount:      totalAmount,
			Currency:    "KES",        // Use KES as default since it's working
			Reference:   newReference, // Use new reference to avoid duplicate error
			CallbackURL: "http://localhost:8080/payment/paystack/callback",
			Channels:    []string{"card", "bank", "ussd", "mobile_money"},
			Metadata: map[string]string{
				"customer_name": billingEmail, // Use email as fallback
				"payment_type":  "paystack",
				"original_ref":  paymentID, // Store original reference for tracking
			},
		}

		fmt.Printf("   🔄 Re-initializing with new reference: %s\n", newReference)
		fmt.Printf("   📝 Request: Email=%s, Amount=%d, Currency=%s, Reference=%s\n",
			req.Email, req.Amount, req.Currency, req.Reference)

		resp, err := paystackService.InitializeTransaction(req)
		if err != nil {
			fmt.Printf("   ❌ Failed to get authorization URL: %v\n", err)
			fmt.Printf("   📝 Request details: %+v\n", req)
			http.Error(w, fmt.Sprintf("Failed to initialize payment: %v", err), http.StatusInternalServerError)
			return
		}

		// Update session with new payment ID and authorization URL
		session.Values["pending_payment_id"] = newReference
		session.Values["pending_authorization_url"] = resp.Data.AuthorizationURL
		session.Save(r, w)

		fmt.Printf("   ✅ Redirecting to Paystack with new reference: %s\n", resp.Data.AuthorizationURL)
		http.Redirect(w, r, resp.Data.AuthorizationURL, http.StatusSeeOther)
		return
	} else {
		fmt.Printf("   ❌ Payment service is not Paystack: %T\n", h.paymentService)
	}

	// Fallback error
	fmt.Printf("   ❌ Reached fallback error\n")
	http.Error(w, "Payment service not available", http.StatusInternalServerError)
}

// InitiatePayment initiates a payment with Paystack (for testing)
func (h *PaymentHandler) InitiatePayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	amountStr := r.FormValue("amount")
	email := r.FormValue("email")
	name := r.FormValue("name")

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	// Create billing info
	billingInfo := services.PaymentBillingInfo{
		Email: email,
		Name:  name,
	}

	// Process payment
	result, err := h.paymentService.ProcessPayment(amount, "pesapal", billingInfo)
	if err != nil {
		log.Printf("Payment initiation failed: %v", err)
		http.Error(w, fmt.Sprintf("Payment failed: %v", err), http.StatusInternalServerError)
		return
	}

	// For Pesapal, we need to redirect to the payment URL
	if pesapalService, ok := h.paymentService.(*services.PesapalPaymentService); ok {
		redirectURL := pesapalService.GetRedirectURL(result.PaymentID)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	// Fallback response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// completePendingOrder completes a pending order after successful payment
func (h *PaymentHandler) completePendingOrder(session *sessions.Session, paymentID string, paymentStatus *services.PaymentStatus) error {
	// Get pending order info from session
	pendingCart, ok := session.Values["pending_cart"].(*models.Cart)
	if !ok {
		return fmt.Errorf("no pending cart found in session")
	}

	billingEmail, ok := session.Values["pending_billing_email"].(string)
	if !ok {
		return fmt.Errorf("no pending billing email found in session")
	}

	billingName, ok := session.Values["pending_billing_name"].(string)
	if !ok {
		return fmt.Errorf("no pending billing name found in session")
	}

	// Get user ID from session
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		// Try to convert from other types
		if userIDValue, exists := session.Values["user_id"]; exists {
			switch v := userIDValue.(type) {
			case float64:
				userID = int(v)
				ok = userID != 0
			case string:
				if parsedID, err := strconv.Atoi(v); err == nil {
					userID = parsedID
					ok = userID != 0
				}
			}
		}
		if !ok || userID == 0 {
			return fmt.Errorf("no user ID found in session")
		}
	}

	log.Printf("Completing pending order for payment %s, cart with %d items, user %d", paymentID, len(pendingCart.Items), userID)

	// Create order in database
	orderReq := &models.OrderCreateRequest{
		UserID:       userID,
		EventID:      pendingCart.EventID,
		TotalAmount:  pendingCart.TotalAmount,
		BillingEmail: billingEmail,
		BillingName:  billingName,
		Status:       models.OrderPending,
	}

	order, err := h.orderService.CreateOrder(orderReq)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	log.Printf("Created order %s (ID: %d) for user %d", order.OrderNumber, order.ID, userID)

	// Generate ticket data for order completion
	var ticketData []struct {
		TicketTypeID int
		QRCode       string
	}

	for _, item := range pendingCart.Items {
		for i := 0; i < item.Quantity; i++ {
			// Generate unique QR code for each ticket
			qrCode, err := h.generateQRCode(order.ID, item.TicketTypeID)
			if err != nil {
				return fmt.Errorf("failed to generate QR code: %w", err)
			}

			ticketData = append(ticketData, struct {
				TicketTypeID int
				QRCode       string
			}{
				TicketTypeID: item.TicketTypeID,
				QRCode:       qrCode,
			})
		}
	}

	// Use order service to complete the order (creates tickets and updates status)
	err = h.orderService.CompleteOrder(order.ID, paymentID, ticketData)
	if err != nil {
		return fmt.Errorf("failed to complete order: %w", err)
	}

	log.Printf("Order completed successfully: %s, amount: KES %.2f, tickets created: %d",
		order.OrderNumber, float64(paymentStatus.Amount)/100, len(ticketData))

	return nil
}

// Helper function to map payment status to order status
func mapPaymentStatusToOrderStatus(paymentStatus string) string {
	switch paymentStatus {
	case "success":
		return "completed"
	case "failed":
		return "failed"
	case "pending":
		return "pending"
	default:
		return "pending"
	}
}

// generateQRCode generates a unique QR code for a ticket
func (h *PaymentHandler) generateQRCode(orderID, ticketTypeID int) (string, error) {
	// Generate random bytes for uniqueness
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create QR code with order ID, ticket type ID, and random component
	timestamp := time.Now().Unix()
	qrData := fmt.Sprintf("TKT-%d-%d-%d-%s", orderID, ticketTypeID, timestamp, hex.EncodeToString(randomBytes))

	return qrData, nil
}
