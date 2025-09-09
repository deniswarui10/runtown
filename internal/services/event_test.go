package services

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/textproto"
	"os"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
)

// Mock EventRepository for testing
type mockEventRepository struct {
	events          map[int]*models.Event
	nextID          int
	createError     error
	getError        error
	updateError     error
	deleteError     error
	searchError     error
	searchResults   []*models.Event
	searchTotal     int
}

func newMockEventRepository() *mockEventRepository {
	return &mockEventRepository{
		events: make(map[int]*models.Event),
		nextID: 1,
	}
}

func (m *mockEventRepository) Create(req *models.EventCreateRequest, organizerID int) (*models.Event, error) {
	if m.createError != nil {
		return nil, m.createError
	}

	event := &models.Event{
		ID:          m.nextID,
		Title:       req.Title,
		Description: req.Description,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Location:    req.Location,
		CategoryID:  req.CategoryID,
		OrganizerID: organizerID,
		ImageURL:    req.ImageURL,
		Status:      req.Status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.events[m.nextID] = event
	m.nextID++

	return event, nil
}

func (m *mockEventRepository) GetByID(id int) (*models.Event, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	event, exists := m.events[id]
	if !exists {
		return nil, errors.New("event not found")
	}

	return event, nil
}

func (m *mockEventRepository) Update(id int, req *models.EventUpdateRequest, organizerID int) (*models.Event, error) {
	if m.updateError != nil {
		return nil, m.updateError
	}

	event, exists := m.events[id]
	if !exists {
		return nil, errors.New("event not found")
	}

	if event.OrganizerID != organizerID {
		return nil, errors.New("event does not belong to organizer")
	}

	// Update event
	event.Title = req.Title
	event.Description = req.Description
	event.StartDate = req.StartDate
	event.EndDate = req.EndDate
	event.Location = req.Location
	event.CategoryID = req.CategoryID
	event.ImageURL = req.ImageURL
	event.Status = req.Status
	event.UpdatedAt = time.Now()

	return event, nil
}

func (m *mockEventRepository) Delete(id int, organizerID int) error {
	if m.deleteError != nil {
		return m.deleteError
	}

	event, exists := m.events[id]
	if !exists {
		return errors.New("event not found")
	}

	if event.OrganizerID != organizerID {
		return errors.New("event does not belong to organizer")
	}

	delete(m.events, id)
	return nil
}

func (m *mockEventRepository) GetByOrganizer(organizerID int) ([]*models.Event, error) {
	var events []*models.Event
	for _, event := range m.events {
		if event.OrganizerID == organizerID {
			events = append(events, event)
		}
	}
	return events, nil
}

func (m *mockEventRepository) Search(filters repositories.EventSearchFilters) ([]*models.Event, int, error) {
	if m.searchError != nil {
		return nil, 0, m.searchError
	}
	return m.searchResults, m.searchTotal, nil
}

func (m *mockEventRepository) GetPublishedEvents(limit, offset int) ([]*models.Event, int, error) {
	return m.searchResults, m.searchTotal, m.searchError
}

func (m *mockEventRepository) GetUpcomingEvents(limit int) ([]*models.Event, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}
	return m.searchResults, nil
}

func (m *mockEventRepository) GetEventsByCategory(categoryID int, limit, offset int) ([]*models.Event, int, error) {
	return m.searchResults, m.searchTotal, m.searchError
}

func (m *mockEventRepository) GetFeaturedEvents(limit int) ([]*models.Event, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}
	return m.searchResults, nil
}

func (m *mockEventRepository) GetCategories() ([]*models.Category, error) {
	// Return some test categories
	return []*models.Category{
		{ID: 1, Name: "Music", Slug: "music"},
		{ID: 2, Name: "Sports", Slug: "sports"},
	}, nil
}

func (m *mockEventRepository) GetEventCount() (int, error) {
	return len(m.events), nil
}

func (m *mockEventRepository) GetPublishedEventCount() (int, error) {
	count := 0
	for _, event := range m.events {
		if event.Status == models.StatusPublished {
			count++
		}
	}
	return count, nil
}

// Mock UserRepository for testing
type mockUserRepository struct {
	users   map[int]*models.User
	getError error
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[int]*models.User),
	}
}

func (m *mockUserRepository) GetByID(id int) (*models.User, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	user, exists := m.users[id]
	if !exists {
		return nil, errors.New("user not found")
	}

	return user, nil
}

// Implement other UserRepository methods (not used in these tests)
func (m *mockUserRepository) Create(req *models.UserCreateRequest) (*models.User, error) { return nil, nil }
func (m *mockUserRepository) GetByEmail(email string) (*models.User, error) { return nil, nil }
func (m *mockUserRepository) Update(id int, req *models.UserUpdateRequest) (*models.User, error) { return nil, nil }
func (m *mockUserRepository) UpdatePassword(id int, passwordHash string) error { return nil }
func (m *mockUserRepository) Delete(id int) error { return nil }
func (m *mockUserRepository) Search(filters repositories.UserSearchFilters) ([]*models.User, int, error) { return nil, 0, nil }
func (m *mockUserRepository) GetByRole(role models.UserRole) ([]*models.User, error) { return nil, nil }
func (m *mockUserRepository) CreateSession(userID int, sessionID string, expiresAt time.Time) error { return nil }
func (m *mockUserRepository) GetUserBySession(sessionID string) (*models.User, error) { return nil, nil }
func (m *mockUserRepository) DeleteSession(sessionID string) error { return nil }
func (m *mockUserRepository) DeleteExpiredSessions() error { return nil }
func (m *mockUserRepository) DeleteUserSessions(userID int) error { return nil }
func (m *mockUserRepository) ExtendSession(sessionID string, expiresAt time.Time) error { return nil }
func (m *mockUserRepository) SetVerificationToken(userID int, token string) error { return nil }
func (m *mockUserRepository) GetByVerificationToken(token string) (*models.User, error) { return nil, nil }
func (m *mockUserRepository) VerifyEmail(userID int) error { return nil }
func (m *mockUserRepository) SetPasswordResetToken(userID int, token string, expiresAt time.Time) error { return nil }
func (m *mockUserRepository) GetByPasswordResetToken(token string) (*models.User, error) { return nil, nil }
func (m *mockUserRepository) ClearPasswordResetToken(userID int) error { return nil }
func (m *mockUserRepository) CleanupExpiredTokens() error { return nil }

func setupEventService() (*EventService, *mockEventRepository, *mockUserRepository) {
	eventRepo := newMockEventRepository()
	userRepo := newMockUserRepository()
	authService := NewAuthService(userRepo, nil)
	
	// Create temp directory for uploads
	tempDir, _ := os.MkdirTemp("", "event_service_test")
	
	eventService := NewEventService(eventRepo, authService, tempDir)
	
	return eventService, eventRepo, userRepo
}

func createTestUser(userRepo *mockUserRepository, id int, role models.UserRole) *models.User {
	user := &models.User{
		ID:        id,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      role,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	userRepo.users[id] = user
	return user
}

func createTestEvent(eventRepo *mockEventRepository, id int, organizerID int) *models.Event {
	event := &models.Event{
		ID:          id,
		Title:       "Test Event",
		Description: "Test Description",
		StartDate:   time.Now().Add(24 * time.Hour),
		EndDate:     time.Now().Add(26 * time.Hour),
		Location:    "Test Location",
		CategoryID:  1,
		OrganizerID: organizerID,
		Status:      models.StatusDraft,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	eventRepo.events[id] = event
	return event
}

func TestEventService_CreateEvent(t *testing.T) {
	service, _, userRepo := setupEventService()
	defer os.RemoveAll(service.uploadPath)

	tests := []struct {
		name        string
		setupUser   func() int
		request     *EventCreateRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful event creation by organizer",
			setupUser: func() int {
				user := createTestUser(userRepo, 1, models.RoleOrganizer)
				return user.ID
			},
			request: &EventCreateRequest{
				Title:       "Test Event",
				Description: "Test Description",
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     time.Now().Add(26 * time.Hour),
				Location:    "Test Location",
				CategoryID:  1,
				Status:      models.StatusDraft,
			},
			expectError: false,
		},
		{
			name: "successful event creation by admin",
			setupUser: func() int {
				user := createTestUser(userRepo, 2, models.RoleAdmin)
				return user.ID
			},
			request: &EventCreateRequest{
				Title:       "Admin Event",
				Description: "Admin Description",
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     time.Now().Add(26 * time.Hour),
				Location:    "Admin Location",
				CategoryID:  1,
				Status:      models.StatusPublished,
			},
			expectError: false,
		},
		{
			name: "failed event creation by attendee",
			setupUser: func() int {
				user := createTestUser(userRepo, 3, models.RoleAttendee)
				return user.ID
			},
			request: &EventCreateRequest{
				Title:       "Attendee Event",
				Description: "Should fail",
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     time.Now().Add(26 * time.Hour),
				Location:    "Test Location",
				CategoryID:  1,
				Status:      models.StatusDraft,
			},
			expectError: true,
			errorMsg:    "insufficient permissions",
		},
		{
			name: "failed event creation with non-existent user",
			setupUser: func() int {
				return 999 // Non-existent user
			},
			request: &EventCreateRequest{
				Title:       "Test Event",
				Description: "Test Description",
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     time.Now().Add(26 * time.Hour),
				Location:    "Test Location",
				CategoryID:  1,
				Status:      models.StatusDraft,
			},
			expectError: true,
			errorMsg:    "organizer not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			organizerID := tt.setupUser()

			tt.request.OrganizerID = organizerID
			event, err := service.CreateEvent(tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if event == nil {
				t.Errorf("expected event but got nil")
				return
			}

			if event.Title != tt.request.Title {
				t.Errorf("expected title '%s', got '%s'", tt.request.Title, event.Title)
			}

			if event.OrganizerID != organizerID {
				t.Errorf("expected organizer ID %d, got %d", organizerID, event.OrganizerID)
			}
		})
	}
}

func TestEventService_UpdateEvent(t *testing.T) {
	service, eventRepo, userRepo := setupEventService()
	defer os.RemoveAll(service.uploadPath)

	// Setup test data
	organizer := createTestUser(userRepo, 1, models.RoleOrganizer)
	admin := createTestUser(userRepo, 2, models.RoleAdmin)
	attendee := createTestUser(userRepo, 3, models.RoleAttendee)
	event := createTestEvent(eventRepo, 1, organizer.ID)

	tests := []struct {
		name        string
		eventID     int
		userID      int
		request     *EventUpdateRequest
		expectError bool
		errorMsg    string
	}{
		{
			name:    "successful update by owner",
			eventID: event.ID,
			userID:  organizer.ID,
			request: &EventUpdateRequest{
				Title:       "Updated Event",
				Description: "Updated Description",
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     time.Now().Add(26 * time.Hour),
				Location:    "Updated Location",
				CategoryID:  2,
				Status:      models.StatusPublished,
			},
			expectError: false,
		},
		{
			name:    "successful update by admin",
			eventID: event.ID,
			userID:  admin.ID,
			request: &EventUpdateRequest{
				Title:       "Admin Updated Event",
				Description: "Admin Updated Description",
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     time.Now().Add(26 * time.Hour),
				Location:    "Admin Updated Location",
				CategoryID:  3,
				Status:      models.StatusPublished,
			},
			expectError: false,
		},
		{
			name:    "failed update by attendee",
			eventID: event.ID,
			userID:  attendee.ID,
			request: &EventUpdateRequest{
				Title:       "Should Fail",
				Description: "Should Fail",
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     time.Now().Add(26 * time.Hour),
				Location:    "Should Fail",
				CategoryID:  1,
				Status:      models.StatusDraft,
			},
			expectError: true,
			errorMsg:    "insufficient permissions",
		},
		{
			name:    "failed update of non-existent event",
			eventID: 999,
			userID:  organizer.ID,
			request: &EventUpdateRequest{
				Title:       "Non-existent",
				Description: "Non-existent",
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     time.Now().Add(26 * time.Hour),
				Location:    "Non-existent",
				CategoryID:  1,
				Status:      models.StatusDraft,
			},
			expectError: true,
			errorMsg:    "event not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.request.OrganizerID = tt.userID
			updatedEvent, err := service.UpdateEvent(tt.eventID, tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if updatedEvent == nil {
				t.Errorf("expected updated event but got nil")
				return
			}

			if updatedEvent.Title != tt.request.Title {
				t.Errorf("expected title '%s', got '%s'", tt.request.Title, updatedEvent.Title)
			}
		})
	}
}

func TestEventService_UpdateEventStatus(t *testing.T) {
	service, eventRepo, userRepo := setupEventService()
	defer os.RemoveAll(service.uploadPath)

	// Setup test data
	organizer := createTestUser(userRepo, 1, models.RoleOrganizer)
	event := createTestEvent(eventRepo, 1, organizer.ID)

	tests := []struct {
		name        string
		eventID     int
		userID      int
		newStatus   models.EventStatus
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successful status change from draft to published",
			eventID:     event.ID,
			userID:      organizer.ID,
			newStatus:   models.StatusPublished,
			expectError: false,
		},
		{
			name:        "successful status change from draft to cancelled",
			eventID:     event.ID,
			userID:      organizer.ID,
			newStatus:   models.StatusCancelled,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset event status to draft for each test
			event.Status = models.StatusDraft

			updatedEvent, err := service.UpdateEventStatus(tt.eventID, tt.newStatus, tt.userID)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if updatedEvent == nil {
				t.Errorf("expected updated event but got nil")
				return
			}

			if updatedEvent.Status != tt.newStatus {
				t.Errorf("expected status '%s', got '%s'", tt.newStatus, updatedEvent.Status)
			}
		})
	}
}

func TestEventService_SearchEvents(t *testing.T) {
	service, eventRepo, _ := setupEventService()
	defer os.RemoveAll(service.uploadPath)

	// Setup mock search results
	mockEvents := []*models.Event{
		{ID: 1, Title: "Event 1", Status: models.StatusPublished},
		{ID: 2, Title: "Event 2", Status: models.StatusPublished},
	}
	eventRepo.searchResults = mockEvents
	eventRepo.searchTotal = 2

	tests := []struct {
		name     string
		request  *EventSearchRequest
		expected int
	}{
		{
			name: "basic search",
			request: &EventSearchRequest{
				Query:    "test",
				Page:     1,
				PageSize: 10,
			},
			expected: 2,
		},
		{
			name: "search with filters",
			request: &EventSearchRequest{
				Query:      "test",
				CategoryID: 1,
				Location:   "New York",
				Page:       1,
				PageSize:   5,
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := service.SearchEventsDetailed(tt.request)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if response == nil {
				t.Errorf("expected response but got nil")
				return
			}

			if len(response.Events) != tt.expected {
				t.Errorf("expected %d events, got %d", tt.expected, len(response.Events))
			}

			if response.Total != tt.expected {
				t.Errorf("expected total %d, got %d", tt.expected, response.Total)
			}
		})
	}
}

func TestEventService_DeleteEvent(t *testing.T) {
	service, eventRepo, userRepo := setupEventService()
	defer os.RemoveAll(service.uploadPath)

	// Setup test data
	organizer := createTestUser(userRepo, 1, models.RoleOrganizer)
	admin := createTestUser(userRepo, 2, models.RoleAdmin)
	attendee := createTestUser(userRepo, 3, models.RoleAttendee)
	event := createTestEvent(eventRepo, 1, organizer.ID)

	tests := []struct {
		name        string
		eventID     int
		userID      int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successful deletion by owner",
			eventID:     event.ID,
			userID:      organizer.ID,
			expectError: false,
		},
		{
			name:        "successful deletion by admin",
			eventID:     event.ID,
			userID:      admin.ID,
			expectError: false,
		},
		{
			name:        "failed deletion by attendee",
			eventID:     event.ID,
			userID:      attendee.ID,
			expectError: true,
			errorMsg:    "insufficient permissions",
		},
		{
			name:        "failed deletion of non-existent event",
			eventID:     999,
			userID:      organizer.ID,
			expectError: true,
			errorMsg:    "event not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Recreate event for each test
			if tt.eventID == event.ID {
				eventRepo.events[event.ID] = event
			}

			// Check permissions first (as would be done in handlers)
			canDelete, permErr := service.CanUserDeleteEvent(tt.eventID, tt.userID)
			if permErr != nil && tt.expectError {
				if tt.errorMsg != "" && !contains(permErr.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, permErr.Error())
				}
				return
			}
			
			if !canDelete && tt.expectError {
				// Expected permission error
				return
			}

			err := service.DeleteEvent(tt.eventID)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify event was deleted
			if tt.eventID == event.ID {
				_, exists := eventRepo.events[event.ID]
				if exists {
					t.Errorf("expected event to be deleted but it still exists")
				}
			}
		})
	}
}

func TestEventService_ImageValidation(t *testing.T) {
	service, _, _ := setupEventService()
	defer os.RemoveAll(service.uploadPath)

	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{
			name:        "valid JPEG image",
			contentType: "image/jpeg",
			expected:    true,
		},
		{
			name:        "valid PNG image",
			contentType: "image/png",
			expected:    true,
		},
		{
			name:        "valid GIF image",
			contentType: "image/gif",
			expected:    true,
		},
		{
			name:        "invalid file type",
			contentType: "text/plain",
			expected:    false,
		},
		{
			name:        "invalid file type - PDF",
			contentType: "application/pdf",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isValidImageType(tt.contentType)

			if result != tt.expected {
				t.Errorf("expected %v for content type %s, got %v", tt.expected, tt.contentType, result)
			}
		})
	}
}

// mockFile implements multipart.File interface for testing
type mockFile struct {
	*bytes.Reader
	closed bool
}

func (m *mockFile) Close() error {
	m.closed = true
	return nil
}

// Helper function to create a mock file header
func createMockFileHeader(filename, contentType string, size int64) *multipart.FileHeader {
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", contentType)
	
	return &multipart.FileHeader{
		Filename: filename,
		Header:   header,
		Size:     size,
	}
}

func TestEventService_ValidateEventForPublishing(t *testing.T) {
	service, _, userRepo := setupEventService()
	defer os.RemoveAll(service.uploadPath)

	// Setup test data
	organizer := createTestUser(userRepo, 1, models.RoleOrganizer)

	tests := []struct {
		name        string
		event       *models.Event
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid event for publishing",
			event: &models.Event{
				ID:          1,
				Title:       "Valid Event",
				Description: "Valid Description",
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     time.Now().Add(26 * time.Hour),
				Location:    "Valid Location",
				CategoryID:  1,
				OrganizerID: organizer.ID,
				Status:      models.StatusDraft,
			},
			expectError: false,
		},
		{
			name: "event without title",
			event: &models.Event{
				ID:          2,
				Title:       "",
				Description: "Valid Description",
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     time.Now().Add(26 * time.Hour),
				Location:    "Valid Location",
				CategoryID:  1,
				OrganizerID: organizer.ID,
				Status:      models.StatusDraft,
			},
			expectError: true,
			errorMsg:    "event must have a title",
		},
		{
			name: "event without description",
			event: &models.Event{
				ID:          3,
				Title:       "Valid Title",
				Description: "",
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     time.Now().Add(26 * time.Hour),
				Location:    "Valid Location",
				CategoryID:  1,
				OrganizerID: organizer.ID,
				Status:      models.StatusDraft,
			},
			expectError: true,
			errorMsg:    "event must have a description",
		},
		{
			name: "event in the past",
			event: &models.Event{
				ID:          4,
				Title:       "Past Event",
				Description: "Past Description",
				StartDate:   time.Now().Add(-24 * time.Hour),
				EndDate:     time.Now().Add(-22 * time.Hour),
				Location:    "Past Location",
				CategoryID:  1,
				OrganizerID: organizer.ID,
				Status:      models.StatusDraft,
			},
			expectError: true,
			errorMsg:    "event must be scheduled for the future",
		},
		{
			name: "event with too short duration",
			event: &models.Event{
				ID:          5,
				Title:       "Short Event",
				Description: "Short Description",
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     time.Now().Add(24*time.Hour + 10*time.Minute),
				Location:    "Short Location",
				CategoryID:  1,
				OrganizerID: organizer.ID,
				Status:      models.StatusDraft,
			},
			expectError: true,
			errorMsg:    "event duration must be at least 15 minutes",
		},
		{
			name: "event with too long duration",
			event: &models.Event{
				ID:          6,
				Title:       "Long Event",
				Description: "Long Description",
				StartDate:   time.Now().Add(24 * time.Hour),
				EndDate:     time.Now().Add(24*time.Hour + 31*24*time.Hour),
				Location:    "Long Location",
				CategoryID:  1,
				OrganizerID: organizer.ID,
				Status:      models.StatusDraft,
			},
			expectError: true,
			errorMsg:    "event duration cannot exceed 30 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateEventForPublishing(tt.event)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEventService_CanUserEditEvent(t *testing.T) {
	service, eventRepo, userRepo := setupEventService()
	defer os.RemoveAll(service.uploadPath)

	// Setup test data
	organizer := createTestUser(userRepo, 1, models.RoleOrganizer)
	admin := createTestUser(userRepo, 2, models.RoleAdmin)
	attendee := createTestUser(userRepo, 3, models.RoleAttendee)
	otherOrganizer := createTestUser(userRepo, 4, models.RoleOrganizer)

	// Create events
	upcomingEvent := createTestEvent(eventRepo, 1, organizer.ID)
	upcomingEvent.StartDate = time.Now().Add(24 * time.Hour)
	upcomingEvent.EndDate = time.Now().Add(26 * time.Hour)

	pastEvent := createTestEvent(eventRepo, 2, organizer.ID)
	pastEvent.StartDate = time.Now().Add(-26 * time.Hour)
	pastEvent.EndDate = time.Now().Add(-24 * time.Hour)
	eventRepo.events[2] = pastEvent

	tests := []struct {
		name     string
		eventID  int
		userID   int
		expected bool
	}{
		{
			name:     "organizer can edit their upcoming event",
			eventID:  upcomingEvent.ID,
			userID:   organizer.ID,
			expected: true,
		},
		{
			name:     "admin can edit any event",
			eventID:  upcomingEvent.ID,
			userID:   admin.ID,
			expected: true,
		},
		{
			name:     "attendee cannot edit events",
			eventID:  upcomingEvent.ID,
			userID:   attendee.ID,
			expected: false,
		},
		{
			name:     "organizer cannot edit other's events",
			eventID:  upcomingEvent.ID,
			userID:   otherOrganizer.ID,
			expected: false,
		},
		{
			name:     "organizer cannot edit past events",
			eventID:  pastEvent.ID,
			userID:   organizer.ID,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canEdit, err := service.CanUserEditEvent(tt.eventID, tt.userID)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if canEdit != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, canEdit)
			}
		})
	}
}

func TestEventService_GetEventStatistics(t *testing.T) {
	service, eventRepo, userRepo := setupEventService()
	defer os.RemoveAll(service.uploadPath)

	// Setup test data
	organizer := createTestUser(userRepo, 1, models.RoleOrganizer)
	admin := createTestUser(userRepo, 2, models.RoleAdmin)
	attendee := createTestUser(userRepo, 3, models.RoleAttendee)
	event := createTestEvent(eventRepo, 1, organizer.ID)

	tests := []struct {
		name        string
		eventID     int
		userID      int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "organizer can view their event statistics",
			eventID:     event.ID,
			userID:      organizer.ID,
			expectError: false,
		},
		{
			name:        "admin can view any event statistics",
			eventID:     event.ID,
			userID:      admin.ID,
			expectError: false,
		},
		{
			name:        "attendee cannot view event statistics",
			eventID:     event.ID,
			userID:      attendee.ID,
			expectError: true,
			errorMsg:    "insufficient permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := service.GetEventStatistics(tt.eventID, tt.userID)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if stats == nil {
				t.Errorf("expected statistics but got nil")
				return
			}

			if stats.EventID != tt.eventID {
				t.Errorf("expected event ID %d, got %d", tt.eventID, stats.EventID)
			}

			if stats.Title != event.Title {
				t.Errorf("expected title '%s', got '%s'", event.Title, stats.Title)
			}
		})
	}
}

func TestEventService_DuplicateEvent(t *testing.T) {
	service, eventRepo, userRepo := setupEventService()
	defer os.RemoveAll(service.uploadPath)

	// Setup test data
	organizer := createTestUser(userRepo, 1, models.RoleOrganizer)
	admin := createTestUser(userRepo, 2, models.RoleAdmin)
	attendee := createTestUser(userRepo, 3, models.RoleAttendee)
	originalEvent := createTestEvent(eventRepo, 1, organizer.ID)

	newStartDate := time.Now().Add(48 * time.Hour)
	newEndDate := time.Now().Add(50 * time.Hour)

	tests := []struct {
		name        string
		eventID     int
		userID      int
		newTitle    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "organizer can duplicate their event",
			eventID:     originalEvent.ID,
			userID:      organizer.ID,
			newTitle:    "Duplicated Event",
			expectError: false,
		},
		{
			name:        "admin can duplicate any event",
			eventID:     originalEvent.ID,
			userID:      admin.ID,
			newTitle:    "Admin Duplicated Event",
			expectError: false,
		},
		{
			name:        "attendee cannot duplicate events",
			eventID:     originalEvent.ID,
			userID:      attendee.ID,
			newTitle:    "Should Fail",
			expectError: true,
			errorMsg:    "insufficient permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duplicatedEvent, err := service.DuplicateEvent(tt.eventID, tt.userID, tt.newTitle, newStartDate, newEndDate)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if duplicatedEvent == nil {
				t.Errorf("expected duplicated event but got nil")
				return
			}

			if duplicatedEvent.Title != tt.newTitle {
				t.Errorf("expected title '%s', got '%s'", tt.newTitle, duplicatedEvent.Title)
			}

			if duplicatedEvent.Description != originalEvent.Description {
				t.Errorf("expected description to match original")
			}

			if duplicatedEvent.Status != models.StatusDraft {
				t.Errorf("expected status to be draft, got %s", duplicatedEvent.Status)
			}

			if duplicatedEvent.OrganizerID != tt.userID {
				t.Errorf("expected organizer ID %d, got %d", tt.userID, duplicatedEvent.OrganizerID)
			}
		})
	}
}

func TestEventService_GetEventsByCategory(t *testing.T) {
	service, eventRepo, _ := setupEventService()
	defer os.RemoveAll(service.uploadPath)

	// Setup mock search results
	mockEvents := []*models.Event{
		{ID: 1, Title: "Event 1", CategoryID: 1, Status: models.StatusPublished},
		{ID: 2, Title: "Event 2", CategoryID: 1, Status: models.StatusPublished},
	}
	eventRepo.searchResults = mockEvents
	eventRepo.searchTotal = 2

	tests := []struct {
		name       string
		categoryID int
		page       int
		pageSize   int
		expected   int
	}{
		{
			name:       "get events by category",
			categoryID: 1,
			page:       1,
			pageSize:   10,
			expected:   2,
		},
		{
			name:       "get events with pagination",
			categoryID: 1,
			page:       1,
			pageSize:   1,
			expected:   2, // Total should still be 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := service.GetEventsByCategory(tt.categoryID, tt.page, tt.pageSize)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if response == nil {
				t.Errorf("expected response but got nil")
				return
			}

			if response.Total != tt.expected {
				t.Errorf("expected total %d, got %d", tt.expected, response.Total)
			}

			if len(response.Events) != len(mockEvents) {
				t.Errorf("expected %d events, got %d", len(mockEvents), len(response.Events))
			}
		})
	}
}

func TestEventService_StatusTransitionValidation(t *testing.T) {
	service, eventRepo, userRepo := setupEventService()
	defer os.RemoveAll(service.uploadPath)

	// Setup test data
	organizer := createTestUser(userRepo, 1, models.RoleOrganizer)

	// Create events with different statuses and times
	draftEvent := createTestEvent(eventRepo, 1, organizer.ID)
	draftEvent.Status = models.StatusDraft
	draftEvent.StartDate = time.Now().Add(24 * time.Hour)
	draftEvent.EndDate = time.Now().Add(26 * time.Hour)

	publishedEvent := createTestEvent(eventRepo, 2, organizer.ID)
	publishedEvent.Status = models.StatusPublished
	publishedEvent.StartDate = time.Now().Add(24 * time.Hour)
	publishedEvent.EndDate = time.Now().Add(26 * time.Hour)
	eventRepo.events[2] = publishedEvent

	cancelledEvent := createTestEvent(eventRepo, 3, organizer.ID)
	cancelledEvent.Status = models.StatusCancelled
	cancelledEvent.StartDate = time.Now().Add(24 * time.Hour)
	cancelledEvent.EndDate = time.Now().Add(26 * time.Hour)
	eventRepo.events[3] = cancelledEvent

	ongoingEvent := createTestEvent(eventRepo, 4, organizer.ID)
	ongoingEvent.Status = models.StatusPublished
	ongoingEvent.StartDate = time.Now().Add(-1 * time.Hour)
	ongoingEvent.EndDate = time.Now().Add(1 * time.Hour)
	eventRepo.events[4] = ongoingEvent

	tests := []struct {
		name        string
		eventID     int
		newStatus   models.EventStatus
		expectError bool
		errorMsg    string
	}{
		{
			name:        "draft to published - valid",
			eventID:     draftEvent.ID,
			newStatus:   models.StatusPublished,
			expectError: false,
		},
		{
			name:        "draft to cancelled - valid",
			eventID:     draftEvent.ID,
			newStatus:   models.StatusCancelled,
			expectError: false,
		},
		{
			name:        "published to cancelled - valid",
			eventID:     publishedEvent.ID,
			newStatus:   models.StatusCancelled,
			expectError: false,
		},
		{
			name:        "published to draft - invalid",
			eventID:     publishedEvent.ID,
			newStatus:   models.StatusDraft,
			expectError: true,
			errorMsg:    "invalid status transition: published events can only be cancelled",
		},
		{
			name:        "cancelled to published - invalid",
			eventID:     cancelledEvent.ID,
			newStatus:   models.StatusPublished,
			expectError: true,
			errorMsg:    "invalid status transition: cancelled events cannot change status",
		},
		{
			name:        "ongoing event to published - invalid",
			eventID:     ongoingEvent.ID,
			newStatus:   models.StatusPublished,
			expectError: true,
			errorMsg:    "invalid status transition: ongoing events can only be cancelled",
		},
		{
			name:        "ongoing event to cancelled - valid",
			eventID:     ongoingEvent.ID,
			newStatus:   models.StatusCancelled,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset event status before each test to ensure clean state
			switch tt.eventID {
			case draftEvent.ID:
				draftEvent.Status = models.StatusDraft
				eventRepo.events[draftEvent.ID] = draftEvent
			case publishedEvent.ID:
				publishedEvent.Status = models.StatusPublished
				eventRepo.events[publishedEvent.ID] = publishedEvent
			case cancelledEvent.ID:
				cancelledEvent.Status = models.StatusCancelled
				eventRepo.events[cancelledEvent.ID] = cancelledEvent
			case ongoingEvent.ID:
				ongoingEvent.Status = models.StatusPublished
				eventRepo.events[ongoingEvent.ID] = ongoingEvent
			}

			_, err := service.UpdateEventStatus(tt.eventID, tt.newStatus, organizer.ID)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

