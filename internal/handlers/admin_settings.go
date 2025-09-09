package handlers

import (
	"net/http"
	"strconv"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/web/templates/pages"
)

// AdminSettingsHandler handles admin settings management
type AdminSettingsHandler struct {
	settingsService *services.SettingsService
}

// NewAdminSettingsHandler creates a new admin settings handler
func NewAdminSettingsHandler(settingsService *services.SettingsService) *AdminSettingsHandler {
	return &AdminSettingsHandler{
		settingsService: settingsService,
	}
}

// SettingsPage displays the admin settings page
func (h *AdminSettingsHandler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get current settings
	settings, err := h.settingsService.GetSettings()
	if err != nil {
		http.Error(w, "Failed to load settings", http.StatusInternalServerError)
		return
	}

	// Render settings page
	component := pages.AdminSettingsPage(user, settings, nil, nil)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// UpdateSettings handles settings update form submission
func (h *AdminSettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Build update request
	req := &models.SettingsUpdateRequest{}
	errors := make(map[string]string)

	// Parse platform fee percentage
	if feeStr := r.FormValue("platform_fee_percentage"); feeStr != "" {
		if fee, err := strconv.ParseFloat(feeStr, 64); err == nil {
			req.PlatformFeePercentage = &fee
		} else {
			errors["platform_fee_percentage"] = "Invalid platform fee percentage"
		}
	}

	// Parse minimum withdrawal amount
	if minStr := r.FormValue("min_withdrawal_amount"); minStr != "" {
		if min, err := strconv.ParseFloat(minStr, 64); err == nil {
			req.MinWithdrawalAmount = &min
		} else {
			errors["min_withdrawal_amount"] = "Invalid minimum withdrawal amount"
		}
	}

	// Parse maximum withdrawal amount
	if maxStr := r.FormValue("max_withdrawal_amount"); maxStr != "" {
		if max, err := strconv.ParseFloat(maxStr, 64); err == nil {
			req.MaxWithdrawalAmount = &max
		} else {
			errors["max_withdrawal_amount"] = "Invalid maximum withdrawal amount"
		}
	}

	// Parse withdrawal processing days
	if daysStr := r.FormValue("withdrawal_processing_days"); daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil {
			req.WithdrawalProcessingDays = &days
		} else {
			errors["withdrawal_processing_days"] = "Invalid processing days"
		}
	}

	// Parse boolean settings
	eventModeration := r.FormValue("event_moderation_enabled") == "on"
	req.EventModerationEnabled = &eventModeration

	autoApprove := r.FormValue("auto_approve_organizers") == "on"
	req.AutoApproveOrganizers = &autoApprove

	maintenanceMode := r.FormValue("maintenance_mode") == "on"
	req.MaintenanceMode = &maintenanceMode

	// If there are validation errors, re-render the form
	if len(errors) > 0 {
		settings, _ := h.settingsService.GetSettings()
		formData := map[string]interface{}{
			"platform_fee_percentage":    r.FormValue("platform_fee_percentage"),
			"min_withdrawal_amount":      r.FormValue("min_withdrawal_amount"),
			"max_withdrawal_amount":      r.FormValue("max_withdrawal_amount"),
			"withdrawal_processing_days": r.FormValue("withdrawal_processing_days"),
			"event_moderation_enabled":   eventModeration,
			"auto_approve_organizers":    autoApprove,
			"maintenance_mode":           maintenanceMode,
		}

		component := pages.AdminSettingsPage(user, settings, formData, errors)
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// Update settings
	_, err := h.settingsService.UpdateSettings(req)
	if err != nil {
		errors["general"] = err.Error()
		settings, _ := h.settingsService.GetSettings()
		formData := map[string]interface{}{
			"platform_fee_percentage":    r.FormValue("platform_fee_percentage"),
			"min_withdrawal_amount":      r.FormValue("min_withdrawal_amount"),
			"max_withdrawal_amount":      r.FormValue("max_withdrawal_amount"),
			"withdrawal_processing_days": r.FormValue("withdrawal_processing_days"),
			"event_moderation_enabled":   eventModeration,
			"auto_approve_organizers":    autoApprove,
			"maintenance_mode":           maintenanceMode,
		}

		component := pages.AdminSettingsPage(user, settings, formData, errors)
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// Redirect to settings page with success message
	http.Redirect(w, r, "/admin/settings?success=1", http.StatusSeeOther)
}