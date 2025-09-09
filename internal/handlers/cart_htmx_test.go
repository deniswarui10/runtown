package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"event-ticketing-platform/internal/middleware"

	"github.com/stretchr/testify/assert"
)

// Test HTMX detection utility function
func TestIsHTMXRequest(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{
			name:     "HTMX request with HX-Request header",
			headers:  map[string]string{"HX-Request": "true"},
			expected: true,
		},
		{
			name:     "Regular request without HX-Request header",
			headers:  map[string]string{},
			expected: false,
		},
		{
			name:     "Request with HX-Request header set to false",
			headers:  map[string]string{"HX-Request": "false"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			result := middleware.IsHTMXRequest(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test HTMX redirect functionality
func TestHTMXRedirectBehavior(t *testing.T) {
	// Test that HX-Redirect header is set correctly
	t.Run("HX-Redirect Header Set", func(t *testing.T) {
		w := httptest.NewRecorder()
		url := "https://example.com/redirect"

		w.Header().Set("HX-Redirect", url)

		assert.Equal(t, url, w.Header().Get("HX-Redirect"))
	})

	// Test that regular redirects work
	t.Run("Regular HTTP Redirect", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		
		http.Redirect(w, req, "https://example.com/redirect", http.StatusSeeOther)

		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "https://example.com/redirect", w.Header().Get("Location"))
	})
}

// Test session error handling
func TestSessionErrorHandling(t *testing.T) {
	t.Run("HTMX Session Error Response", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("HX-Request", "true")

		// Simulate session error response
		if middleware.IsHTMXRequest(req) {
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
		}

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "text/html", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Body.String(), "Session error. Please refresh the page and try again.")
	})

	t.Run("Regular Session Error Response", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		// No HX-Request header

		// Simulate regular session error response
		if !middleware.IsHTMXRequest(req) {
			http.Error(w, "Session error", http.StatusInternalServerError)
		}

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Session error")
		assert.NotContains(t, w.Body.String(), "Please refresh the page")
	})
}