package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/web/templates/pages"
)

// WithdrawalHandler handles withdrawal-related requests
type WithdrawalHandler struct {
	withdrawalService *services.WithdrawalService
}

// NewWithdrawalHandler creates a new withdrawal handler
func NewWithdrawalHandler(withdrawalService *services.WithdrawalService) *WithdrawalHandler {
	return &WithdrawalHandler{
		withdrawalService: withdrawalService,
	}
}

// WithdrawalsPage displays the organizer's withdrawals
func (h *WithdrawalHandler) WithdrawalsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if user.Role != models.UserRoleOrganizer && user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get page parameter
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	// Get withdrawals
	withdrawals, totalCount, err := h.withdrawalService.GetOrganizerWithdrawals(user.ID, page, 10)
	if err != nil {
		http.Error(w, "Failed to load withdrawals", http.StatusInternalServerError)
		return
	}

	// Get available balance
	availableBalance, err := h.withdrawalService.GetOrganizerBalance(user.ID)
	if err != nil {
		http.Error(w, "Failed to load balance", http.StatusInternalServerError)
		return
	}

	// Calculate pagination
	totalPages := (totalCount + 9) / 10
	paginationInfo := map[string]interface{}{
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"TotalCount":  totalCount,
		"HasPrev":     page > 1,
		"HasNext":     page < totalPages,
		"PrevPage":    page - 1,
		"NextPage":    page + 1,
	}

	// Render withdrawals page
	component := pages.WithdrawalsPage(user, withdrawals, availableBalance, paginationInfo)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// CreateWithdrawalPage displays the withdrawal creation form
func (h *WithdrawalHandler) CreateWithdrawalPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if user.Role != models.UserRoleOrganizer && user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get available balance
	availableBalance, err := h.withdrawalService.GetOrganizerBalance(user.ID)
	if err != nil {
		http.Error(w, "Failed to load balance", http.StatusInternalServerError)
		return
	}

	// Render create withdrawal page
	component := pages.CreateWithdrawalPage(user, availableBalance, nil, nil)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// CreateWithdrawalSubmit handles withdrawal creation
func (h *WithdrawalHandler) CreateWithdrawalSubmit(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if user.Role != models.UserRoleOrganizer && user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get form values
	amountStr := r.FormValue("amount")
	reason := r.FormValue("reason")
	bankDetails := r.FormValue("bank_details")

	// Parse amount
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		amount = 0
	}

	// Create request
	req := &models.WithdrawalCreateRequest{
		Amount:      amount,
		Reason:      reason,
		BankDetails: bankDetails,
	}

	// Validate
	errors := make(map[string]string)
	if amount <= 0 {
		errors["amount"] = "Amount must be greater than 0"
	}
	if amount < 10 {
		errors["amount"] = "Minimum withdrawal amount is $10"
	}
	if reason == "" {
		errors["reason"] = "Reason is required"
	}
	if bankDetails == "" {
		errors["bank_details"] = "Bank details are required"
	}

	if len(errors) > 0 {
		// Get available balance for re-rendering
		availableBalance, _ := h.withdrawalService.GetOrganizerBalance(user.ID)
		
		formData := map[string]interface{}{
			"amount":       amountStr,
			"reason":       reason,
			"bank_details": bankDetails,
		}
		
		component := pages.CreateWithdrawalPage(user, availableBalance, formData, errors)
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// Create withdrawal
	_, err = h.withdrawalService.CreateWithdrawal(user.ID, req)
	if err != nil {
		// Get available balance for re-rendering
		availableBalance, _ := h.withdrawalService.GetOrganizerBalance(user.ID)
		
		formData := map[string]interface{}{
			"amount":       amountStr,
			"reason":       reason,
			"bank_details": bankDetails,
		}
		
		errors["general"] = err.Error()
		component := pages.CreateWithdrawalPage(user, availableBalance, formData, errors)
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// Redirect to withdrawals page
	http.Redirect(w, r, "/organizer/withdrawals", http.StatusSeeOther)
}

// AdminWithdrawalsPage displays all withdrawals for admin
func (h *WithdrawalHandler) AdminWithdrawalsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get query parameters
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	status := r.URL.Query().Get("status")

	// Get withdrawals
	withdrawals, totalCount, err := h.withdrawalService.GetAllWithdrawals(page, 20, status)
	if err != nil {
		http.Error(w, "Failed to load withdrawals", http.StatusInternalServerError)
		return
	}

	// Calculate pagination
	totalPages := (totalCount + 19) / 20
	paginationInfo := map[string]interface{}{
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"TotalCount":  totalCount,
		"HasPrev":     page > 1,
		"HasNext":     page < totalPages,
		"PrevPage":    page - 1,
		"NextPage":    page + 1,
	}

	// Render admin withdrawals page
	component := pages.AdminWithdrawalsPage(user, withdrawals, paginationInfo, status)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// UpdateWithdrawalStatus handles withdrawal status updates (admin only)
func (h *WithdrawalHandler) UpdateWithdrawalStatus(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get withdrawal ID
	withdrawalID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid withdrawal ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	status := models.WithdrawalStatus(r.FormValue("status"))
	adminNotes := r.FormValue("admin_notes")

	// Update status
	err = h.withdrawalService.UpdateWithdrawalStatus(withdrawalID, status, adminNotes)
	if err != nil {
		http.Error(w, "Failed to update withdrawal status", http.StatusInternalServerError)
		return
	}

	// Redirect back to admin withdrawals
	http.Redirect(w, r, "/admin/withdrawals", http.StatusSeeOther)
}