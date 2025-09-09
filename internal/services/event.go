package services

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
)

// EventRepository interface for event data operations
type EventRepository interface {
	Create(req *models.EventCreateRequest, organizerID int) (*models.Event, error)
	GetByID(id int) (*models.Event, error)
	Update(id int, req *models.EventUpdateRequest, organizerID int) (*models.Event, error)
	Delete(id int, organizerID int) error
	GetByOrganizer(organizerID int) ([]*models.Event, error)
	Search(filters repositories.EventSearchFilters) ([]*models.Event, int, error)
	GetPublishedEvents(limit, offset int) ([]*models.Event, int, error)
	GetUpcomingEvents(limit int) ([]*models.Event, error)
	GetEventsByCategory(categoryID int, limit, offset int) ([]*models.Event, int, error)
	GetFeaturedEvents(limit int) ([]*models.Event, error)
	GetCategories() ([]*models.Category, error)
	
	// Admin-specific methods
	GetEventCount() (int, error)
	GetPublishedEventCount() (int, error)
}

// EventService handles event-related business logic
type EventService struct {
	eventRepo   EventRepository
	authService *AuthService
	uploadPath  string
}

// NewEventService creates a new event service
func NewEventService(eventRepo EventRepository, authService *AuthService, uploadPath string) *EventService {
	return &EventService{
		eventRepo:   eventRepo,
		authService: authService,
		uploadPath:  uploadPath,
	}
}

// EventCreateRequest represents a request to create an event
type EventCreateRequest struct {
	Title       string                `json:"title"`
	Description string                `json:"description"`
	StartDate   time.Time             `json:"start_date"`
	EndDate     time.Time             `json:"end_date"`
	Location    string                `json:"location"`
	CategoryID  int                   `json:"category_id"`
	Status      models.EventStatus    `json:"status"`
	OrganizerID int                   `json:"organizer_id"`
	Image       *multipart.FileHeader `json:"-"` // For image upload
}

// EventUpdateRequest represents a request to update an event
type EventUpdateRequest struct {
	Title       string                `json:"title"`
	Description string                `json:"description"`
	StartDate   time.Time             `json:"start_date"`
	EndDate     time.Time             `json:"end_date"`
	Location    string                `json:"location"`
	CategoryID  int                   `json:"category_id"`
	Status      models.EventStatus    `json:"status"`
	OrganizerID int                   `json:"organizer_id"`
	Image       *multipart.FileHeader `json:"-"` // For image upload
}

// EventSearchRequest represents a request to search events
type EventSearchRequest struct {
	Query      string             `json:"query"`
	CategoryID int                `json:"category_id"`
	Location   string             `json:"location"`
	Status     models.EventStatus `json:"status"`
	DateFrom   *time.Time         `json:"date_from"`
	DateTo     *time.Time         `json:"date_to"`
	PriceMin   *int               `json:"price_min"`
	PriceMax   *int               `json:"price_max"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	SortBy     string             `json:"sort_by"`
	SortDesc   bool               `json:"sort_desc"`
}

// EventSearchResponse represents the response from event search
type EventSearchResponse struct {
	Events     []*models.Event `json:"events"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// EventStatistics represents statistics for an event
type EventStatistics struct {
	EventID             int                `json:"event_id"`
	Title               string             `json:"title"`
	Status              models.EventStatus `json:"status"`
	IsUpcoming          bool               `json:"is_upcoming"`
	IsOngoing           bool               `json:"is_ongoing"`
	IsPast              bool               `json:"is_past"`
	Duration            time.Duration      `json:"duration"`
	TotalTicketsSold    int                `json:"total_tickets_sold"`
	TotalRevenue        int                `json:"total_revenue"` // in cents
	TicketTypesCount    int                `json:"ticket_types_count"`
	RegistrationCount   int                `json:"registration_count"`
}

// CreateEvent creates a new event with organizer validation
func (s *EventService) CreateEvent(req *EventCreateRequest) (*models.Event, error) {
	// Validate organizer permissions
	organizer, err := s.authService.userRepo.GetByID(req.OrganizerID)
	if err != nil {
		return nil, fmt.Errorf("organizer not found: %w", err)
	}

	// Check if user has organizer or admin role
	if err := s.authService.RequireRoles(organizer, models.RoleOrganizer, models.RoleAdmin); err != nil {
		return nil, fmt.Errorf("insufficient permissions to create events: %w", err)
	}

	// Handle image upload if provided
	var imageURL, imageKey, imageFormat string
	var imageSize int64
	if req.Image != nil {
		url, key, size, format, err := s.handleImageUpload(req.Image)
		if err != nil {
			return nil, fmt.Errorf("failed to upload image: %w", err)
		}
		imageURL = url
		imageKey = key
		imageSize = size
		imageFormat = format
	}

	// Set default status if not provided
	if req.Status == "" {
		req.Status = models.StatusDraft
	}

	// Create the event request for repository
	createReq := &models.EventCreateRequest{
		Title:       req.Title,
		Description: req.Description,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Location:    req.Location,
		CategoryID:  req.CategoryID,
		ImageURL:    imageURL,
		ImageKey:    imageKey,
		ImageSize:   imageSize,
		ImageFormat: imageFormat,
		ImageWidth:  800,  // Default width - TODO: Read actual dimensions
		ImageHeight: 600,  // Default height - TODO: Read actual dimensions
		Status:      req.Status,
	}

	// Create the event
	event, err := s.eventRepo.Create(createReq, req.OrganizerID)
	if err != nil {
		// Clean up uploaded image if event creation fails
		if imageURL != "" {
			s.cleanupImage(imageURL)
		}
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return event, nil
}

// UpdateEvent updates an existing event with organizer validation
func (s *EventService) UpdateEvent(eventID int, req *EventUpdateRequest) (*models.Event, error) {
	// Validate organizer permissions
	organizer, err := s.authService.userRepo.GetByID(req.OrganizerID)
	if err != nil {
		return nil, fmt.Errorf("organizer not found: %w", err)
	}

	// Check if user has organizer or admin role
	if err := s.authService.RequireRoles(organizer, models.RoleOrganizer, models.RoleAdmin); err != nil {
		return nil, fmt.Errorf("insufficient permissions to update events: %w", err)
	}

	// Get existing event to check ownership and get current image
	existingEvent, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return nil, fmt.Errorf("event not found: %w", err)
	}

	// For non-admin users, ensure they own the event
	if organizer.Role != models.RoleAdmin && existingEvent.OrganizerID != req.OrganizerID {
		return nil, fmt.Errorf("insufficient permissions: event belongs to another organizer")
	}

	// Handle image upload if provided
	var imageURL, imageKey, imageFormat string
	var imageSize int64
	if req.Image != nil {
		url, key, size, format, err := s.handleImageUpload(req.Image)
		if err != nil {
			return nil, fmt.Errorf("failed to upload image: %w", err)
		}
		imageURL = url
		imageKey = key
		imageSize = size
		imageFormat = format
	} else {
		// Keep existing image metadata if no new image provided
		imageURL = existingEvent.ImageURL
		imageKey = existingEvent.ImageKey
		imageSize = existingEvent.ImageSize
		imageFormat = existingEvent.ImageFormat
	}

	// Create the update request for repository
	updateReq := &models.EventUpdateRequest{
		Title:       req.Title,
		Description: req.Description,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Location:    req.Location,
		CategoryID:  req.CategoryID,
		ImageURL:    imageURL,
		ImageKey:    imageKey,
		ImageSize:   imageSize,
		ImageFormat: imageFormat,
		ImageWidth:  func() int { if req.Image != nil { return 800 } else { return existingEvent.ImageWidth } }(),   // Set default for new images
		ImageHeight: func() int { if req.Image != nil { return 600 } else { return existingEvent.ImageHeight } }(), // Set default for new images
		Status:      req.Status,
	}

	// Update the event
	event, err := s.eventRepo.Update(eventID, updateReq, existingEvent.OrganizerID)
	if err != nil {
		// Clean up uploaded image if event update fails
		if req.Image != nil && imageURL != "" {
			s.cleanupImage(imageURL)
		}
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	// Clean up old image if a new one was uploaded successfully
	if req.Image != nil && existingEvent.ImageURL != "" && existingEvent.ImageURL != imageURL {
		s.cleanupImage(existingEvent.ImageURL)
	}

	return event, nil
}

// GetEvent retrieves an event by ID
func (s *EventService) GetEvent(eventID int) (*models.Event, error) {
	event, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return event, nil
}

// GetEventsByOrganizer retrieves events for a specific organizer
// Note: Authorization should be handled at the handler/middleware level
func (s *EventService) GetEventsByOrganizer(organizerID int) ([]*models.Event, error) {
	events, err := s.eventRepo.GetByOrganizer(organizerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by organizer: %w", err)
	}

	return events, nil
}

// SearchEventsDetailed searches for events with filtering and sorting (detailed response)
func (s *EventService) SearchEventsDetailed(req *EventSearchRequest) (*EventSearchResponse, error) {
	// Set default pagination
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100 // Limit max page size
	}

	// Calculate offset
	offset := (req.Page - 1) * req.PageSize

	// Build repository filters
	filters := repositories.EventSearchFilters{
		Query:      req.Query,
		CategoryID: req.CategoryID,
		Location:   req.Location,
		Status:     req.Status,
		DateFrom:   req.DateFrom,
		DateTo:     req.DateTo,
		PriceMin:   req.PriceMin,
		PriceMax:   req.PriceMax,
		Limit:      req.PageSize,
		Offset:     offset,
		SortBy:     req.SortBy,
		SortDesc:   req.SortDesc,
	}

	// Search events
	events, total, err := s.eventRepo.Search(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search events: %w", err)
	}

	// Calculate total pages
	totalPages := (total + req.PageSize - 1) / req.PageSize

	return &EventSearchResponse{
		Events:     events,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetPublishedEvents retrieves published events with pagination
func (s *EventService) GetPublishedEvents(page, pageSize int) (*EventSearchResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	events, total, err := s.eventRepo.GetPublishedEvents(pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get published events: %w", err)
	}

	totalPages := (total + pageSize - 1) / pageSize

	return &EventSearchResponse{
		Events:     events,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetUpcomingEvents retrieves upcoming published events
func (s *EventService) GetUpcomingEvents(limit int) ([]*models.Event, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	events, err := s.eventRepo.GetUpcomingEvents(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming events: %w", err)
	}

	return events, nil
}

// GetFeaturedEvents retrieves featured events
func (s *EventService) GetFeaturedEvents(limit int) ([]*models.Event, error) {
	if limit <= 0 {
		limit = 6
	}
	if limit > 20 {
		limit = 20
	}

	events, err := s.eventRepo.GetFeaturedEvents(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get featured events: %w", err)
	}

	return events, nil
}

// GetEventsByCategory retrieves events by category with pagination
func (s *EventService) GetEventsByCategory(categoryID int, page, pageSize int) (*EventSearchResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	events, total, err := s.eventRepo.GetEventsByCategory(categoryID, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by category: %w", err)
	}

	totalPages := (total + pageSize - 1) / pageSize

	return &EventSearchResponse{
		Events:     events,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// CanUserEditEvent checks if a user can edit a specific event
func (s *EventService) CanUserEditEvent(eventID int, userID int) (bool, error) {
	// Get the user
	user, err := s.authService.userRepo.GetByID(userID)
	if err != nil {
		return false, fmt.Errorf("user not found: %w", err)
	}

	// Admins can edit any event
	if user.Role == models.RoleAdmin {
		return true, nil
	}

	// Check if user has organizer role
	if user.Role != models.RoleOrganizer {
		return false, nil
	}

	// Get the event
	event, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return false, fmt.Errorf("event not found: %w", err)
	}

	// Check if user owns the event and it can be edited
	return event.OrganizerID == userID && event.CanBeEdited(), nil
}

// CanUserDeleteEvent checks if a user can delete a specific event
func (s *EventService) CanUserDeleteEvent(eventID int, userID int) (bool, error) {
	// Get the user
	user, err := s.authService.userRepo.GetByID(userID)
	if err != nil {
		return false, fmt.Errorf("user not found: %w", err)
	}

	// Admins can delete any event
	if user.Role == models.RoleAdmin {
		return true, nil
	}

	// Check if user has organizer role
	if user.Role != models.RoleOrganizer {
		return false, nil
	}

	// Get the event
	event, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return false, fmt.Errorf("event not found: %w", err)
	}

	// Check if user owns the event
	return event.OrganizerID == userID, nil
}

// GetEventStatistics returns statistics for an event (for organizers)
func (s *EventService) GetEventStatistics(eventID int, requestingUserID int) (*EventStatistics, error) {
	// Get the requesting user
	user, err := s.authService.userRepo.GetByID(requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Get the event
	event, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return nil, fmt.Errorf("event not found: %w", err)
	}

	// Check permissions - only event owner or admin can view statistics
	if user.Role != models.RoleAdmin && event.OrganizerID != requestingUserID {
		return nil, fmt.Errorf("insufficient permissions to view event statistics")
	}

	// For now, return basic statistics (this would be enhanced with actual ticket sales data)
	stats := &EventStatistics{
		EventID:     eventID,
		Title:       event.Title,
		Status:      event.Status,
		IsUpcoming:  event.IsUpcoming(),
		IsOngoing:   event.IsOngoing(),
		IsPast:      event.IsPast(),
		Duration:    event.Duration(),
		// These would be populated from ticket sales data in a real implementation
		TotalTicketsSold:    0,
		TotalRevenue:        0,
		TicketTypesCount:    0,
		RegistrationCount:   0,
	}

	return stats, nil
}

// DuplicateEvent creates a copy of an existing event
func (s *EventService) DuplicateEvent(eventID int, organizerID int, newTitle string, newStartDate, newEndDate time.Time) (*models.Event, error) {
	// Validate organizer permissions
	organizer, err := s.authService.userRepo.GetByID(organizerID)
	if err != nil {
		return nil, fmt.Errorf("organizer not found: %w", err)
	}

	// Check if user has organizer or admin role
	if err := s.authService.RequireRoles(organizer, models.RoleOrganizer, models.RoleAdmin); err != nil {
		return nil, fmt.Errorf("insufficient permissions to duplicate events: %w", err)
	}

	// Get the original event
	originalEvent, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return nil, fmt.Errorf("original event not found: %w", err)
	}

	// For non-admin users, ensure they own the original event
	if organizer.Role != models.RoleAdmin && originalEvent.OrganizerID != organizerID {
		return nil, fmt.Errorf("insufficient permissions: event belongs to another organizer")
	}

	// Create the duplicate event request
	duplicateReq := &models.EventCreateRequest{
		Title:       newTitle,
		Description: originalEvent.Description,
		StartDate:   newStartDate,
		EndDate:     newEndDate,
		Location:    originalEvent.Location,
		CategoryID:  originalEvent.CategoryID,
		ImageURL:    originalEvent.ImageURL, // Keep same image
		Status:      models.StatusDraft,     // Always start as draft
	}

	// Create the duplicate event
	duplicateEvent, err := s.eventRepo.Create(duplicateReq, organizerID)
	if err != nil {
		return nil, fmt.Errorf("failed to duplicate event: %w", err)
	}

	return duplicateEvent, nil
}

// UpdateEventStatus updates the status of an event
func (s *EventService) UpdateEventStatus(eventID int, status models.EventStatus, organizerID int) (*models.Event, error) {
	// Validate organizer permissions
	organizer, err := s.authService.userRepo.GetByID(organizerID)
	if err != nil {
		return nil, fmt.Errorf("organizer not found: %w", err)
	}

	// Check if user has organizer or admin role
	if err := s.authService.RequireRoles(organizer, models.RoleOrganizer, models.RoleAdmin); err != nil {
		return nil, fmt.Errorf("insufficient permissions to update event status: %w", err)
	}

	// Get existing event
	existingEvent, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return nil, fmt.Errorf("event not found: %w", err)
	}

	// For non-admin users, ensure they own the event
	if organizer.Role != models.RoleAdmin && existingEvent.OrganizerID != organizerID {
		return nil, fmt.Errorf("insufficient permissions: event belongs to another organizer")
	}

	// Validate status transition
	if err := s.validateStatusTransition(existingEvent, status); err != nil {
		return nil, fmt.Errorf("invalid status transition: %w", err)
	}

	// Create update request with only status change
	updateReq := &models.EventUpdateRequest{
		Title:       existingEvent.Title,
		Description: existingEvent.Description,
		StartDate:   existingEvent.StartDate,
		EndDate:     existingEvent.EndDate,
		Location:    existingEvent.Location,
		CategoryID:  existingEvent.CategoryID,
		ImageURL:    existingEvent.ImageURL,
		Status:      status,
	}

	// Update the event
	event, err := s.eventRepo.Update(eventID, updateReq, existingEvent.OrganizerID)
	if err != nil {
		return nil, fmt.Errorf("failed to update event status: %w", err)
	}

	return event, nil
}

// DeleteEvent deletes an event
// Note: Authorization should be handled at the handler/middleware level
func (s *EventService) DeleteEvent(eventID int) error {
	// Get existing event to get image URL for cleanup
	existingEvent, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return fmt.Errorf("event not found: %w", err)
	}

	// Delete the event
	err = s.eventRepo.Delete(eventID, existingEvent.OrganizerID)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	// Clean up image if it exists
	if existingEvent.ImageURL != "" {
		s.cleanupImage(existingEvent.ImageURL)
	}

	return nil
}

// handleImageUpload handles the upload of event images and returns URL and metadata
func (s *EventService) handleImageUpload(fileHeader *multipart.FileHeader) (string, string, int64, string, error) {
	// Validate file size (max 5MB)
	if fileHeader.Size > 5*1024*1024 {
		return "", "", 0, "", fmt.Errorf("image file too large (max 5MB)")
	}

	// Validate file type by content type and extension
	contentType := fileHeader.Header.Get("Content-Type")
	if !s.isValidImageType(contentType) {
		// Fallback to file extension check
		if !s.isValidImageExtension(fileHeader.Filename) {
			return "", "", 0, "", fmt.Errorf("invalid image type (only JPEG, PNG, GIF allowed). Content-Type: %s, Filename: %s", contentType, fileHeader.Filename)
		}
	}

	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		return "", "", 0, "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	// Generate unique filename
	filename := s.generateImageFilename(fileHeader.Filename)

	// Ensure upload directory exists
	if err := os.MkdirAll(s.uploadPath, 0755); err != nil {
		return "", "", 0, "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Create destination file
	destPath := filepath.Join(s.uploadPath, filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy file content
	_, err = io.Copy(destFile, file)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("failed to copy file content: %w", err)
	}

	// Determine image format from content type or extension
	format := s.getImageFormat(fileHeader.Header.Get("Content-Type"), fileHeader.Filename)
	
	// Return URL, key, size, format
	return "/uploads/events/" + filename, "events/" + filename, fileHeader.Size, format, nil
}

// isValidImageType checks if the content type is a valid image type
func (s *EventService) isValidImageType(contentType string) bool {
	validTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
	}

	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}

	return false
}

// isValidImageExtension checks if the file extension is a valid image type
func (s *EventService) isValidImageExtension(filename string) bool {
	validExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".JPG", ".JPEG", ".PNG", ".GIF"}
	
	for _, ext := range validExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	
	return false
}

// getImageFormat determines the image format from content type or filename
func (s *EventService) getImageFormat(contentType, filename string) string {
	// First try content type
	switch contentType {
	case "image/jpeg", "image/jpg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	}
	
	// Fallback to file extension
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "jpeg"
	case ".png":
		return "png"
	case ".gif":
		return "gif"
	case ".webp":
		return "webp"
	}
	
	// Default fallback
	return "jpeg"
}

// generateImageFilename generates a unique filename for uploaded images
func (s *EventService) generateImageFilename(originalFilename string) string {
	// Get file extension
	ext := filepath.Ext(originalFilename)
	
	// Generate timestamp-based filename
	timestamp := time.Now().Unix()
	
	// Create unique filename
	return fmt.Sprintf("event_%d%s", timestamp, ext)
}

// cleanupImage removes an uploaded image file
func (s *EventService) cleanupImage(imageURL string) {
	if imageURL == "" {
		return
	}

	// Extract filename from URL
	filename := strings.TrimPrefix(imageURL, "/uploads/")
	if filename == imageURL {
		return // Not a local upload
	}

	// Remove file
	filePath := filepath.Join(s.uploadPath, filename)
	os.Remove(filePath) // Ignore errors for cleanup
}

// validateStatusTransition validates if a status transition is allowed
func (s *EventService) validateStatusTransition(event *models.Event, newStatus models.EventStatus) error {
	// Can't change status of past events
	if event.IsPast() {
		return fmt.Errorf("cannot change status of past events")
	}

	// Can't change status if event is currently ongoing
	if event.IsOngoing() && newStatus != models.StatusCancelled {
		return fmt.Errorf("ongoing events can only be cancelled")
	}

	// Validate specific transitions
	switch event.Status {
	case models.StatusDraft:
		// Draft can go to published or cancelled
		if newStatus != models.StatusPublished && newStatus != models.StatusCancelled {
			return fmt.Errorf("draft events can only be published or cancelled")
		}
		// Additional validation for publishing
		if newStatus == models.StatusPublished {
			if err := s.validateEventForPublishing(event); err != nil {
				return fmt.Errorf("event cannot be published: %w", err)
			}
		}
	case models.StatusPublished:
		// Published can only be cancelled
		if newStatus != models.StatusCancelled {
			return fmt.Errorf("published events can only be cancelled")
		}
	case models.StatusCancelled:
		// Cancelled events cannot change status
		return fmt.Errorf("cancelled events cannot change status")
	}

	return nil
}

// validateEventForPublishing validates that an event meets requirements for publishing
func (s *EventService) validateEventForPublishing(event *models.Event) error {
	// Event must have all required fields
	if event.Title == "" {
		return fmt.Errorf("event must have a title")
	}
	
	if event.Description == "" {
		return fmt.Errorf("event must have a description")
	}
	
	if event.Location == "" {
		return fmt.Errorf("event must have a location")
	}
	
	if event.CategoryID <= 0 {
		return fmt.Errorf("event must have a valid category")
	}
	
	// Event must be in the future
	if !event.IsUpcoming() {
		return fmt.Errorf("event must be scheduled for the future")
	}
	
	// Event must have a reasonable duration (at least 15 minutes, max 30 days)
	duration := event.Duration()
	if duration < 15*time.Minute {
		return fmt.Errorf("event duration must be at least 15 minutes")
	}
	
	if duration > 30*24*time.Hour {
		return fmt.Errorf("event duration cannot exceed 30 days")
	}
	
	return nil
}

// GetEventByID retrieves an event by ID
func (s *EventService) GetEventByID(id int) (*models.Event, error) {
	return s.eventRepo.GetByID(id)
}

// GetEventOrganizer retrieves the organizer of an event
func (s *EventService) GetEventOrganizer(eventID int) (*models.User, error) {
	// Get the event first
	event, err := s.eventRepo.GetByID(eventID)
	if err != nil {
		return nil, fmt.Errorf("event not found: %w", err)
	}
	
	// Get the organizer user
	organizer, err := s.authService.userRepo.GetByID(event.OrganizerID)
	if err != nil {
		return nil, fmt.Errorf("organizer not found: %w", err)
	}
	
	return organizer, nil
}

// SearchEvents searches for events with the interface-compatible signature
func (s *EventService) SearchEvents(filters EventSearchFilters) ([]*models.Event, int, error) {
	// Convert interface filters to internal request format
	req := &EventSearchRequest{
		Query:    filters.Query,
		Location: filters.Location,
		Page:     filters.Page,
		PageSize: filters.PerPage,
	}
	
	// Convert category string to CategoryID if provided
	if filters.Category != "" {
		// Look up category ID by name or slug
		categoryID, err := s.getCategoryIDByName(filters.Category)
		if err == nil && categoryID > 0 {
			req.CategoryID = categoryID
		}
	}
	
	// Convert date strings to time.Time pointers if provided
	if filters.DateFrom != "" {
		if dateFrom, err := time.Parse("2006-01-02", filters.DateFrom); err == nil {
			req.DateFrom = &dateFrom
		}
	}
	
	if filters.DateTo != "" {
		if dateTo, err := time.Parse("2006-01-02", filters.DateTo); err == nil {
			req.DateTo = &dateTo
		}
	}
	
	// Call the detailed search method
	response, err := s.SearchEventsDetailed(req)
	if err != nil {
		return nil, 0, err
	}
	
	return response.Events, response.Total, nil
}

// GetCategories retrieves all event categories
func (s *EventService) GetCategories() ([]*models.Category, error) {
	return s.eventRepo.GetCategories()
}

// getCategoryIDByName looks up a category ID by name or slug
func (s *EventService) getCategoryIDByName(categoryName string) (int, error) {
	categories, err := s.GetCategories()
	if err != nil {
		return 0, err
	}
	
	// Try to match by name (case-insensitive) or slug
	for _, category := range categories {
		if strings.EqualFold(category.Name, categoryName) || 
		   strings.EqualFold(category.Slug, categoryName) {
			return category.ID, nil
		}
	}
	
	// Try to parse as ID
	if categoryID, err := strconv.Atoi(categoryName); err == nil {
		// Verify the ID exists
		for _, category := range categories {
			if category.ID == categoryID {
				return categoryID, nil
			}
		}
	}
	
	return 0, fmt.Errorf("category not found: %s", categoryName)
}

// Admin-specific methods

// GetEventCount returns the total number of events
func (s *EventService) GetEventCount() (int, error) {
	return s.eventRepo.GetEventCount()
}

// GetPublishedEventCount returns the number of published events
func (s *EventService) GetPublishedEventCount() (int, error) {
	return s.eventRepo.GetPublishedEventCount()
}