package services

import (
	"fmt"
	"sort"
	"strings"

	"event-ticketing-platform/internal/models"
)

// EventDiscoveryService provides enhanced event discovery capabilities
type EventDiscoveryService struct {
	eventService EventServiceInterface
}

// NewEventDiscoveryService creates a new event discovery service
func NewEventDiscoveryService(eventService EventServiceInterface) *EventDiscoveryService {
	return &EventDiscoveryService{
		eventService: eventService,
	}
}

// DiscoveryFilters represents enhanced filters for event discovery
type DiscoveryFilters struct {
	Query        string    `json:"query"`
	Category     string    `json:"category"`
	Location     string    `json:"location"`
	DateFrom     string    `json:"date_from"`
	DateTo       string    `json:"date_to"`
	PriceMin     int       `json:"price_min"`
	PriceMax     int       `json:"price_max"`
	EventType    string    `json:"event_type"`    // online, offline, hybrid
	SortBy       string    `json:"sort_by"`       // date, price, popularity, relevance
	SortOrder    string    `json:"sort_order"`    // asc, desc
	Page         int       `json:"page"`
	PerPage      int       `json:"per_page"`
	UserID       int       `json:"user_id"`       // For personalized results
	Radius       int       `json:"radius"`        // Search radius in km
	Tags         []string  `json:"tags"`
	Availability string    `json:"availability"`  // available, sold_out, all
}

// DiscoveryResult represents the result of event discovery
type DiscoveryResult struct {
	Events         []*models.Event      `json:"events"`
	TotalCount     int                  `json:"total_count"`
	FilteredCount  int                  `json:"filtered_count"`
	Suggestions    []string             `json:"suggestions"`
	Categories     []*models.Category   `json:"categories"`
	Locations      []string             `json:"locations"`
	PriceRange     PriceRange           `json:"price_range"`
	Facets         map[string][]Facet   `json:"facets"`
	RecommendedFor *models.User         `json:"recommended_for,omitempty"`
}

// PriceRange represents the price range of events
type PriceRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// Facet represents a filter facet with count
type Facet struct {
	Value string `json:"value"`
	Count int    `json:"count"`
	Label string `json:"label"`
}

// DiscoverEvents performs enhanced event discovery with intelligent filtering and sorting
func (s *EventDiscoveryService) DiscoverEvents(filters DiscoveryFilters) (*DiscoveryResult, error) {
	// Convert to basic search filters for now
	basicFilters := EventSearchFilters{
		Query:    filters.Query,
		Category: filters.Category,
		Location: filters.Location,
		DateFrom: filters.DateFrom,
		DateTo:   filters.DateTo,
		Page:     filters.Page,
		PerPage:  filters.PerPage,
	}

	// Get events using existing search
	events, totalCount, err := s.eventService.SearchEvents(basicFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to search events: %w", err)
	}

	// Apply additional filtering
	filteredEvents := s.applyAdvancedFilters(events, filters)

	// Apply sorting
	sortedEvents := s.applySorting(filteredEvents, filters)

	// Apply pagination to sorted results
	paginatedEvents := s.applyPagination(sortedEvents, filters)

	// Get categories for facets
	categories, err := s.eventService.GetCategories()
	if err != nil {
		categories = []*models.Category{} // Continue with empty categories
	}

	// Build result
	result := &DiscoveryResult{
		Events:        paginatedEvents,
		TotalCount:    totalCount,
		FilteredCount: len(filteredEvents),
		Categories:    categories,
		Suggestions:   s.generateSuggestions(filters.Query, events),
		Locations:     s.extractLocations(events),
		PriceRange:    s.calculatePriceRange(events),
		Facets:        s.buildFacets(events),
	}

	return result, nil
}

// applyAdvancedFilters applies additional filters not handled by basic search
func (s *EventDiscoveryService) applyAdvancedFilters(events []*models.Event, filters DiscoveryFilters) []*models.Event {
	var filtered []*models.Event

	for _, event := range events {
		// Price filtering
		if filters.PriceMin > 0 || filters.PriceMax > 0 {
			eventPrice := s.getEventMinPrice(event)
			if filters.PriceMin > 0 && eventPrice < filters.PriceMin {
				continue
			}
			if filters.PriceMax > 0 && eventPrice > filters.PriceMax {
				continue
			}
		}

		// Event type filtering (online/offline/hybrid)
		if filters.EventType != "" && filters.EventType != "all" {
			if !s.matchesEventType(event, filters.EventType) {
				continue
			}
		}

		// Tag filtering
		if len(filters.Tags) > 0 {
			if !s.matchesTags(event, filters.Tags) {
				continue
			}
		}

		// Availability filtering
		if filters.Availability != "" && filters.Availability != "all" {
			if !s.matchesAvailability(event, filters.Availability) {
				continue
			}
		}

		filtered = append(filtered, event)
	}

	return filtered
}

// applySorting sorts events based on the specified criteria
func (s *EventDiscoveryService) applySorting(events []*models.Event, filters DiscoveryFilters) []*models.Event {
	if len(events) == 0 {
		return events
	}

	// Create a copy to avoid modifying the original slice
	sorted := make([]*models.Event, len(events))
	copy(sorted, events)

	switch filters.SortBy {
	case "date":
		sort.Slice(sorted, func(i, j int) bool {
			if filters.SortOrder == "desc" {
				return sorted[i].StartDate.After(sorted[j].StartDate)
			}
			return sorted[i].StartDate.Before(sorted[j].StartDate)
		})
	case "price":
		sort.Slice(sorted, func(i, j int) bool {
			priceI := s.getEventMinPrice(sorted[i])
			priceJ := s.getEventMinPrice(sorted[j])
			if filters.SortOrder == "desc" {
				return priceI > priceJ
			}
			return priceI < priceJ
		})
	case "popularity":
		sort.Slice(sorted, func(i, j int) bool {
			// Sort by ticket sales or views (placeholder logic)
			popularityI := s.getEventPopularity(sorted[i])
			popularityJ := s.getEventPopularity(sorted[j])
			if filters.SortOrder == "desc" {
				return popularityI > popularityJ
			}
			return popularityI < popularityJ
		})
	case "relevance":
		// Sort by relevance to search query
		if filters.Query != "" {
			sort.Slice(sorted, func(i, j int) bool {
				relevanceI := s.calculateRelevance(sorted[i], filters.Query)
				relevanceJ := s.calculateRelevance(sorted[j], filters.Query)
				return relevanceI > relevanceJ // Higher relevance first
			})
		}
	default:
		// Default sort by date (upcoming first)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].StartDate.Before(sorted[j].StartDate)
		})
	}

	return sorted
}

// applyPagination applies pagination to the sorted results
func (s *EventDiscoveryService) applyPagination(events []*models.Event, filters DiscoveryFilters) []*models.Event {
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.PerPage <= 0 {
		filters.PerPage = 12
	}

	start := (filters.Page - 1) * filters.PerPage
	end := start + filters.PerPage

	if start >= len(events) {
		return []*models.Event{}
	}
	if end > len(events) {
		end = len(events)
	}

	return events[start:end]
}

// Helper methods

func (s *EventDiscoveryService) getEventMinPrice(event *models.Event) int {
	// This would typically get the minimum ticket price for the event
	// For now, return a placeholder value
	return 1000 // KES 10.00
}

func (s *EventDiscoveryService) matchesEventType(event *models.Event, eventType string) bool {
	// This would check if the event matches the type (online/offline/hybrid)
	// For now, return true as a placeholder
	return true
}

func (s *EventDiscoveryService) matchesTags(event *models.Event, tags []string) bool {
	// This would check if the event has any of the specified tags
	// For now, return true as a placeholder
	return true
}

func (s *EventDiscoveryService) matchesAvailability(event *models.Event, availability string) bool {
	// This would check ticket availability
	// For now, return true as a placeholder
	return true
}

func (s *EventDiscoveryService) getEventPopularity(event *models.Event) int {
	// This would calculate event popularity based on views, sales, etc.
	// For now, return a placeholder value
	return 0
}

func (s *EventDiscoveryService) calculateRelevance(event *models.Event, query string) float64 {
	if query == "" {
		return 0
	}

	query = strings.ToLower(query)
	relevance := 0.0

	// Title match (highest weight)
	if strings.Contains(strings.ToLower(event.Title), query) {
		relevance += 10.0
	}

	// Description match (medium weight)
	if strings.Contains(strings.ToLower(event.Description), query) {
		relevance += 5.0
	}

	// Location match (medium weight)
	if strings.Contains(strings.ToLower(event.Location), query) {
		relevance += 5.0
	}

	// Category match (low weight)
	if event.Category != nil && strings.Contains(strings.ToLower(event.Category.Name), query) {
		relevance += 2.0
	}

	return relevance
}

func (s *EventDiscoveryService) generateSuggestions(query string, events []*models.Event) []string {
	if query == "" {
		return []string{}
	}

	suggestions := make(map[string]bool)
	query = strings.ToLower(query)

	// Generate suggestions based on event titles and categories
	for _, event := range events {
		title := strings.ToLower(event.Title)
		if strings.Contains(title, query) && title != query {
			suggestions[event.Title] = true
		}
		
		if event.Category != nil {
			category := strings.ToLower(event.Category.Name)
			if strings.Contains(category, query) && category != query {
				suggestions[event.Category.Name] = true
			}
		}
	}

	// Convert to slice and limit
	result := make([]string, 0, 5)
	for suggestion := range suggestions {
		if len(result) >= 5 {
			break
		}
		result = append(result, suggestion)
	}

	return result
}

func (s *EventDiscoveryService) extractLocations(events []*models.Event) []string {
	locationMap := make(map[string]bool)
	for _, event := range events {
		if event.Location != "" {
			locationMap[event.Location] = true
		}
	}

	locations := make([]string, 0, len(locationMap))
	for location := range locationMap {
		locations = append(locations, location)
	}

	sort.Strings(locations)
	return locations
}

func (s *EventDiscoveryService) calculatePriceRange(events []*models.Event) PriceRange {
	if len(events) == 0 {
		return PriceRange{Min: 0, Max: 0}
	}

	min := s.getEventMinPrice(events[0])
	max := min

	for _, event := range events {
		price := s.getEventMinPrice(event)
		if price < min {
			min = price
		}
		if price > max {
			max = price
		}
	}

	return PriceRange{Min: min, Max: max}
}

func (s *EventDiscoveryService) buildFacets(events []*models.Event) map[string][]Facet {
	facets := make(map[string][]Facet)

	// Category facets
	categoryCount := make(map[string]int)
	for _, event := range events {
		if event.Category != nil {
			categoryCount[event.Category.Name]++
		}
	}

	categoryFacets := make([]Facet, 0, len(categoryCount))
	for category, count := range categoryCount {
		categoryFacets = append(categoryFacets, Facet{
			Value: category,
			Count: count,
			Label: category,
		})
	}
	facets["category"] = categoryFacets

	// Location facets
	locationCount := make(map[string]int)
	for _, event := range events {
		if event.Location != "" {
			locationCount[event.Location]++
		}
	}

	locationFacets := make([]Facet, 0, len(locationCount))
	for location, count := range locationCount {
		locationFacets = append(locationFacets, Facet{
			Value: location,
			Count: count,
			Label: location,
		})
	}
	facets["location"] = locationFacets

	return facets
}

// GetPersonalizedRecommendations gets personalized event recommendations for a user
func (s *EventDiscoveryService) GetPersonalizedRecommendations(userID int, limit int) ([]*models.Event, error) {
	// This would implement personalized recommendations based on:
	// - User's past event attendance
	// - User's preferences
	// - Similar users' behavior
	// - Popular events in user's location
	
	// For now, return upcoming events as a placeholder
	return s.eventService.GetUpcomingEvents(limit)
}

// GetTrendingEvents gets currently trending events
func (s *EventDiscoveryService) GetTrendingEvents(limit int) ([]*models.Event, error) {
	// This would implement trending logic based on:
	// - Recent ticket sales velocity
	// - Social media mentions
	// - Search frequency
	// - View counts
	
	// For now, return featured events as a placeholder
	return s.eventService.GetFeaturedEvents(limit)
}

// GetSimilarEvents finds events similar to a given event
func (s *EventDiscoveryService) GetSimilarEvents(eventID int, limit int) ([]*models.Event, error) {
	// Get the reference event
	event, err := s.eventService.GetEventByID(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reference event: %w", err)
	}

	// Find similar events based on:
	// - Same category
	// - Similar location
	// - Similar price range
	// - Similar date range
	
	categoryName := ""
	if event.Category != nil {
		categoryName = event.Category.Name
	}
	
	filters := EventSearchFilters{
		Category: categoryName,
		Location: event.Location,
		Page:     1,
		PerPage:  limit + 1, // +1 to exclude the original event
	}

	events, _, err := s.eventService.SearchEvents(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar events: %w", err)
	}

	// Remove the original event from results
	var similarEvents []*models.Event
	for _, e := range events {
		if e.ID != eventID {
			similarEvents = append(similarEvents, e)
		}
		if len(similarEvents) >= limit {
			break
		}
	}

	return similarEvents, nil
}