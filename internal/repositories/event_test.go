package repositories

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"

	_ "github.com/lib/pq"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// For integration tests, use the development database
	// In a real project, you'd want a separate test database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Try to read from .env.local for local development
		dbURL = "postgresql://neondb_owner:npg_96KzrRoFeOWq@ep-silent-moon-a2h6b371-pooler.eu-central-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		t.Skipf("Failed to ping test database: %v", err)
	}

	return db
}

// setupEventTestDB creates a test database connection and sets up tables
func setupEventTestDB(t *testing.T) *sql.DB {
	db := setupTestDB(t)

	// Create categories table first (foreign key dependency)
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			slug VARCHAR(100) UNIQUE NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		t.Fatalf("Failed to create categories table: %v", err)
	}

	// Create users table (foreign key dependency)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			role VARCHAR(20) DEFAULT 'attendee',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Create events table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			start_date TIMESTAMP NOT NULL,
			end_date TIMESTAMP NOT NULL,
			location VARCHAR(255) NOT NULL,
			category_id INTEGER REFERENCES categories(id),
			organizer_id INTEGER REFERENCES users(id),
			image_url VARCHAR(500),
			status VARCHAR(20) DEFAULT 'draft',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		t.Fatalf("Failed to create events table: %v", err)
	}

	// Create ticket_types table for price filtering tests
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ticket_types (
			id SERIAL PRIMARY KEY,
			event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
			name VARCHAR(100) NOT NULL,
			description TEXT,
			price INTEGER NOT NULL,
			quantity INTEGER NOT NULL,
			sold INTEGER DEFAULT 0,
			sale_start TIMESTAMP NOT NULL,
			sale_end TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		t.Fatalf("Failed to create ticket_types table: %v", err)
	}

	return db
}

// cleanupTestData removes test data from the database
func cleanupTestData(t *testing.T, db *sql.DB, organizerID int) {
	// Delete events created by the test organizer
	_, err := db.Exec("DELETE FROM events WHERE organizer_id = $1", organizerID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup test events: %v", err)
	}
}

// createTestCategory creates a test category with unique slug
func createTestCategory(t *testing.T, db *sql.DB) int {
	var categoryID int
	// Generate unique slug using timestamp
	slug := fmt.Sprintf("test-category-%d", time.Now().UnixNano())
	name := fmt.Sprintf("Test Category %d", time.Now().UnixNano())
	err := db.QueryRow(`
		INSERT INTO categories (name, slug, description, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		name, slug, "Test category description", time.Now(),
	).Scan(&categoryID)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}
	return categoryID
}

// createTestUser creates a test user with unique email
func createTestUser(t *testing.T, db *sql.DB, role models.UserRole) int {
	var userID int
	// Generate unique email using timestamp
	email := fmt.Sprintf("test%d@example.com", time.Now().UnixNano())
	err := db.QueryRow(`
		INSERT INTO users (email, password_hash, first_name, last_name, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		email, "hashedpassword", "Test", "User", role, time.Now(), time.Now(),
	).Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return userID
}

func TestEventRepository_Create(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)
	categoryID := createTestCategory(t, db)
	organizerID := createTestUser(t, db, models.RoleOrganizer)

	startDate := time.Now().Add(24 * time.Hour)
	endDate := startDate.Add(2 * time.Hour)

	req := &models.EventCreateRequest{
		Title:       "Test Event",
		Description: "Test event description",
		StartDate:   startDate,
		EndDate:     endDate,
		Location:    "Test Location",
		CategoryID:  categoryID,
		ImageURL:    "https://example.com/image.jpg",
		ImageFormat: "jpg",
		ImageSize:   1024,
		ImageKey:    "test-image-key",
		ImageWidth:  800,
		ImageHeight: 600,
		Status:      models.StatusDraft,
	}

	event, err := repo.Create(req, organizerID)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	if event.ID == 0 {
		t.Error("Expected event ID to be set")
	}

	if event.Title != req.Title {
		t.Errorf("Expected title %s, got %s", req.Title, event.Title)
	}

	if event.OrganizerID != organizerID {
		t.Errorf("Expected organizer ID %d, got %d", organizerID, event.OrganizerID)
	}

	if event.Status != req.Status {
		t.Errorf("Expected status %s, got %s", req.Status, event.Status)
	}
}

func TestEventRepository_Create_ValidationError(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)
	organizerID := createTestUser(t, db, models.RoleOrganizer)

	req := &models.EventCreateRequest{
		Title: "", // Invalid: empty title
	}

	_, err := repo.Create(req, organizerID)
	if err == nil {
		t.Error("Expected validation error for empty title")
	}
}

func TestEventRepository_GetByID(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)
	categoryID := createTestCategory(t, db)
	organizerID := createTestUser(t, db, models.RoleOrganizer)

	// Create test event
	startDate := time.Now().Add(24 * time.Hour)
	endDate := startDate.Add(2 * time.Hour)

	req := &models.EventCreateRequest{
		Title:       "Test Event",
		Description: "Test event description",
		StartDate:   startDate,
		EndDate:     endDate,
		Location:    "Test Location",
		CategoryID:  categoryID,
		Status:      models.StatusPublished,
	}

	createdEvent, err := repo.Create(req, organizerID)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	// Test GetByID
	event, err := repo.GetByID(createdEvent.ID)
	if err != nil {
		t.Fatalf("Failed to get event by ID: %v", err)
	}

	if event.ID != createdEvent.ID {
		t.Errorf("Expected ID %d, got %d", createdEvent.ID, event.ID)
	}

	if event.Title != req.Title {
		t.Errorf("Expected title %s, got %s", req.Title, event.Title)
	}
}

func TestEventRepository_GetByID_NotFound(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)

	_, err := repo.GetByID(999)
	if err == nil {
		t.Error("Expected error for non-existent event")
	}
}

func TestEventRepository_Update(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)
	categoryID := createTestCategory(t, db)
	organizerID := createTestUser(t, db, models.RoleOrganizer)

	// Create test event
	startDate := time.Now().Add(24 * time.Hour)
	endDate := startDate.Add(2 * time.Hour)

	createReq := &models.EventCreateRequest{
		Title:       "Original Title",
		Description: "Original description",
		StartDate:   startDate,
		EndDate:     endDate,
		Location:    "Original Location",
		CategoryID:  categoryID,
		Status:      models.StatusDraft,
	}

	createdEvent, err := repo.Create(createReq, organizerID)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	// Update event
	updateReq := &models.EventUpdateRequest{
		Title:       "Updated Title",
		Description: "Updated description",
		StartDate:   startDate.Add(1 * time.Hour),
		EndDate:     endDate.Add(1 * time.Hour),
		Location:    "Updated Location",
		CategoryID:  categoryID,
		Status:      models.StatusPublished,
	}

	updatedEvent, err := repo.Update(createdEvent.ID, updateReq, organizerID)
	if err != nil {
		t.Fatalf("Failed to update event: %v", err)
	}

	if updatedEvent.Title != updateReq.Title {
		t.Errorf("Expected title %s, got %s", updateReq.Title, updatedEvent.Title)
	}

	if updatedEvent.Status != updateReq.Status {
		t.Errorf("Expected status %s, got %s", updateReq.Status, updatedEvent.Status)
	}
}

func TestEventRepository_Update_WrongOrganizer(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)
	categoryID := createTestCategory(t, db)
	organizerID := createTestUser(t, db, models.RoleOrganizer)
	otherOrganizerID := createTestUser(t, db, models.RoleOrganizer)

	// Create test event
	startDate := time.Now().Add(24 * time.Hour)
	endDate := startDate.Add(2 * time.Hour)

	createReq := &models.EventCreateRequest{
		Title:      "Test Event",
		StartDate:  startDate,
		EndDate:    endDate,
		Location:   "Test Location",
		CategoryID: categoryID,
		Status:     models.StatusDraft,
	}

	createdEvent, err := repo.Create(createReq, organizerID)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	// Try to update with different organizer
	updateReq := &models.EventUpdateRequest{
		Title:      "Updated Title",
		StartDate:  startDate,
		EndDate:    endDate,
		Location:   "Test Location",
		CategoryID: categoryID,
		Status:     models.StatusPublished,
	}

	_, err = repo.Update(createdEvent.ID, updateReq, otherOrganizerID)
	if err == nil {
		t.Error("Expected error when updating event with wrong organizer")
	}
}

func TestEventRepository_Delete(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)
	categoryID := createTestCategory(t, db)
	organizerID := createTestUser(t, db, models.RoleOrganizer)

	// Create test event
	startDate := time.Now().Add(24 * time.Hour)
	endDate := startDate.Add(2 * time.Hour)

	req := &models.EventCreateRequest{
		Title:      "Test Event",
		StartDate:  startDate,
		EndDate:    endDate,
		Location:   "Test Location",
		CategoryID: categoryID,
		Status:     models.StatusDraft,
	}

	createdEvent, err := repo.Create(req, organizerID)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	// Delete event
	err = repo.Delete(createdEvent.ID, organizerID)
	if err != nil {
		t.Fatalf("Failed to delete event: %v", err)
	}

	// Verify event is deleted
	_, err = repo.GetByID(createdEvent.ID)
	if err == nil {
		t.Error("Expected error when getting deleted event")
	}
}

func TestEventRepository_GetByOrganizer(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)
	categoryID := createTestCategory(t, db)
	organizerID := createTestUser(t, db, models.RoleOrganizer)

	// Create multiple test events
	startDate := time.Now().Add(24 * time.Hour)
	endDate := startDate.Add(2 * time.Hour)

	for i := 0; i < 3; i++ {
		req := &models.EventCreateRequest{
			Title:      fmt.Sprintf("Test Event %d", i+1),
			StartDate:  startDate.Add(time.Duration(i) * time.Hour),
			EndDate:    endDate.Add(time.Duration(i) * time.Hour),
			Location:   "Test Location",
			CategoryID: categoryID,
			Status:     models.StatusPublished,
		}

		_, err := repo.Create(req, organizerID)
		if err != nil {
			t.Fatalf("Failed to create event %d: %v", i+1, err)
		}
	}

	// Get events by organizer
	events, err := repo.GetByOrganizer(organizerID)
	if err != nil {
		t.Fatalf("Failed to get events by organizer: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	// Verify all events belong to the organizer
	for _, event := range events {
		if event.OrganizerID != organizerID {
			t.Errorf("Expected organizer ID %d, got %d", organizerID, event.OrganizerID)
		}
	}
}

func TestEventRepository_Search_TextQuery(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)
	categoryID := createTestCategory(t, db)
	organizerID := createTestUser(t, db, models.RoleOrganizer)

	// Create test events with unique prefix
	startDate := time.Now().Add(24 * time.Hour)
	endDate := startDate.Add(2 * time.Hour)
	testPrefix := fmt.Sprintf("TextQuery%d", time.Now().UnixNano())

	events := []struct {
		title       string
		description string
	}{
		{fmt.Sprintf("%s Music Concert", testPrefix), "Live music performance"},
		{fmt.Sprintf("%s Art Exhibition", testPrefix), "Contemporary art showcase"},
		{fmt.Sprintf("%s Music Festival", testPrefix), "Multi-day music event"},
	}

	for _, e := range events {
		req := &models.EventCreateRequest{
			Title:       e.title,
			Description: e.description,
			StartDate:   startDate,
			EndDate:     endDate,
			Location:    "Test Location",
			CategoryID:  categoryID,
			Status:      models.StatusPublished,
		}

		_, err := repo.Create(req, organizerID)
		if err != nil {
			t.Fatalf("Failed to create event: %v", err)
		}
	}

	// Search for "music" within our test events
	filters := EventSearchFilters{
		Query: fmt.Sprintf("%s Music", testPrefix),
		Limit: 10,
	}

	results, total, err := repo.Search(filters)
	if err != nil {
		t.Fatalf("Failed to search events: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 results for 'music', got %d", total)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 events in results, got %d", len(results))
	}

	// Clean up test data
	cleanupTestData(t, db, organizerID)
}

func TestEventRepository_Search_CategoryFilter(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)
	categoryID1 := createTestCategory(t, db)
	
	// Create second category with unique slug
	var categoryID2 int
	slug2 := fmt.Sprintf("music-category-%d", time.Now().UnixNano())
	name2 := fmt.Sprintf("Music Category %d", time.Now().UnixNano())
	err := db.QueryRow(`
		INSERT INTO categories (name, slug, description, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		name2, slug2, "Music events", time.Now(),
	).Scan(&categoryID2)
	if err != nil {
		t.Fatalf("Failed to create second test category: %v", err)
	}

	organizerID := createTestUser(t, db, models.RoleOrganizer)

	// Create events in different categories
	startDate := time.Now().Add(24 * time.Hour)
	endDate := startDate.Add(2 * time.Hour)

	// Category 1 events
	for i := 0; i < 2; i++ {
		req := &models.EventCreateRequest{
			Title:      fmt.Sprintf("Category 1 Event %d", i+1),
			StartDate:  startDate,
			EndDate:    endDate,
			Location:   "Test Location",
			CategoryID: categoryID1,
			Status:     models.StatusPublished,
		}

		_, err := repo.Create(req, organizerID)
		if err != nil {
			t.Fatalf("Failed to create category 1 event: %v", err)
		}
	}

	// Category 2 events
	req := &models.EventCreateRequest{
		Title:      "Category 2 Event",
		StartDate:  startDate,
		EndDate:    endDate,
		Location:   "Test Location",
		CategoryID: categoryID2,
		Status:     models.StatusPublished,
	}

	_, err = repo.Create(req, organizerID)
	if err != nil {
		t.Fatalf("Failed to create category 2 event: %v", err)
	}

	// Search by category 1
	filters := EventSearchFilters{
		CategoryID: categoryID1,
		Limit:      10,
	}

	results, total, err := repo.Search(filters)
	if err != nil {
		t.Fatalf("Failed to search events by category: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 results for category 1, got %d", total)
	}

	// Verify all results are from category 1
	for _, event := range results {
		if event.CategoryID != categoryID1 {
			t.Errorf("Expected category ID %d, got %d", categoryID1, event.CategoryID)
		}
	}
}

func TestEventRepository_Search_DateFilter(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)
	categoryID := createTestCategory(t, db)
	organizerID := createTestUser(t, db, models.RoleOrganizer)

	// Create events at different dates with unique titles
	baseDate := time.Now().Add(24 * time.Hour)
	testPrefix := fmt.Sprintf("DateFilter%d", time.Now().UnixNano())
	
	events := []struct {
		title     string
		startDate time.Time
	}{
		{fmt.Sprintf("%s Event 1", testPrefix), baseDate},
		{fmt.Sprintf("%s Event 2", testPrefix), baseDate.Add(48 * time.Hour)},
		{fmt.Sprintf("%s Event 3", testPrefix), baseDate.Add(96 * time.Hour)},
	}

	for _, e := range events {
		req := &models.EventCreateRequest{
			Title:      e.title,
			StartDate:  e.startDate,
			EndDate:    e.startDate.Add(2 * time.Hour),
			Location:   "Test Location",
			CategoryID: categoryID,
			Status:     models.StatusPublished,
		}

		_, err := repo.Create(req, organizerID)
		if err != nil {
			t.Fatalf("Failed to create event: %v", err)
		}
	}

	// Search for events starting from 2 days from now with query filter
	dateFrom := baseDate.Add(36 * time.Hour)
	filters := EventSearchFilters{
		Query:    testPrefix,
		DateFrom: &dateFrom,
		Limit:    10,
	}

	results, total, err := repo.Search(filters)
	if err != nil {
		t.Fatalf("Failed to search events by date: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 results for date filter, got %d", total)
	}

	// Verify all results are after the filter date
	for _, event := range results {
		if event.StartDate.Before(dateFrom) {
			t.Errorf("Event start date %v is before filter date %v", event.StartDate, dateFrom)
		}
	}

	// Clean up test data
	cleanupTestData(t, db, organizerID)
}

func TestEventRepository_Search_Pagination(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)
	categoryID := createTestCategory(t, db)
	organizerID := createTestUser(t, db, models.RoleOrganizer)

	// Create 5 test events with unique titles for this test
	startDate := time.Now().Add(24 * time.Hour)
	endDate := startDate.Add(2 * time.Hour)
	testPrefix := fmt.Sprintf("PaginationTest%d", time.Now().UnixNano())

	for i := 0; i < 5; i++ {
		req := &models.EventCreateRequest{
			Title:      fmt.Sprintf("%s Event %d", testPrefix, i+1),
			StartDate:  startDate.Add(time.Duration(i) * time.Hour),
			EndDate:    endDate.Add(time.Duration(i) * time.Hour),
			Location:   "Test Location",
			CategoryID: categoryID,
			Status:     models.StatusPublished,
		}

		_, err := repo.Create(req, organizerID)
		if err != nil {
			t.Fatalf("Failed to create event %d: %v", i+1, err)
		}
	}

	// Test pagination - first page with query filter to isolate our test events
	filters := EventSearchFilters{
		Query:  testPrefix,
		Limit:  2,
		Offset: 0,
	}

	results, total, err := repo.Search(filters)
	if err != nil {
		t.Fatalf("Failed to search events with pagination: %v", err)
	}

	if total != 5 {
		t.Errorf("Expected total count of 5, got %d", total)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results on first page, got %d", len(results))
	}

	// Test pagination - second page
	filters.Offset = 2

	results, total, err = repo.Search(filters)
	if err != nil {
		t.Fatalf("Failed to search events with pagination (page 2): %v", err)
	}

	if total != 5 {
		t.Errorf("Expected total count of 5, got %d", total)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results on second page, got %d", len(results))
	}

	// Clean up test data
	cleanupTestData(t, db, organizerID)
}

func TestEventRepository_GetPublishedEvents(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)
	categoryID := createTestCategory(t, db)
	organizerID := createTestUser(t, db, models.RoleOrganizer)

	// Create events with different statuses and unique titles
	startDate := time.Now().Add(24 * time.Hour)
	endDate := startDate.Add(2 * time.Hour)
	testPrefix := fmt.Sprintf("PublishedTest%d", time.Now().UnixNano())

	statuses := []models.EventStatus{
		models.StatusDraft,
		models.StatusPublished,
		models.StatusPublished,
		models.StatusCancelled,
	}

	for i, status := range statuses {
		req := &models.EventCreateRequest{
			Title:      fmt.Sprintf("%s Event %d", testPrefix, i+1),
			StartDate:  startDate,
			EndDate:    endDate,
			Location:   "Test Location",
			CategoryID: categoryID,
			Status:     status,
		}

		_, err := repo.Create(req, organizerID)
		if err != nil {
			t.Fatalf("Failed to create event %d: %v", i+1, err)
		}
	}

	// Get published events by searching with our test prefix
	filters := EventSearchFilters{
		Query:  testPrefix,
		Status: models.StatusPublished,
		Limit:  10,
	}

	events, total, err := repo.Search(filters)
	if err != nil {
		t.Fatalf("Failed to get published events: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 published events, got %d", total)
	}

	// Verify all returned events are published
	for _, event := range events {
		if event.Status != models.StatusPublished {
			t.Errorf("Expected published status, got %s", event.Status)
		}
	}

	// Clean up test data
	cleanupTestData(t, db, organizerID)
}

func TestEventRepository_GetUpcomingEvents(t *testing.T) {
	db := setupEventTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)
	categoryID := createTestCategory(t, db)
	organizerID := createTestUser(t, db, models.RoleOrganizer)

	// Create events at different future times with unique titles
	now := time.Now()
	futureDate := now.Add(24 * time.Hour)
	testPrefix := fmt.Sprintf("UpcomingTest%d", time.Now().UnixNano())

	events := []struct {
		title     string
		startDate time.Time
		endDate   time.Time
	}{
		{fmt.Sprintf("%s Future Event 1", testPrefix), futureDate, futureDate.Add(2 * time.Hour)},
		{fmt.Sprintf("%s Future Event 2", testPrefix), futureDate.Add(24 * time.Hour), futureDate.Add(26 * time.Hour)},
	}

	for _, e := range events {
		req := &models.EventCreateRequest{
			Title:      e.title,
			StartDate:  e.startDate,
			EndDate:    e.endDate,
			Location:   "Test Location",
			CategoryID: categoryID,
			Status:     models.StatusPublished,
		}

		_, err := repo.Create(req, organizerID)
		if err != nil {
			t.Fatalf("Failed to create event: %v", err)
		}
	}

	// Get upcoming events with our test prefix to isolate results
	filters := EventSearchFilters{
		Query:    testPrefix,
		DateFrom: &now,
		Status:   models.StatusPublished,
		Limit:    10,
	}

	upcomingEvents, _, err := repo.Search(filters)
	if err != nil {
		t.Fatalf("Failed to get upcoming events: %v", err)
	}

	if len(upcomingEvents) != 2 {
		t.Errorf("Expected 2 upcoming events, got %d", len(upcomingEvents))
	}

	// Verify all returned events are in the future
	for _, event := range upcomingEvents {
		if event.StartDate.Before(now) {
			t.Errorf("Event %s starts in the past: %v", event.Title, event.StartDate)
		}
	}

	// Clean up test data
	cleanupTestData(t, db, organizerID)
}