package models

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// EventStatus represents the status of an event
type EventStatus string

const (
	StatusDraft           EventStatus = "draft"
	StatusPendingReview   EventStatus = "pending_review"
	StatusPublished       EventStatus = "published"
	StatusRejected        EventStatus = "rejected"
	StatusCancelled       EventStatus = "cancelled"
)

// EventImageMetadata represents image metadata for an event
type EventImageMetadata struct {
	Key         string     `json:"key"`
	URL         string     `json:"url"`
	Size        int64      `json:"size"`
	Format      string     `json:"format"`
	Width       int        `json:"width"`
	Height      int        `json:"height"`
	UploadedAt  *time.Time `json:"uploaded_at"`
}

// Event represents an event in the system
type Event struct {
	ID          int         `json:"id" db:"id"`
	Title       string      `json:"title" db:"title"`
	Description string      `json:"description" db:"description"`
	StartDate   time.Time   `json:"start_date" db:"start_date"`
	EndDate     time.Time   `json:"end_date" db:"end_date"`
	Location    string      `json:"location" db:"location"`
	CategoryID  int         `json:"category_id" db:"category_id"`
	OrganizerID int         `json:"organizer_id" db:"organizer_id"`
	ImageURL    string      `json:"image_url" db:"image_url"`
	ImageKey    string      `json:"image_key" db:"image_key"`
	ImageSize   int64       `json:"image_size" db:"image_size"`
	ImageFormat string      `json:"image_format" db:"image_format"`
	ImageWidth  int         `json:"image_width" db:"image_width"`
	ImageHeight int         `json:"image_height" db:"image_height"`
	ImageUploadedAt *time.Time `json:"image_uploaded_at" db:"image_uploaded_at"`
	Status      EventStatus `json:"status" db:"status"`
	ReviewedAt  *time.Time  `json:"reviewed_at" db:"reviewed_at"`
	ReviewedBy  *int        `json:"reviewed_by" db:"reviewed_by"`
	RejectionReason string  `json:"rejection_reason" db:"rejection_reason"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
	
	// Related data
	Organizer *User     `json:"organizer,omitempty"`
	Category  *Category `json:"category,omitempty"`
	Reviewer  *User     `json:"reviewer,omitempty"`
}

// EventCreateRequest represents the data needed to create a new event
type EventCreateRequest struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	StartDate   time.Time   `json:"start_date"`
	EndDate     time.Time   `json:"end_date"`
	Location    string      `json:"location"`
	CategoryID  int         `json:"category_id"`
	ImageURL    string      `json:"image_url"`
	ImageKey    string      `json:"image_key"`
	ImageSize   int64       `json:"image_size"`
	ImageFormat string      `json:"image_format"`
	ImageWidth  int         `json:"image_width"`
	ImageHeight int         `json:"image_height"`
	Status      EventStatus `json:"status"`
}

// EventUpdateRequest represents the data that can be updated for an event
type EventUpdateRequest struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	StartDate   time.Time   `json:"start_date"`
	EndDate     time.Time   `json:"end_date"`
	Location    string      `json:"location"`
	CategoryID  int         `json:"category_id"`
	ImageURL    string      `json:"image_url"`
	ImageKey    string      `json:"image_key"`
	ImageSize   int64       `json:"image_size"`
	ImageFormat string      `json:"image_format"`
	ImageWidth  int         `json:"image_width"`
	ImageHeight int         `json:"image_height"`
	Status      EventStatus `json:"status"`
}

// Validate validates the event data
func (e *Event) Validate() error {
	if err := e.validateTitle(); err != nil {
		return err
	}
	
	if err := e.validateDates(); err != nil {
		return err
	}
	
	if err := e.validateLocation(); err != nil {
		return err
	}
	
	if err := e.validateStatus(); err != nil {
		return err
	}
	
	if err := e.validateDescription(); err != nil {
		return err
	}
	
	if err := e.validateImageURL(); err != nil {
		return err
	}
	
	if err := e.validateImageMetadata(); err != nil {
		return err
	}
	
	return nil
}

// ValidateCreate validates event creation data
func (req *EventCreateRequest) Validate() error {
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	
	if err := validateDates(req.StartDate, req.EndDate); err != nil {
		return err
	}
	
	if err := validateLocation(req.Location); err != nil {
		return err
	}
	
	if err := validateStatus(req.Status); err != nil {
		return err
	}
	
	if err := validateDescription(req.Description); err != nil {
		return err
	}
	
	if err := validateImageURL(req.ImageURL); err != nil {
		return err
	}
	
	if err := validateImageMetadata(req.ImageKey, req.ImageFormat, req.ImageSize, req.ImageWidth, req.ImageHeight); err != nil {
		return err
	}
	
	return nil
}

// ValidateUpdate validates event update data
func (req *EventUpdateRequest) Validate() error {
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	
	if err := validateDates(req.StartDate, req.EndDate); err != nil {
		return err
	}
	
	if err := validateLocation(req.Location); err != nil {
		return err
	}
	
	if err := validateStatus(req.Status); err != nil {
		return err
	}
	
	if err := validateDescription(req.Description); err != nil {
		return err
	}
	
	if err := validateImageURL(req.ImageURL); err != nil {
		return err
	}
	
	if err := validateImageMetadata(req.ImageKey, req.ImageFormat, req.ImageSize, req.ImageWidth, req.ImageHeight); err != nil {
		return err
	}
	
	return nil
}

// validateTitle validates the event title
func (e *Event) validateTitle() error {
	return validateTitle(e.Title)
}

// validateDates validates the event dates
func (e *Event) validateDates() error {
	return validateDates(e.StartDate, e.EndDate)
}

// validateLocation validates the event location
func (e *Event) validateLocation() error {
	return validateLocation(e.Location)
}

// validateStatus validates the event status
func (e *Event) validateStatus() error {
	return validateStatus(e.Status)
}

// validateDescription validates the event description
func (e *Event) validateDescription() error {
	return validateDescription(e.Description)
}

// validateImageURL validates the event image URL
func (e *Event) validateImageURL() error {
	return validateImageURL(e.ImageURL)
}

// validateImageMetadata validates the event image metadata
func (e *Event) validateImageMetadata() error {
	return validateImageMetadata(e.ImageKey, e.ImageFormat, e.ImageSize, e.ImageWidth, e.ImageHeight)
}

// validateTitle validates an event title
func validateTitle(title string) error {
	if title == "" {
		return errors.New("title is required")
	}
	
	if len(title) > 255 {
		return errors.New("title must be less than 255 characters")
	}
	
	if strings.TrimSpace(title) == "" {
		return errors.New("title cannot be only whitespace")
	}
	
	return nil
}

// validateDates validates event start and end dates
func validateDates(startDate, endDate time.Time) error {
	if startDate.IsZero() {
		return errors.New("start date is required")
	}
	
	if endDate.IsZero() {
		return errors.New("end date is required")
	}
	
	if startDate.After(endDate) {
		return errors.New("start date must be before end date")
	}
	
	// Don't allow events to be created in the past (with some tolerance for timezone issues)
	now := time.Now().Add(-1 * time.Hour)
	if startDate.Before(now) {
		return errors.New("start date cannot be in the past")
	}
	
	return nil
}

// validateLocation validates an event location
func validateLocation(location string) error {
	if location == "" {
		return errors.New("location is required")
	}
	
	if len(location) > 255 {
		return errors.New("location must be less than 255 characters")
	}
	
	if strings.TrimSpace(location) == "" {
		return errors.New("location cannot be only whitespace")
	}
	
	return nil
}

// validateStatus validates an event status
func validateStatus(status EventStatus) error {
	switch status {
	case StatusDraft, StatusPendingReview, StatusPublished, StatusRejected, StatusCancelled:
		return nil
	default:
		return errors.New("invalid event status")
	}
}

// validateDescription validates an event description
func validateDescription(description string) error {
	// Description is optional, but if provided, it should not be too long
	if len(description) > 10000 {
		return errors.New("description must be less than 10000 characters")
	}
	
	return nil
}

// validateImageURL validates an event image URL
func validateImageURL(imageURL string) error {
	// Image URL is optional, but if provided, it should not be too long
	if len(imageURL) > 500 {
		return errors.New("image URL must be less than 500 characters")
	}
	
	// If URL is provided, validate its format
	if imageURL != "" {
		if err := validateImageURLFormat(imageURL); err != nil {
			return err
		}
	}
	
	return nil
}

// validateImageURLFormat validates the format of an image URL
func validateImageURLFormat(imageURL string) error {
	// Parse URL
	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		return errors.New("invalid image URL format")
	}
	
	// Allow relative paths (for local uploads) or HTTP/HTTPS URLs
	if parsedURL.Scheme != "" && parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("image URL must use HTTP or HTTPS protocol, or be a relative path")
	}
	
	// Check if it looks like an image URL (basic check)
	path := strings.ToLower(parsedURL.Path)
	validExtensions := []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}
	
	hasValidExtension := false
	for _, ext := range validExtensions {
		if strings.HasSuffix(path, ext) {
			hasValidExtension = true
			break
		}
	}
	
	// Allow URLs without extensions if they're from known CDN patterns
	if !hasValidExtension {
		// Check for R2 or other CDN patterns that might not have file extensions
		r2Pattern := regexp.MustCompile(`\.r2\.dev/|\.r2\.cloudflarestorage\.com/`)
		if !r2Pattern.MatchString(imageURL) {
			return errors.New("image URL must point to a valid image file")
		}
	}
	
	return nil
}

// validateImageMetadata validates image metadata fields
func validateImageMetadata(key, format string, size int64, width, height int) error {
	// Validate image key
	if key != "" && len(key) > 255 {
		return errors.New("image key must be less than 255 characters")
	}
	
	// Validate image format
	if format != "" {
		validFormats := []string{"jpeg", "jpg", "png", "webp", "gif"}
		isValid := false
		for _, validFormat := range validFormats {
			if strings.ToLower(format) == validFormat {
				isValid = true
				break
			}
		}
		if !isValid {
			return errors.New("invalid image format")
		}
	}
	
	// Validate image size (max 10MB)
	if size > 10*1024*1024 {
		return errors.New("image size cannot exceed 10MB")
	}
	
	// Validate dimensions
	if width < 0 || height < 0 {
		return errors.New("image dimensions cannot be negative")
	}
	
	if width > 10000 || height > 10000 {
		return errors.New("image dimensions cannot exceed 10000 pixels")
	}
	
	return nil
}

// IsPublished returns true if the event is published
func (e *Event) IsPublished() bool {
	return e.Status == StatusPublished
}

// IsDraft returns true if the event is a draft
func (e *Event) IsDraft() bool {
	return e.Status == StatusDraft
}

// IsCancelled returns true if the event is cancelled
func (e *Event) IsCancelled() bool {
	return e.Status == StatusCancelled
}

// IsPendingReview returns true if the event is pending review
func (e *Event) IsPendingReview() bool {
	return e.Status == StatusPendingReview
}

// IsRejected returns true if the event is rejected
func (e *Event) IsRejected() bool {
	return e.Status == StatusRejected
}

// IsUpcoming returns true if the event is in the future
func (e *Event) IsUpcoming() bool {
	return e.StartDate.After(time.Now())
}

// IsOngoing returns true if the event is currently happening
func (e *Event) IsOngoing() bool {
	now := time.Now()
	return now.After(e.StartDate) && now.Before(e.EndDate)
}

// IsPast returns true if the event has ended
func (e *Event) IsPast() bool {
	return e.EndDate.Before(time.Now())
}

// CanBeEdited returns true if the event can be edited
func (e *Event) CanBeEdited() bool {
	// Events can be edited if they haven't started yet
	return e.StartDate.After(time.Now())
}

// CanBeCancelled returns true if the event can be cancelled
func (e *Event) CanBeCancelled() bool {
	// Events can be cancelled if they haven't ended and aren't already cancelled
	return e.EndDate.After(time.Now()) && e.Status != StatusCancelled
}

// Duration returns the duration of the event
func (e *Event) Duration() time.Duration {
	return e.EndDate.Sub(e.StartDate)
}

// HasImage returns true if the event has an associated image
func (e *Event) HasImage() bool {
	return e.ImageURL != "" && e.ImageKey != ""
}

// GetImageMetadata returns the image metadata for the event
func (e *Event) GetImageMetadata() *EventImageMetadata {
	if !e.HasImage() {
		return nil
	}
	
	return &EventImageMetadata{
		Key:        e.ImageKey,
		URL:        e.ImageURL,
		Size:       e.ImageSize,
		Format:     e.ImageFormat,
		Width:      e.ImageWidth,
		Height:     e.ImageHeight,
		UploadedAt: e.ImageUploadedAt,
	}
}

// SetImageMetadata sets the image metadata for the event
func (e *Event) SetImageMetadata(metadata *EventImageMetadata) {
	if metadata == nil {
		e.ClearImageMetadata()
		return
	}
	
	e.ImageKey = metadata.Key
	e.ImageURL = metadata.URL
	e.ImageSize = metadata.Size
	e.ImageFormat = metadata.Format
	e.ImageWidth = metadata.Width
	e.ImageHeight = metadata.Height
	e.ImageUploadedAt = metadata.UploadedAt
}

// ClearImageMetadata clears all image metadata for the event
func (e *Event) ClearImageMetadata() {
	e.ImageURL = ""
	e.ImageKey = ""
	e.ImageSize = 0
	e.ImageFormat = ""
	e.ImageWidth = 0
	e.ImageHeight = 0
	e.ImageUploadedAt = nil
}

// GetImageAspectRatio returns the aspect ratio of the image (width/height)
func (e *Event) GetImageAspectRatio() float64 {
	if e.ImageHeight == 0 {
		return 0
	}
	return float64(e.ImageWidth) / float64(e.ImageHeight)
}

// IsImageLandscape returns true if the image is in landscape orientation
func (e *Event) IsImageLandscape() bool {
	return e.ImageWidth > e.ImageHeight
}

// IsImagePortrait returns true if the image is in portrait orientation
func (e *Event) IsImagePortrait() bool {
	return e.ImageHeight > e.ImageWidth
}

// IsImageSquare returns true if the image is square
func (e *Event) IsImageSquare() bool {
	return e.ImageWidth == e.ImageHeight && e.ImageWidth > 0
}