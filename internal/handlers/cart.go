package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/web/templates/pages"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
)

// CartHandler handles shopping cart and checkout requests
type CartHandler struct {
	ticketService  *services.TicketService
	eventService   services.EventServiceInterface
	paymentService services.PaymentService
	store          sessions.Store
}

// NewCartHandler creates a new cart handler
func NewCartHandler(
	ticketService *services.TicketService,
	eventService services.EventServiceInterface,
	paymentService services.PaymentService,
	store sessions.Store,
) *CartHandler {
	return &CartHandler{
		ticketService:  ticketService,
		eventService:   eventService,
		paymentService: paymentService,
		store:          store,
	}
}

// AddToCartUnified adds tickets to the shopping cart (unified endpoint that accepts event_id as form parameter)
func (h *CartHandler) AddToCartUnified(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get event ID from form data
	eventIDStr := r.FormValue("event_id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	ticketTypeIDStr := r.FormValue("ticket_type_id")
	ticketTypeID, err := strconv.Atoi(ticketTypeIDStr)
	if err != nil {
		http.Error(w, "Invalid ticket type ID", http.StatusBadRequest)
		return
	}

	quantityStr := r.FormValue("quantity")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity <= 0 {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	// Get ticket type details
	ticketTypes, err := h.ticketService.GetTicketTypesByEventID(eventID)
	if err != nil {
		http.Error(w, "Failed to get ticket types", http.StatusInternalServerError)
		return
	}

	var selectedTicketType *models.TicketType
	for _, tt := range ticketTypes {
		if tt.ID == ticketTypeID {
			selectedTicketType = tt
			break
		}
	}

	if selectedTicketType == nil {
		http.Error(w, "Ticket type not found", http.StatusNotFound)
		return
	}

	// Check availability
	if !selectedTicketType.IsAvailable() {
		http.Error(w, "Tickets are not available", http.StatusBadRequest)
		return
	}

	if quantity > selectedTicketType.Available() {
		http.Error(w, fmt.Sprintf("Only %d tickets available", selectedTicketType.Available()), http.StatusBadRequest)
		return
	}

	// Get or create cart from session
	session, err := h.store.Get(r, "session")
	if err != nil {
		h.handleSessionError(w, r, err)
		return
	}

	cart := h.getCartFromSession(session)

	// If cart is for a different event, clear it
	if cart.EventID != 0 && cart.EventID != eventID {
		cart = &models.Cart{}
	}

	// Set event info if new cart
	if cart.EventID == 0 {
		event, err := h.eventService.GetEventByID(eventID)
		if err != nil {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}
		cart.EventID = eventID
		cart.EventTitle = event.Title
	}

	// Add or update item in cart
	found := false
	for i := range cart.Items {
		if cart.Items[i].TicketTypeID == ticketTypeID {
			cart.Items[i].Quantity += quantity
			cart.Items[i].Subtotal = cart.Items[i].Price * cart.Items[i].Quantity
			found = true
			break
		}
	}

	if !found {
		cart.Items = append(cart.Items, models.CartItem{
			TicketTypeID: ticketTypeID,
			TicketName:   selectedTicketType.Name,
			Price:        selectedTicketType.Price,
			Quantity:     quantity,
			Subtotal:     selectedTicketType.Price * quantity,
		})
	}

	// Recalculate total
	cart.TotalAmount = 0
	for _, item := range cart.Items {
		cart.TotalAmount += item.Subtotal
	}

	// Set expiration (15 minutes from now)
	cart.ExpiresAt = time.Now().Add(15 * time.Minute).Unix()

	// Save cart to session
	h.saveCartToSession(session, cart)
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Return success response for HTMX
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	// Format cart total as currency
	totalFormatted := fmt.Sprintf("KES %.2f", float64(cart.TotalAmount)/100)

	// Return HTML response for HTMX
	fmt.Fprintf(w, `
		<div class="bg-green-50 border border-green-200 rounded-md p-3 mb-4">
			<div class="flex">
				<div class="flex-shrink-0">
					<svg class="h-5 w-5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
					</svg>
				</div>
				<div class="ml-3">
					<p class="text-sm font-medium text-green-800">
						✅ Added %d ticket(s) to cart
					</p>
					<p class="text-sm text-green-700">
						Cart total: %s
					</p>
				</div>
				<div class="ml-auto pl-3">
					<a href="/cart" class="text-sm font-medium text-green-800 hover:text-green-900 underline">
						View Cart
					</a>
				</div>
			</div>
		</div>
	`, quantity, totalFormatted)
}

// AddToCart adds tickets to the shopping cart (URL parameter version for /events/{id}/cart/add)
func (h *CartHandler) AddToCart(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get event ID from URL parameter
	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	ticketTypeIDStr := r.FormValue("ticket_type_id")
	ticketTypeID, err := strconv.Atoi(ticketTypeIDStr)
	if err != nil {
		http.Error(w, "Invalid ticket type ID", http.StatusBadRequest)
		return
	}

	quantityStr := r.FormValue("quantity")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity <= 0 {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	// Get ticket type details
	ticketTypes, err := h.ticketService.GetTicketTypesByEventID(eventID)
	if err != nil {
		http.Error(w, "Failed to get ticket types", http.StatusInternalServerError)
		return
	}

	var selectedTicketType *models.TicketType
	for _, tt := range ticketTypes {
		if tt.ID == ticketTypeID {
			selectedTicketType = tt
			break
		}
	}

	if selectedTicketType == nil {
		http.Error(w, "Ticket type not found", http.StatusNotFound)
		return
	}

	// Check availability
	if !selectedTicketType.IsAvailable() {
		http.Error(w, "Tickets are not available", http.StatusBadRequest)
		return
	}

	if quantity > selectedTicketType.Available() {
		http.Error(w, fmt.Sprintf("Only %d tickets available", selectedTicketType.Available()), http.StatusBadRequest)
		return
	}

	// Get or create cart from session
	session, err := h.store.Get(r, "session")
	if err != nil {
		h.handleSessionError(w, r, err)
		return
	}

	cart := h.getCartFromSession(session)

	// If cart is for a different event, clear it
	if cart.EventID != 0 && cart.EventID != eventID {
		cart = &models.Cart{}
	}

	// Set event info if new cart
	if cart.EventID == 0 {
		event, err := h.eventService.GetEventByID(eventID)
		if err != nil {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}
		cart.EventID = eventID
		cart.EventTitle = event.Title
	}

	// Add or update item in cart
	found := false
	for i := range cart.Items {
		if cart.Items[i].TicketTypeID == ticketTypeID {
			cart.Items[i].Quantity += quantity
			cart.Items[i].Subtotal = cart.Items[i].Price * cart.Items[i].Quantity
			found = true
			break
		}
	}

	if !found {
		cart.Items = append(cart.Items, models.CartItem{
			TicketTypeID: ticketTypeID,
			TicketName:   selectedTicketType.Name,
			Price:        selectedTicketType.Price,
			Quantity:     quantity,
			Subtotal:     selectedTicketType.Price * quantity,
		})
	}

	// Recalculate total
	cart.TotalAmount = 0
	for _, item := range cart.Items {
		cart.TotalAmount += item.Subtotal
	}

	// Set expiration (15 minutes from now)
	cart.ExpiresAt = time.Now().Add(15 * time.Minute).Unix()

	// Save cart to session
	h.saveCartToSession(session, cart)
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Return success response for HTMX
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	// Format cart total as currency
	totalFormatted := fmt.Sprintf("KES %.2f", float64(cart.TotalAmount)/100)

	// Return HTML response for HTMX
	fmt.Fprintf(w, `
		<div class="bg-green-50 border border-green-200 rounded-md p-3 mb-4">
			<div class="flex">
				<div class="flex-shrink-0">
					<svg class="h-5 w-5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
					</svg>
				</div>
				<div class="ml-3">
					<p class="text-sm font-medium text-green-800">
						✅ Added %d ticket(s) to cart
					</p>
					<p class="text-sm text-green-700">
						Cart total: %s
					</p>
				</div>
				<div class="ml-auto pl-3">
					<a href="/cart" class="text-sm font-medium text-green-800 hover:text-green-900 underline">
						View Cart
					</a>
				</div>
			</div>
		</div>
	`, quantity, totalFormatted)
}

// ViewCart displays the shopping cart
func (h *CartHandler) ViewCart(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		h.handleRedirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get cart from session
	session, err := h.store.Get(r, "session")
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	cart := h.getCartFromSession(session)

	// Check if cart is expired
	if cart.ExpiresAt > 0 && time.Now().Unix() > cart.ExpiresAt {
		// Clear expired cart
		cart = &models.Cart{}
		h.saveCartToSession(session, cart)
		session.Save(r, w)
	}

	// Render cart page
	component := pages.CartPage(user, cart, nil, make(map[string]string))
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render cart page", http.StatusInternalServerError)
		return
	}
}

// UpdateCartItem updates quantity of an item in the cart
func (h *CartHandler) UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	ticketTypeIDStr := r.FormValue("ticket_type_id")
	ticketTypeID, err := strconv.Atoi(ticketTypeIDStr)
	if err != nil {
		http.Error(w, "Invalid ticket type ID", http.StatusBadRequest)
		return
	}

	quantityStr := r.FormValue("quantity")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity < 0 {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	// Get cart from session
	session, err := h.store.Get(r, "session")
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	cart := h.getCartFromSession(session)

	// Update or remove item
	for i := range cart.Items {
		if cart.Items[i].TicketTypeID == ticketTypeID {
			if quantity == 0 {
				// Remove item
				cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			} else {
				// Update quantity
				cart.Items[i].Quantity = quantity
				cart.Items[i].Subtotal = cart.Items[i].Price * quantity
			}
			break
		}
	}

	// Recalculate total
	cart.TotalAmount = 0
	for _, item := range cart.Items {
		cart.TotalAmount += item.Subtotal
	}

	// Save cart to session
	h.saveCartToSession(session, cart)
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Return updated cart partial for HTMX
	component := pages.CartItemsPartial(cart)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render cart items", http.StatusInternalServerError)
		return
	}
}

// CheckoutPage displays the checkout form
func (h *CartHandler) CheckoutPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		h.handleRedirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get cart from session
	session, err := h.store.Get(r, "session")
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	cart := h.getCartFromSession(session)

	// Check if cart is empty or expired
	if len(cart.Items) == 0 {
		h.handleRedirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	if cart.ExpiresAt > 0 && time.Now().Unix() > cart.ExpiresAt {
		// Clear expired cart
		cart = &models.Cart{}
		h.saveCartToSession(session, cart)
		session.Save(r, w)
		h.handleRedirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	// Pre-fill form with user data
	formData := map[string]string{
		"billing_email": user.Email,
		"billing_name":  fmt.Sprintf("%s %s", user.FirstName, user.LastName),
	}

	// Render checkout page
	component := pages.CheckoutPage(user, cart, nil, formData)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render checkout page", http.StatusInternalServerError)
		return
	}
}

// ProcessCheckout processes the checkout and payment
func (h *CartHandler) ProcessCheckout(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get cart from session
	session, err := h.store.Get(r, "session")
	if err != nil {
		h.handleSessionError(w, r, err)
		return
	}

	cart := h.getCartFromSession(session)

	// Validate cart
	if len(cart.Items) == 0 {
		h.handleRedirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	// Debug: Log all form values
	fmt.Printf("🛒 Checkout Form Debug:\n")
	fmt.Printf("   All form values: %v\n", r.Form)

	// Extract form data
	billingEmail := strings.TrimSpace(r.FormValue("billing_email"))
	billingName := strings.TrimSpace(r.FormValue("billing_name"))
	paymentMethod := r.FormValue("payment_method")

	fmt.Printf("   Extracted values:\n")
	fmt.Printf("     billing_email: '%s'\n", billingEmail)
	fmt.Printf("     billing_name: '%s'\n", billingName)
	fmt.Printf("     payment_method: '%s'\n", paymentMethod)

	// Validate form data
	errors := make(map[string][]string)
	formData := map[string]string{
		"billing_email":  billingEmail,
		"billing_name":   billingName,
		"payment_method": paymentMethod,
	}

	if billingEmail == "" {
		errors["billing_email"] = []string{"Billing email is required"}
	} else if !validateEmail(billingEmail) {
		errors["billing_email"] = []string{"Please enter a valid email address"}
	}
	if billingName == "" {
		errors["billing_name"] = []string{"Billing name is required"}
	}
	if paymentMethod == "" {
		errors["payment_method"] = []string{"Payment method is required"}
	}

	fmt.Printf("   Validation errors: %v\n", errors)

	if len(errors) > 0 {
		h.handleCheckoutError(w, r, errors, formData, user, cart)
		return
	}

	// Convert cart items to ticket selections
	var ticketSelections []services.TicketSelection
	for _, item := range cart.Items {
		ticketSelections = append(ticketSelections, services.TicketSelection{
			TicketTypeID: item.TicketTypeID,
			Quantity:     item.Quantity,
		})
	}

	// Create purchase request
	purchaseReq := &services.TicketPurchaseRequest{
		EventID:          cart.EventID,
		TicketSelections: ticketSelections,
		BillingInfo: services.PaymentBillingInfo{
			Email:       billingEmail,
			Name:        billingName,
			PaymentType: paymentMethod,
		},
		PaymentMethod: paymentMethod,
		UserID:        user.ID,
	}

	// Handle Paystack payment differently (redirect-based)
	if paymentMethod == "paystack" {
		// For Paystack, we need to initiate payment and redirect
		totalAmount := 0
		for _, item := range cart.Items {
			totalAmount += item.Price * item.Quantity
		}

		fmt.Printf("   💳 Processing Paystack payment for amount: %d\n", totalAmount)

		// Process payment with Paystack (this will return a pending status)
		paymentResult, err := h.paymentService.ProcessPayment(
			totalAmount,
			paymentMethod,
			services.PaymentBillingInfo{
				Email:       billingEmail,
				Name:        billingName,
				PaymentType: paymentMethod,
			},
		)
		if err != nil {
			fmt.Printf("   ❌ Paystack payment failed: %v\n", err)
			errors["general"] = []string{fmt.Sprintf("Payment initiation failed: %s", err.Error())}
			h.handleCheckoutError(w, r, errors, formData, user, cart)
			return
		}

		fmt.Printf("   ✅ Paystack payment initialized: %s\n", paymentResult.PaymentID)

		// Store cart and payment info in session for callback processing
		session.Values["pending_payment_id"] = paymentResult.PaymentID
		session.Values["pending_cart"] = cart
		session.Values["pending_billing_email"] = billingEmail
		session.Values["pending_billing_name"] = billingName
		session.Values["pending_authorization_url"] = paymentResult.AuthorizationURL // Store the authorization URL

		// Debug: Print session data before saving
		fmt.Printf("   💾 Saving session data:\n")
		fmt.Printf("      - pending_payment_id: %s\n", paymentResult.PaymentID)
		fmt.Printf("      - pending_billing_email: %s\n", billingEmail)
		fmt.Printf("      - pending_cart items: %d\n", len(cart.Items))
		fmt.Printf("      - pending_authorization_url: %s\n", paymentResult.AuthorizationURL)

		// Save session with error handling
		if err := session.Save(r, w); err != nil {
			fmt.Printf("   ❌ Failed to save session: %v\n", err)
			h.handleSessionError(w, r, err)
			return
		}

		fmt.Printf("   ✅ Session saved successfully\n")

		// Check if we have the authorization URL directly
		if paymentResult.AuthorizationURL != "" {
			fmt.Printf("   🔄 Redirecting to Paystack: %s\n", paymentResult.AuthorizationURL)

			// Handle HTMX requests differently to avoid CORS issues
			if middleware.IsHTMXRequest(r) {
				// For HTMX requests, use HX-Redirect header to trigger client-side redirect
				w.Header().Set("HX-Redirect", paymentResult.AuthorizationURL)
				w.WriteHeader(http.StatusOK)
				fmt.Printf("   ✅ HTMX redirect header set\n")
				return
			} else {
				// For regular requests, use standard HTTP redirect
				http.Redirect(w, r, paymentResult.AuthorizationURL, http.StatusSeeOther)
				fmt.Printf("   ✅ Standard HTTP redirect\n")
				return
			}
		}

		// Fallback to payment redirect page (shouldn't be needed now)
		redirectURL := fmt.Sprintf("/payment/redirect?payment_id=%s", paymentResult.PaymentID)
		fmt.Printf("   🔄 Fallback redirect to: %s\n", redirectURL)

		if middleware.IsHTMXRequest(r) {
			w.Header().Set("HX-Redirect", redirectURL)
			w.WriteHeader(http.StatusOK)
		} else {
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		}
		return
	}

	// For other payment methods (Stripe, PayPal), use the existing synchronous flow
	result, err := h.ticketService.PurchaseTickets(purchaseReq)
	if err != nil {
		errors["general"] = []string{fmt.Sprintf("Purchase failed: %s", err.Error())}
		h.handleCheckoutError(w, r, errors, formData, user, cart)
		return
	}

	// Clear cart after successful purchase
	cart = &models.Cart{}
	h.saveCartToSession(session, cart)
	session.Save(r, w)

	// Redirect to order confirmation
	confirmationURL := fmt.Sprintf("/orders/%d/confirmation", result.Order.ID)
	h.handleRedirect(w, r, confirmationURL, http.StatusSeeOther)
}

// ClearCart clears the shopping cart
func (h *CartHandler) ClearCart(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Get cart from session
	session, err := h.store.Get(r, "session")
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	// Clear cart
	cart := &models.Cart{}
	h.saveCartToSession(session, cart)
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Redirect to cart page
	h.handleRedirect(w, r, "/cart", http.StatusSeeOther)
}

// Helper methods

func (h *CartHandler) getCartFromSession(session *sessions.Session) *models.Cart {
	cartData, ok := session.Values["cart"]
	if !ok {
		return &models.Cart{}
	}

	cartJSON, ok := cartData.(string)
	if !ok {
		return &models.Cart{}
	}

	var cart models.Cart
	if err := json.Unmarshal([]byte(cartJSON), &cart); err != nil {
		return &models.Cart{}
	}

	return &cart
}

func (h *CartHandler) saveCartToSession(session *sessions.Session, cart *models.Cart) {
	cartJSON, err := json.Marshal(cart)
	if err != nil {
		return
	}
	session.Values["cart"] = string(cartJSON)
}

// validateEmail validates email format
func validateEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// handleCheckoutError returns appropriate error response based on request type
func (h *CartHandler) handleCheckoutError(w http.ResponseWriter, r *http.Request, errors map[string][]string, formData map[string]string, user *models.User, cart *models.Cart) {
	component := pages.CheckoutPage(user, cart, errors, formData)
	w.WriteHeader(http.StatusUnprocessableEntity)
	err := component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render checkout page", http.StatusInternalServerError)
	}
}

// handleRedirect handles redirects appropriately for HTMX vs regular requests
func (h *CartHandler) handleRedirect(w http.ResponseWriter, r *http.Request, url string, statusCode int) {
	if middleware.IsHTMXRequest(r) {
		w.Header().Set("HX-Redirect", url)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, url, statusCode)
	}
}

// handleSessionError handles session errors appropriately for HTMX vs regular requests
func (h *CartHandler) handleSessionError(w http.ResponseWriter, r *http.Request, err error) {
	if middleware.IsHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`
			<div class="bg-red-50 border border-red-200 text-red-800 p-4 rounded-lg">
				<div class="flex">
					<div class="flex-shrink-0">
						<svg class="h-5 w-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
						</svg>
					</div>
					<div class="ml-3">
						<p class="text-sm">Session error. Please refresh the page and try again.</p>
					</div>
				</div>
			</div>
		`))
	} else {
		http.Error(w, "Session error", http.StatusInternalServerError)
	}
}
