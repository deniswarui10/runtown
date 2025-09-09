package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"
	"event-ticketing-platform/web/templates/pages"
)

type AdminHandler struct {
	userService  services.UserServiceInterface
	eventService services.EventServiceInterface
	orderService services.OrderServiceInterface
}

func NewAdminHandler(userService services.UserServiceInterface, eventService services.EventServiceInterface, orderService services.OrderServiceInterface) *AdminHandler {
	return &AdminHandler{
		userService:  userService,
		eventService: eventService,
		orderService: orderService,
	}
}

// AdminDashboard displays the admin dashboard with system overview
func (h *AdminHandler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		// Redirect to login page for unauthenticated users
		http.Redirect(w, r, "/auth/login?redirect=/admin", http.StatusSeeOther)
		return
	}
	
	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Forbidden - Admin access required", http.StatusForbidden)
		return
	}

	// Get system statistics
	stats, err := h.getSystemStats()
	if err != nil {
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	// Render admin dashboard
	component := pages.AdminDashboard(user, stats)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// UserManagement displays the user management interface
func (h *AdminHandler) UserManagement(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		// Redirect to login page for unauthenticated users
		http.Redirect(w, r, "/auth/login?redirect=/admin/users", http.StatusSeeOther)
		return
	}
	
	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Forbidden - Admin access required", http.StatusForbidden)
		return
	}

	// Get query parameters for filtering and pagination
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	search := r.URL.Query().Get("search")
	roleFilter := r.URL.Query().Get("role")

	// Get users with pagination
	users, totalCount, err := h.userService.GetUsersWithPagination(page, 20, search, roleFilter)
	if err != nil {
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	// Calculate pagination info
	totalPages := (totalCount + 19) / 20 // Ceiling division
	
	paginationInfo := map[string]interface{}{
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"TotalCount":  totalCount,
		"HasPrev":     page > 1,
		"HasNext":     page < totalPages,
		"PrevPage":    page - 1,
		"NextPage":    page + 1,
	}

	// Render user management page
	component := pages.AdminUserManagement(user, users, paginationInfo, search, roleFilter)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// UpdateUserRole handles user role updates
func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	newRole := r.FormValue("role")
	if newRole == "" {
		http.Error(w, "Role is required", http.StatusBadRequest)
		return
	}

	// Validate role
	var role models.UserRole
	switch newRole {
	case "user":
		role = models.UserRoleUser
	case "organizer":
		role = models.UserRoleOrganizer
	case "admin":
		role = models.UserRoleAdmin
	default:
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	// Update user role
	err = h.userService.UpdateUserRole(userID, role)
	if err != nil {
		http.Error(w, "Failed to update user role", http.StatusInternalServerError)
		return
	}

	// Redirect back to user management
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// SuspendUser handles user account suspension
func (h *AdminHandler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Suspend user
	err = h.userService.SuspendUser(userID)
	if err != nil {
		http.Error(w, "Failed to suspend user", http.StatusInternalServerError)
		return
	}

	// Redirect back to user management
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// ActivateUser handles user account activation
func (h *AdminHandler) ActivateUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Activate user
	err = h.userService.ActivateUser(userID)
	if err != nil {
		http.Error(w, "Failed to activate user", http.StatusInternalServerError)
		return
	}

	// Redirect back to user management
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// getSystemStats retrieves system statistics for the dashboard
func (h *AdminHandler) getSystemStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get user counts
	totalUsers, err := h.userService.GetUserCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get user count: %w", err)
	}
	stats["TotalUsers"] = totalUsers

	activeUsers, err := h.userService.GetActiveUserCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get active user count: %w", err)
	}
	stats["ActiveUsers"] = activeUsers

	// Get event counts
	totalEvents, err := h.eventService.GetEventCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get event count: %w", err)
	}
	stats["TotalEvents"] = totalEvents

	publishedEvents, err := h.eventService.GetPublishedEventCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get published event count: %w", err)
	}
	stats["PublishedEvents"] = publishedEvents

	// Get order statistics
	totalOrders, err := h.orderService.GetOrderCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get order count: %w", err)
	}
	stats["TotalOrders"] = totalOrders

	totalRevenue, err := h.orderService.GetTotalRevenue()
	if err != nil {
		return nil, fmt.Errorf("failed to get total revenue: %w", err)
	}
	stats["TotalRevenue"] = totalRevenue

	return stats, nil
}

// CategoryManagement displays the category management interface
func (h *AdminHandler) CategoryManagement(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login?redirect=/admin/categories", http.StatusSeeOther)
		return
	}
	
	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Forbidden - Admin access required", http.StatusForbidden)
		return
	}

	// Get all categories
	categories, err := h.eventService.GetCategories()
	if err != nil {
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	// Render category management page
	component := pages.AdminCategoryManagement(user, categories)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// CreateCategoryPage displays the category creation form
func (h *AdminHandler) CreateCategoryPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	
	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Render create category page
	component := pages.AdminCreateCategory(user, nil, nil)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// CreateCategory handles category creation
func (h *AdminHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	slug := r.FormValue("slug")

	// Validate
	errors := make(map[string]string)
	if name == "" {
		errors["name"] = "Name is required"
	}
	if slug == "" {
		errors["slug"] = "Slug is required"
	}

	if len(errors) > 0 {
		formData := map[string]interface{}{
			"name": name,
			"description": description,
			"slug": slug,
		}
		component := pages.AdminCreateCategory(user, formData, errors)
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
		return
	}

	// Create category (we'll need to add this to the service)
	// For now, redirect back to category management
	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}

// EditCategoryPage displays the category edit form
func (h *AdminHandler) EditCategoryPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}
	
	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get category ID from URL
	_, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Get category (we'll need to add this to the service)
	// For now, redirect back to category management
	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}

// UpdateCategory handles category updates
func (h *AdminHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get category ID from URL
	_, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Parse form and update category (we'll need to add this to the service)
	// For now, redirect back to category management
	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}

// DeleteCategory handles category deletion
func (h *AdminHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	if user.Role != models.UserRoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get category ID from URL
	_, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Delete category (we'll need to add this to the service)
	// For now, redirect back to category management
	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}