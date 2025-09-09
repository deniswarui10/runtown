package handlers

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEventService for testing
type MockEventService struct {
	mock.Mock
}

func (m *MockEventService) GetFeaturedEvents(limit int) ([]*models.Event, error) {
	args := m.Called(limit)
	return args.Get(0).([]*models.Event), args.Error(1)
}

func (m *MockEventService) GetUpcomingEvents(limit int) ([]*models.Event, error) {
	args := m.Called(limit)
	return args.Get(0).([]*models.Event), args.Error(1)
}

func (m *MockEventService) SearchEvents(filters services.EventSearchFilters) ([]*models.Event, int, error) {
	args := m.Called(filters)
	return args.Get(0).([]*models.Event), args.Int(1), args.Error(2)
}

func (m *MockEventService) GetCategories() ([]*models.Category, error) {
	args := m.Called()
	return args.Get(0).([]*models.Category), args.Error(1)
}

func (m *MockEventService) GetEventByID(id int) (*models.Event, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventService) GetEventOrganizer(eventID int) (*models.User, error) {
	args := m.Called(eventID)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockEventService) CreateEvent(req *services.EventCreateRequest) (*models.Event, error) {
	args := m.Called(req)
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventService) UpdateEvent(id int, req *services.EventUpdateRequest) (*models.Event, error) {
	args := m.Called(id, req)
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventService) DeleteEvent(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockEventService) GetEventsByOrganizer(organizerID int) ([]*models.Event, error) {
	args := m.Called(organizerID)
	return args.Get(0).([]*models.Event), args.Error(1)
}

func (m *MockEventService) CanUserEditEvent(eventID int, userID int) (bool, error) {
	args := m.Called(eventID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockEventService) CanUserDeleteEvent(eventID int, userID int) (bool, error) {
	args := m.Called(eventID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockEventService) UpdateEventStatus(eventID int, status models.EventStatus, organizerID int) (*models.Event, error) {
	args := m.Called(eventID, status, organizerID)
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventService) DuplicateEvent(eventID int, organizerID int, newTitle string, newStartDate, newEndDate time.Time) (*models.Event, error) {
	args := m.Called(eventID, organizerID, newTitle, newStartDate, newEndDate)
	return args.Get(0).(*models.Event), args.Error(1)
}

// MockStorageService for testing
type MockStorageService struct {
	mock.Mock
}

func (m *MockStorageService) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error) {
	args := m.Called(ctx, key, reader, contentType, size)
	return args.String(0), args.Error(1)
}

func (m *MockStorageService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorageService) GetURL(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func (m *MockStorageService) GeneratePresignedURL(ctx context.Context, key string, contentType string, expiration time.Duration) (string, error) {
	args := m.Called(ctx, key, contentType, expiration)
	return args.String(0), args.Error(1)
}

func (m *MockStorageService) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

// MockImageService for testing
type MockImageService struct {
	mock.Mock
}

func (m *MockImageService) UploadImage(ctx context.Context, reader io.Reader, filename string) (*services.ImageUploadResult, error) {
	args := m.Called(ctx, reader, filename)
	return args.Get(0).(*services.ImageUploadResult), args.Error(1)
}

func (m *MockImageService) UploadImageWithOptions(ctx context.Context, reader io.Reader, filename string, options services.ImageProcessingOptions) (*services.ImageUploadResult, error) {
	args := m.Called(ctx, reader, filename, options)
	return args.Get(0).(*services.ImageUploadResult), args.Error(1)
}

func (m *MockImageService) DeleteImage(ctx context.Context, keyPrefix string) error {
	args := m.Called(ctx, keyPrefix)
	return args.Error(0)
}

func (m *MockImageService) ValidateImage(reader io.Reader, maxSize int64) error {
	args := m.Called(reader, maxSize)
	return args.Error(0)
}

func (m *MockImageService) GetImageURL(keyPrefix, variant string) string {
	args := m.Called(keyPrefix, variant)
	return args.String(0)
}

func (m *MockImageService) GetOptimalImageURL(keyPrefix, variant string, acceptHeader string) string {
	args := m.Called(keyPrefix, variant, acceptHeader)
	return args.String(0)
}

func (m *MockImageService) GetImageVariants(keyPrefix string) []string {
	args := m.Called(keyPrefix)
	return args.Get(0).([]string)
}

func TestOrganizerEventHandler_EventsListPage(t *testing.T) {
	// Setup
	mockEventService := new(MockEventService)
	mockStorageService := new(MockStorageService)
	mockImageService := new(MockImageService)
	
	handler := NewOrganizerEventHandler(mockEventService, mockStorageService, mockImageService)
	
	// Create test user
	testUser := &models.User{
		ID:    1,
		Email: "organizer@test.com",
		Role:  models.RoleOrganizer,
	}
	
	// Create test events
	testEvents := []*models.Event{
		{
			ID:          1,
			Title:       "Test Event 1",
			Description: "Test Description 1",
			Location:    "Test Location 1",
			Status:      models.StatusDraft,
			StartDate:   time.Now().Add(24 * time.Hour),
			EndDate:     time.Now().Add(26 * time.Hour),
		},
		{
			ID:          2,
			Title:       "Test Event 2",
			Description: "Test Description 2",
			Location:    "Test Location 2",
			Status:      models.StatusPublished,
			StartDate:   time.Now().Add(48 * time.Hour),
			EndDate:     time.Now().Add(50 * time.Hour),
		},
	}
	
	testCategories := []*models.Category{
		{ID: 1, Name: "Music", Slug: "music"},
		{ID: 2, Name: "Sports", Slug: "sports"},
	}
	
	// Setup expectations
	mockEventService.On("GetEventsByOrganizer", 1).Return(testEvents, nil)
	mockEventService.On("GetCategories").Return(testCategories, nil)
	
	// Create request
	req := httptest.NewRequest("GET", "/organizer/events", nil)
	
	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, testUser)
	req = req.WithContext(ctx)
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Call handler
	handler.EventsListPage(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Test Event 1")
	assert.Contains(t, w.Body.String(), "Test Event 2")
	
	// Verify mock expectations
	mockEventService.AssertExpectations(t)
}

func TestOrganizerEventHandler_CreateEventPage(t *testing.T) {
	// Setup
	mockEventService := new(MockEventService)
	mockStorageService := new(MockStorageService)
	mockImageService := new(MockImageService)
	
	handler := NewOrganizerEventHandler(mockEventService, mockStorageService, mockImageService)
	
	// Create test user
	testUser := &models.User{
		ID:    1,
		Email: "organizer@test.com",
		Role:  models.RoleOrganizer,
	}
	
	testCategories := []*models.Category{
		{ID: 1, Name: "Music", Slug: "music"},
		{ID: 2, Name: "Sports", Slug: "sports"},
	}
	
	// Setup expectations
	mockEventService.On("GetCategories").Return(testCategories, nil)
	
	// Create request
	req := httptest.NewRequest("GET", "/organizer/events/create", nil)
	
	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, testUser)
	req = req.WithContext(ctx)
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Call handler
	handler.CreateEventPage(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Create New Event")
	assert.Contains(t, w.Body.String(), "Music")
	assert.Contains(t, w.Body.String(), "Sports")
	
	// Verify mock expectations
	mockEventService.AssertExpectations(t)
}

func TestOrganizerEventHandler_CreateEventSubmit(t *testing.T) {
	// Setup
	mockEventService := new(MockEventService)
	mockStorageService := new(MockStorageService)
	mockImageService := new(MockImageService)
	
	handler := NewOrganizerEventHandler(mockEventService, mockStorageService, mockImageService)
	
	// Create test user
	testUser := &models.User{
		ID:    1,
		Email: "organizer@test.com",
		Role:  models.RoleOrganizer,
	}
	
	// Create test event response
	testEvent := &models.Event{
		ID:          1,
		Title:       "New Test Event",
		Description: "New Test Description",
		Location:    "New Test Location",
		Status:      models.StatusDraft,
		StartDate:   time.Now().Add(24 * time.Hour),
		EndDate:     time.Now().Add(26 * time.Hour),
		CategoryID:  1,
		OrganizerID: 1,
	}
	
	// Setup expectations
	mockEventService.On("CreateEvent", mock.AnythingOfType("*services.EventCreateRequest")).Return(testEvent, nil)
	
	// Create form data
	formData := url.Values{
		"title":       {"New Test Event"},
		"description": {"New Test Description"},
		"location":    {"New Test Location"},
		"start_date":  {time.Now().Add(24 * time.Hour).Format("2006-01-02T15:04")},
		"end_date":    {time.Now().Add(26 * time.Hour).Format("2006-01-02T15:04")},
		"category_id": {"1"},
		"status":      {"draft"},
	}
	
	// Create request
	req := httptest.NewRequest("POST", "/organizer/events", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, testUser)
	req = req.WithContext(ctx)
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Call handler
	handler.CreateEventSubmit(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/organizer/events/1/edit", w.Header().Get("Location"))
	
	// Verify mock expectations
	mockEventService.AssertExpectations(t)
}

func TestOrganizerEventHandler_EditEventPage(t *testing.T) {
	// Setup
	mockEventService := new(MockEventService)
	mockStorageService := new(MockStorageService)
	mockImageService := new(MockImageService)
	
	handler := NewOrganizerEventHandler(mockEventService, mockStorageService, mockImageService)
	
	// Create test user
	testUser := &models.User{
		ID:    1,
		Email: "organizer@test.com",
		Role:  models.RoleOrganizer,
	}
	
	// Create test event
	testEvent := &models.Event{
		ID:          1,
		Title:       "Test Event",
		Description: "Test Description",
		Location:    "Test Location",
		Status:      models.StatusDraft,
		StartDate:   time.Now().Add(24 * time.Hour),
		EndDate:     time.Now().Add(26 * time.Hour),
		CategoryID:  1,
		OrganizerID: 1,
	}
	
	testCategories := []*models.Category{
		{ID: 1, Name: "Music", Slug: "music"},
		{ID: 2, Name: "Sports", Slug: "sports"},
	}
	
	// Setup expectations
	mockEventService.On("CanUserEditEvent", 1, 1).Return(true, nil)
	mockEventService.On("GetEventByID", 1).Return(testEvent, nil)
	mockEventService.On("GetCategories").Return(testCategories, nil)
	
	// Create request with URL parameter
	req := httptest.NewRequest("GET", "/organizer/events/1/edit", nil)
	
	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, testUser)
	req = req.WithContext(ctx)
	
	// Add URL parameter to context (simulate chi router)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Call handler
	handler.EditEventPage(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Edit Event")
	assert.Contains(t, w.Body.String(), "Test Event")
	
	// Verify mock expectations
	mockEventService.AssertExpectations(t)
}

func TestOrganizerEventHandler_DeleteEvent(t *testing.T) {
	// Setup
	mockEventService := new(MockEventService)
	mockStorageService := new(MockStorageService)
	mockImageService := new(MockImageService)
	
	handler := NewOrganizerEventHandler(mockEventService, mockStorageService, mockImageService)
	
	// Create test user
	testUser := &models.User{
		ID:    1,
		Email: "organizer@test.com",
		Role:  models.RoleOrganizer,
	}
	
	// Create test event
	testEvent := &models.Event{
		ID:          1,
		Title:       "Test Event",
		Description: "Test Description",
		Location:    "Test Location",
		Status:      models.StatusDraft,
		StartDate:   time.Now().Add(24 * time.Hour),
		EndDate:     time.Now().Add(26 * time.Hour),
		CategoryID:  1,
		OrganizerID: 1,
	}
	
	// Setup expectations
	mockEventService.On("CanUserDeleteEvent", 1, 1).Return(true, nil)
	mockEventService.On("GetEventByID", 1).Return(testEvent, nil)
	mockEventService.On("DeleteEvent", 1).Return(nil)
	
	// Create request with URL parameter
	req := httptest.NewRequest("DELETE", "/organizer/events/1", nil)
	req.Header.Set("HX-Request", "true") // Simulate HTMX request
	
	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, testUser)
	req = req.WithContext(ctx)
	
	// Add URL parameter to context (simulate chi router)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Call handler
	handler.DeleteEvent(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Verify mock expectations
	mockEventService.AssertExpectations(t)
}

func TestOrganizerEventHandler_CreateEventSubmit_WithImageUpload(t *testing.T) {
	// Setup
	mockEventService := new(MockEventService)
	mockStorageService := new(MockStorageService)
	mockImageService := new(MockImageService)
	
	handler := NewOrganizerEventHandler(mockEventService, mockStorageService, mockImageService)
	
	// Create test user
	testUser := &models.User{
		ID:    1,
		Email: "organizer@test.com",
		Role:  models.RoleOrganizer,
	}
	
	// Create test event response
	testEvent := &models.Event{
		ID:          1,
		Title:       "New Test Event",
		Description: "New Test Description",
		Location:    "New Test Location",
		Status:      models.StatusDraft,
		StartDate:   time.Now().Add(24 * time.Hour),
		EndDate:     time.Now().Add(26 * time.Hour),
		CategoryID:  1,
		OrganizerID: 1,
		ImageURL:    "https://example.com/image.jpg",
	}
	
	// Setup expectations
	mockEventService.On("CreateEvent", mock.AnythingOfType("*services.EventCreateRequest")).Return(testEvent, nil)
	
	// Create multipart form with image
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	
	// Add form fields
	writer.WriteField("title", "New Test Event")
	writer.WriteField("description", "New Test Description")
	writer.WriteField("location", "New Test Location")
	writer.WriteField("start_date", time.Now().Add(24*time.Hour).Format("2006-01-02T15:04"))
	writer.WriteField("end_date", time.Now().Add(26*time.Hour).Format("2006-01-02T15:04"))
	writer.WriteField("category_id", "1")
	writer.WriteField("status", "draft")
	
	// Add image file
	fileWriter, err := writer.CreateFormFile("image", "test.jpg")
	assert.NoError(t, err)
	fileWriter.Write([]byte("fake image data"))
	
	writer.Close()
	
	// Create request
	req := httptest.NewRequest("POST", "/organizer/events", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, testUser)
	req = req.WithContext(ctx)
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Call handler
	handler.CreateEventSubmit(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/organizer/events/1/edit", w.Header().Get("Location"))
	
	// Verify mock expectations
	mockEventService.AssertExpectations(t)
}

func TestOrganizerEventHandler_CreateEventSubmit_ValidationErrors(t *testing.T) {
	// Setup
	mockEventService := new(MockEventService)
	mockStorageService := new(MockStorageService)
	mockImageService := new(MockImageService)
	
	handler := NewOrganizerEventHandler(mockEventService, mockStorageService, mockImageService)
	
	// Create test user
	testUser := &models.User{
		ID:    1,
		Email: "organizer@test.com",
		Role:  models.RoleOrganizer,
	}
	
	testCategories := []*models.Category{
		{ID: 1, Name: "Music", Slug: "music"},
	}
	
	// Setup expectations for re-rendering form
	mockEventService.On("GetCategories").Return(testCategories, nil)
	
	// Create form data with missing required fields
	formData := url.Values{
		"title": {""},  // Missing title
		// Missing other required fields
	}
	
	// Create request
	req := httptest.NewRequest("POST", "/organizer/events", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, testUser)
	req = req.WithContext(ctx)
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Call handler
	handler.CreateEventSubmit(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusOK, w.Code) // Should re-render form with errors
	assert.Contains(t, w.Body.String(), "Title is required")
	
	// Verify mock expectations
	mockEventService.AssertExpectations(t)
}

func TestOrganizerEventHandler_UpdateEventStatus(t *testing.T) {
	// Setup
	mockEventService := new(MockEventService)
	mockStorageService := new(MockStorageService)
	mockImageService := new(MockImageService)
	
	handler := NewOrganizerEventHandler(mockEventService, mockStorageService, mockImageService)
	
	// Create test user
	testUser := &models.User{
		ID:    1,
		Email: "organizer@test.com",
		Role:  models.RoleOrganizer,
	}
	
	// Create updated event
	updatedEvent := &models.Event{
		ID:          1,
		Title:       "Test Event",
		Description: "Test Description",
		Location:    "Test Location",
		Status:      models.StatusPublished,
		StartDate:   time.Now().Add(24 * time.Hour),
		EndDate:     time.Now().Add(26 * time.Hour),
		CategoryID:  1,
		OrganizerID: 1,
	}
	
	// Setup expectations
	mockEventService.On("UpdateEventStatus", 1, models.StatusPublished, 1).Return(updatedEvent, nil)
	
	// Create form data
	formData := url.Values{
		"status": {"published"},
	}
	
	// Create request with URL parameter
	req := httptest.NewRequest("POST", "/organizer/events/1/status", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, testUser)
	req = req.WithContext(ctx)
	
	// Add URL parameter to context (simulate chi router)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Call handler
	handler.UpdateEventStatus(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Published") // Should contain the status badge
	
	// Verify mock expectations
	mockEventService.AssertExpectations(t)
}

func TestOrganizerEventHandler_DuplicateEvent(t *testing.T) {
	// Setup
	mockEventService := new(MockEventService)
	mockStorageService := new(MockStorageService)
	mockImageService := new(MockImageService)
	
	handler := NewOrganizerEventHandler(mockEventService, mockStorageService, mockImageService)
	
	// Create test user
	testUser := &models.User{
		ID:    1,
		Email: "organizer@test.com",
		Role:  models.RoleOrganizer,
	}
	
	// Create duplicated event
	duplicatedEvent := &models.Event{
		ID:          2,
		Title:       "Test Event (Copy)",
		Description: "Test Description",
		Location:    "Test Location",
		Status:      models.StatusDraft,
		StartDate:   time.Now().Add(48 * time.Hour),
		EndDate:     time.Now().Add(50 * time.Hour),
		CategoryID:  1,
		OrganizerID: 1,
	}
	
	startDate := time.Now().Add(48 * time.Hour)
	endDate := time.Now().Add(50 * time.Hour)
	
	// Setup expectations
	mockEventService.On("DuplicateEvent", 1, 1, "Test Event (Copy)", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(duplicatedEvent, nil)
	
	// Create form data
	formData := url.Values{
		"title":      {"Test Event (Copy)"},
		"start_date": {startDate.Format("2006-01-02T15:04")},
		"end_date":   {endDate.Format("2006-01-02T15:04")},
	}
	
	// Create request with URL parameter
	req := httptest.NewRequest("POST", "/organizer/events/1/duplicate", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, testUser)
	req = req.WithContext(ctx)
	
	// Add URL parameter to context (simulate chi router)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Call handler
	handler.DuplicateEvent(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/organizer/events/2/edit", w.Header().Get("Location"))
	
	// Verify mock expectations
	mockEventService.AssertExpectations(t)
}