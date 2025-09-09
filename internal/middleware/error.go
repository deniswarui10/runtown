package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
)

// ErrorHandlingMiddleware handles panics and errors
func ErrorHandlingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				log.Printf("PANIC: %v\n%s", err, debug.Stack())
				
				// Return appropriate error response
				if IsHTMXRequest(r) {
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
									<p class="text-sm">Something went wrong. Please try again.</p>
								</div>
							</div>
						</div>
					`))
				} else {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// NotFoundHandler handles 404 errors
func NotFoundHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsHTMXRequest(r) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`
				<div class="text-center py-12">
					<svg class="mx-auto h-12 w-12 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 16.172a4 4 0 015.656 0M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
					</svg>
					<h3 class="text-lg font-medium text-gray-900 mb-2">Page Not Found</h3>
					<p class="text-gray-600 mb-6">The page you're looking for doesn't exist.</p>
					<a href="/" class="bg-primary-600 hover:bg-primary-700 text-white px-6 py-3 rounded-lg font-medium transition-colors">
						Go Home
					</a>
				</div>
			`))
		} else {
			// Render full 404 page
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`
				<!DOCTYPE html>
				<html lang="en">
				<head>
					<meta charset="UTF-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<title>Page Not Found - EventHub</title>
					<link href="/static/css/output.css" rel="stylesheet">
				</head>
				<body class="bg-gray-50">
					<div class="min-h-screen flex items-center justify-center">
						<div class="text-center">
							<h1 class="text-6xl font-bold text-gray-900 mb-4">404</h1>
							<h2 class="text-2xl font-semibold text-gray-700 mb-4">Page Not Found</h2>
							<p class="text-gray-600 mb-8">The page you're looking for doesn't exist.</p>
							<a href="/" class="bg-primary-600 hover:bg-primary-700 text-white px-6 py-3 rounded-lg font-medium transition-colors">
								Go Home
							</a>
						</div>
					</div>
				</body>
				</html>
			`))
		}
	})
}

// MethodNotAllowedHandler handles 405 errors
func MethodNotAllowedHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "GET, POST, PUT, DELETE, OPTIONS")
		
		if IsHTMXRequest(r) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`
				<div class="bg-yellow-50 border border-yellow-200 text-yellow-800 p-4 rounded-lg">
					<div class="flex">
						<div class="flex-shrink-0">
							<svg class="h-5 w-5 text-yellow-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16c-.77.833.192 2.5 1.732 2.5z"></path>
							</svg>
						</div>
						<div class="ml-3">
							<p class="text-sm">Method not allowed for this endpoint.</p>
						</div>
					</div>
				</div>
			`))
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
}

// ValidationErrorResponse represents a validation error response
type ValidationErrorResponse struct {
	Success bool                       `json:"success"`
	Errors  map[string][]string       `json:"errors"`
	Message string                    `json:"message"`
}

// HandleValidationErrors creates a middleware for handling validation errors
func HandleValidationErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}