package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"

	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/database"
	"event-ticketing-platform/internal/handlers"
	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
	"event-ticketing-platform/internal/server"
	"event-ticketing-platform/internal/services"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"

	// Import packages to ensure they're included in go.mod
	_ "github.com/a-h/templ"
)

func main() {
	// Register types for session serialization
	gob.Register(&models.Cart{})
	gob.Register(models.CartItem{})
	gob.Register([]models.CartItem{})

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize database connection
	dbConfig := database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
		log.Println("Continuing without database connection for initial setup...")

		// Use simple handlers when database is not available
		setupSimpleRoutes(cfg)
		return
	}
	defer db.Close()
	log.Println("Database connection established successfully")

	// Create session store
	sessionStore := sessions.NewCookieStore([]byte(cfg.Session.Secret))

	// Configure session options
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}

	sessionMiddleware := middleware.NewSessionMiddleware(sessionStore)

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db.DB)
	eventRepo := repositories.NewEventRepository(db.DB)
	ticketRepo := repositories.NewTicketRepository(db.DB)
	orderRepo := repositories.NewOrderRepository(db.DB)

	// Initialize real services with database backing
	emailService := services.NewResendEmailService(services.ResendConfig{
		APIKey:    cfg.Resend.APIKey,
		FromEmail: cfg.Resend.FromEmail,
		FromName:  cfg.Resend.FromName,
	})

	// Initialize payment service with Paystack
	paymentService := services.NewPaystackService(services.PaystackConfig{
		SecretKey:   cfg.Paystack.SecretKey,
		PublicKey:   cfg.Paystack.PublicKey,
		Environment: cfg.Paystack.Environment,
		WebhookURL:  cfg.Paystack.WebhookURL,
		CallbackURL: cfg.Paystack.CallbackURL,
	})

	// Initialize Authboss integration
	baseURL := fmt.Sprintf("http://%s:%s", cfg.Server.Host, cfg.Server.Port)
	isDevelopment := cfg.Server.Env == "development"

	authbossIntegration, err := server.NewAuthbossIntegration(db.DB, sessionStore, emailService, baseURL, isDevelopment)
	if err != nil {
		log.Fatal("Failed to initialize Authboss integration:", err)
	}
	defer authbossIntegration.Close()

	// Initialize services that depend on auth
	authService := services.NewAuthService(userRepo, emailService)
	userService := services.NewUserService(userRepo)
	eventService := services.NewEventService(eventRepo, authService, "uploads/events")

	// Initialize PDF service for ticket generation
	pdfService := services.NewPDFService()

	// Initialize ticket service with proper parameters
	ticketService := services.NewTicketService(ticketRepo, orderRepo, paymentService, authService, pdfService, 900) // 15 minutes reservation TTL

	// Initialize order service
	orderService := services.NewOrderService(orderRepo, ticketRepo, userRepo, paymentService, emailService)

	// Initialize analytics service
	analyticsService := services.NewAnalyticsService(db.DB, orderRepo, eventRepo, ticketRepo, userRepo)

	// Initialize storage service (R2 or fallback)
	var storageService services.StorageService
	if cfg.R2.AccessKeyID != "" && cfg.R2.SecretAccessKey != "" {
		r2Service, err := services.NewR2Service(cfg.R2)
		if err != nil {
			log.Printf("Failed to initialize R2 service: %v, using fallback storage", err)
			storageService = services.NewFallbackStorageService("./uploads", "http://localhost:8080/uploads")
		} else {
			storageService = r2Service
			log.Println("R2 storage service initialized successfully")
		}
	} else {
		storageService = services.NewFallbackStorageService("./uploads", "http://localhost:8080/uploads")
		log.Println("Using fallback storage service (R2 credentials not configured)")
	}

	// Initialize image service
	imageService := services.NewImageService(storageService)

	// Test connections
	if err := emailService.TestConnection(); err != nil {
		log.Printf("Email service connection test failed: %v", err)
	}

	// Initialize CSRF middleware
	csrfMiddleware := middleware.NewCSRFMiddleware(sessionStore)

	// Initialize handlers
	publicHandler := handlers.NewPublicHandler(eventService, ticketService)
	dashboardHandler := handlers.NewDashboardHandler(orderService, eventService, ticketService)
	profileHandler := handlers.NewProfileHandler(authService, userService, sessionStore)
	cartHandler := handlers.NewCartHandler(ticketService, eventService, paymentService, sessionStore)
	paymentHandler := handlers.NewPaymentHandler(paymentService, orderService, ticketService, sessionStore)
	imageHandler := handlers.NewImageManagementHandler(imageService, eventService, storageService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService, authService)
	adminHandler := handlers.NewAdminHandler(userService, eventService, orderService)

	// Initialize withdrawal service and handler
	withdrawalRepo := repositories.NewWithdrawalRepository(db.DB)
	withdrawalService := services.NewWithdrawalService(withdrawalRepo)
	withdrawalHandler := handlers.NewWithdrawalHandler(withdrawalService)

	// Initialize audit and event moderation services
	auditRepo := repositories.NewAuditLogRepository(db.DB)
	auditService := services.NewAuditService(auditRepo)
	eventModerationService := services.NewEventModerationService(eventRepo, auditService)
	eventModerationHandler := handlers.NewEventModerationHandler(eventModerationService)

	// Initialize settings service and handler
	settingsRepo := repositories.NewSettingsRepository(db.DB)
	settingsService := services.NewSettingsService(settingsRepo)
	adminSettingsHandler := handlers.NewAdminSettingsHandler(settingsService)

	// Initialize default settings
	if err := settingsService.InitializeDefaultSettings(); err != nil {
		log.Printf("Failed to initialize default settings: %v", err)
	}

	// Initialize router
	r := chi.NewRouter()

	// Basic middleware
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CORSMiddleware(middleware.DefaultCORSConfig()))
	r.Use(sessionMiddleware.SessionConfig)
	r.Use(authbossIntegration.GetLoadUserMiddleware()) // Use Authboss load user middleware

	// Add security validation middleware
	authbossMiddleware := middleware.NewAuthbossMiddleware(authbossIntegration.GetAuthbossConfig())
	r.Use(authbossMiddleware.SecurityValidation(sessionStore))

	r.Use(csrfMiddleware.EnsureCSRFToken)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	// Uploads files
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads/"))))

	// Public routes
	r.Get("/", publicHandler.HomePage)
	r.Get("/events", publicHandler.EventsListPage)
	r.Get("/events/{id}", publicHandler.EventDetailsPage)
	r.Get("/events/{id}/availability", publicHandler.GetTicketAvailability)
	r.Get("/search", publicHandler.SearchEvents)

	// Additional public routes
	r.Get("/categories", func(w http.ResponseWriter, r *http.Request) {
		// Redirect to events page with category filter for now
		http.Redirect(w, r, "/events", http.StatusTemporaryRedirect)
	})
	r.Get("/pricing", func(w http.ResponseWriter, r *http.Request) {
		// Simple pricing page placeholder
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head><title>Pricing - Event Ticketing Platform</title></head>
			<body style="font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px;">
				<h1>Pricing Plans</h1>
				<p>Our pricing plans are coming soon. For now, event creation is free!</p>
				<a href="/" style="color: #3B82F6;">← Back to Home</a>
			</body>
			</html>
		`))
	})
	r.Get("/help", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head><title>Help Center - Event Ticketing Platform</title></head>
			<body style="font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px;">
				<h1>Help Center</h1>
				<p>Help documentation is coming soon. For support, please contact us.</p>
				<a href="/" style="color: #3B82F6;">← Back to Home</a>
			</body>
			</html>
		`))
	})
	r.Get("/contact", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head><title>Contact Us - Event Ticketing Platform</title></head>
			<body style="font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px;">
				<h1>Contact Us</h1>
				<p>Contact form is coming soon. For immediate support, please email us.</p>
				<a href="/" style="color: #3B82F6;">← Back to Home</a>
			</body>
			</html>
		`))
	})
	r.Get("/resources", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head><title>Resources - Event Ticketing Platform</title></head>
			<body style="font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px;">
				<h1>Resources</h1>
				<p>Resource library is coming soon. Check back later for guides and tutorials.</p>
				<a href="/" style="color: #3B82F6;">← Back to Home</a>
			</body>
			</html>
		`))
	})

	// Setup Authboss authentication routes (replaces old /auth routes)
	authbossIntegration.SetupAuthRoutes(r)

	// Shopping cart and checkout routes
	r.Route("/cart", func(r chi.Router) {
		r.Use(authbossIntegration.GetRequireAuthMiddleware())
		r.Use(csrfMiddleware.CSRFProtection) // Add CSRF protection
		r.Get("/", cartHandler.ViewCart)
		r.Post("/add", cartHandler.AddToCartUnified)
		r.Post("/clear", cartHandler.ClearCart)
		r.Post("/update", cartHandler.UpdateCartItem)
	})

	r.Route("/events/{id}/cart", func(r chi.Router) {
		r.Use(authbossIntegration.GetRequireAuthMiddleware())
		r.Use(csrfMiddleware.CSRFProtection) // Add CSRF protection
		r.Post("/add", cartHandler.AddToCart)
	})

	r.Route("/checkout", func(r chi.Router) {
		r.Use(authbossIntegration.GetRequireAuthMiddleware())
		r.Use(csrfMiddleware.CSRFProtection) // Add CSRF protection
		r.Get("/", cartHandler.CheckoutPage)
		r.Post("/", cartHandler.ProcessCheckout)
	})

	// Payment routes (for Paystack callbacks and status)
	r.Route("/payment", func(r chi.Router) {
		r.Get("/callback", paymentHandler.PaymentCallback)  // Pesapal callback (no auth required)
		r.Post("/ipn", paymentHandler.PaymentIPN)           // Pesapal IPN (no auth required)
		r.Get("/diagnose", paymentHandler.DiagnosePesapal)  // Diagnostic endpoint
		r.Get("/redirect", paymentHandler.PaymentRedirect)  // Paystack redirect handler
		r.Get("/{status}", paymentHandler.PaymentStatus)    // Payment status pages
		r.Post("/initiate", paymentHandler.InitiatePayment) // For testing (requires auth)
	})

	// Protected dashboard routes
	r.Route("/dashboard", func(r chi.Router) {
		r.Use(authbossIntegration.GetRequireAuthMiddleware())
		r.Get("/", dashboardHandler.DashboardPage)
		r.Get("/orders", dashboardHandler.OrdersPage)
		r.Get("/orders/{id}", dashboardHandler.OrderDetailsPage)
		r.Post("/orders/{id}/cancel", dashboardHandler.CancelOrder)
		r.Get("/orders/{id}/tickets/download", dashboardHandler.DownloadTickets)
		r.Get("/orders/{id}/tickets/redownload", dashboardHandler.RedownloadTickets)
		r.Get("/tickets/{id}/download", dashboardHandler.DownloadSingleTicket)

		// Profile management routes
		r.Get("/profile", profileHandler.ProfilePage)
		r.Post("/profile", profileHandler.UpdateProfile)
		r.Get("/security", profileHandler.SecurityPage)
		r.Post("/security/change-password", profileHandler.ChangePassword)
		r.Get("/settings", profileHandler.SettingsPage)
		r.Post("/settings", profileHandler.UpdateSettings)
		r.Get("/delete-account", profileHandler.DeleteAccountPage)
		r.Post("/delete-account", profileHandler.DeleteAccount)
	})

	// Order confirmation route (separate from dashboard for direct access)
	r.Route("/orders/{id}/confirmation", func(r chi.Router) {
		r.Use(authbossIntegration.GetRequireAuthMiddleware())
		r.Get("/", dashboardHandler.OrderConfirmationPage)
	})

	// Organizer routes for event and image management
	organizerEventHandler := handlers.NewOrganizerEventHandler(eventService, ticketService, storageService, imageService)
	ticketTypeHandler := handlers.NewTicketTypeHandler(ticketService, eventService)

	r.Route("/organizer", func(r chi.Router) {
		r.Use(authbossIntegration.GetRequireAuthMiddleware())
		r.Use(authbossIntegration.GetRequireRoleMiddleware(string(models.UserRoleOrganizer)))
		r.Use(csrfMiddleware.CSRFProtection) // Add CSRF protection to POST routes

		// Analytics and dashboard routes
		r.Get("/dashboard", analyticsHandler.OrganizerDashboard)

		// Event management routes
		r.Get("/events", organizerEventHandler.EventsListPage)
		r.Get("/events/create", organizerEventHandler.CreateEventPage)
		r.Get("/events/new", organizerEventHandler.CreateEventPage) // Add alias for backward compatibility
		r.Post("/events", organizerEventHandler.CreateEventSubmit)
		r.Get("/events/{id}/edit", organizerEventHandler.EditEventPage)
		r.Put("/events/{id}", organizerEventHandler.UpdateEventSubmit)
		r.Post("/events/{id}", organizerEventHandler.UpdateEventSubmit) // For forms that can't use PUT
		r.Delete("/events/{id}", organizerEventHandler.DeleteEvent)
		r.Post("/events/{id}/duplicate", organizerEventHandler.DuplicateEvent)
		r.Post("/events/{id}/status", organizerEventHandler.UpdateEventStatus)
		r.Post("/events/{id}/publish", organizerEventHandler.PublishEvent)
		r.Post("/events/{id}/unpublish", organizerEventHandler.UnpublishEvent)

		// Withdrawal routes
		r.Get("/withdrawals", withdrawalHandler.WithdrawalsPage)
		r.Get("/withdrawals/create", withdrawalHandler.CreateWithdrawalPage)
		r.Post("/withdrawals/create", withdrawalHandler.CreateWithdrawalSubmit)

		// Event analytics routes
		r.Get("/events/{id}/analytics", analyticsHandler.EventAnalytics)
		r.Get("/events/{id}/export-attendees", analyticsHandler.ExportAttendees)

		// Ticket type management routes
		r.Route("/events/{eventId}/tickets", func(r chi.Router) {
			r.Get("/", ticketTypeHandler.TicketTypesPage)
			r.Get("/create", ticketTypeHandler.CreateTicketTypePage)
			r.Post("/", ticketTypeHandler.CreateTicketTypeSubmit)
			r.Get("/{id}/edit", ticketTypeHandler.EditTicketTypePage)
			r.Put("/{id}", ticketTypeHandler.UpdateTicketTypeSubmit)
			r.Post("/{id}", ticketTypeHandler.UpdateTicketTypeSubmit) // For forms that can't use PUT
			r.Delete("/{id}", ticketTypeHandler.DeleteTicketType)
		})

		// Image management routes
		r.Route("/events/{eventId}/images", func(r chi.Router) {
			r.Get("/", imageHandler.ImageGalleryPage)
			r.Post("/upload", imageHandler.UploadImage)
			r.Post("/replace", imageHandler.ReplaceImage)
			r.Delete("/delete", imageHandler.DeleteImage)
			r.Post("/presigned-url", imageHandler.GeneratePresignedURL)
			r.Get("/{imageKey}/variants", imageHandler.GetImageVariants)
		})
	})

	// API routes for HTMX requests
	r.Route("/api", func(r chi.Router) {
		r.Use(authbossIntegration.GetRequireAuthMiddleware())

		// Analytics API routes
		r.Route("/organizer", func(r chi.Router) {
			r.Use(authbossIntegration.GetRequireRoleMiddleware(string(models.UserRoleOrganizer)))
			r.Get("/dashboard", analyticsHandler.DashboardAPI)
			r.Get("/events/{id}/analytics", analyticsHandler.EventAnalyticsAPI)
		})
	})

	// Admin routes
	r.Route("/admin", func(r chi.Router) {
		r.Use(authbossIntegration.GetRequireAuthMiddleware())
		r.Use(authbossIntegration.GetRequireRoleMiddleware(string(models.UserRoleAdmin)))
		r.Use(csrfMiddleware.CSRFProtection) // Add CSRF protection to POST routes

		// Admin dashboard
		r.Get("/", adminHandler.AdminDashboard)

		// User management
		r.Get("/users", adminHandler.UserManagement)
		r.Post("/users/{id}/role", adminHandler.UpdateUserRole)
		r.Post("/users/{id}/suspend", adminHandler.SuspendUser)
		r.Post("/users/{id}/activate", adminHandler.ActivateUser)

		// Category management
		r.Get("/categories", adminHandler.CategoryManagement)
		r.Get("/categories/create", adminHandler.CreateCategoryPage)
		r.Post("/categories", adminHandler.CreateCategory)
		r.Get("/categories/{id}/edit", adminHandler.EditCategoryPage)
		r.Post("/categories/{id}", adminHandler.UpdateCategory)
		r.Delete("/categories/{id}", adminHandler.DeleteCategory)

		// Withdrawal management
		r.Get("/withdrawals", withdrawalHandler.AdminWithdrawalsPage)
		r.Post("/withdrawals/{id}/status", withdrawalHandler.UpdateWithdrawalStatus)

		// Event moderation
		r.Get("/events/moderate", eventModerationHandler.AdminEventModerationPage)
		r.Post("/events/{id}/moderate", eventModerationHandler.ModerateEvent)

		// System settings
		r.Get("/settings", adminSettingsHandler.SettingsPage)
		r.Post("/settings", adminSettingsHandler.UpdateSettings)
	})

	// Moderator routes
	r.Route("/moderator", func(r chi.Router) {
		r.Use(authbossIntegration.GetRequireAuthMiddleware())
		r.Use(authbossIntegration.GetRequireRoleMiddleware(string(models.UserRoleModerator)))
		r.Use(csrfMiddleware.CSRFProtection) // Add CSRF protection to POST routes

		// Moderator dashboard
		r.Get("/dashboard", eventModerationHandler.ModeratorDashboard)

		// Event moderation
		r.Get("/events", eventModerationHandler.AdminEventModerationPage)
		r.Post("/events/{id}/moderate", eventModerationHandler.ModerateEvent)
	})

	r.Get("/categories", publicHandler.CategoriesPage)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"event-ticketing-platform","auth":"authboss"}`))
	})

	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s (Environment: %s) - Authboss Integration", serverAddr, cfg.Server.Env)
	log.Fatal(http.ListenAndServe(serverAddr, r))
}

func setupSimpleRoutes(cfg *config.Config) {
	// Initialize simple handlers for when database is not available
	eventService := &services.MockEventService{}
	ticketService := &services.MockTicketService{}

	simpleHandler := handlers.NewSimplePublicHandler(eventService, ticketService)

	r := chi.NewRouter()

	// Basic middleware
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	// Simple routes
	r.Get("/", simpleHandler.HomePage)
	r.Get("/events", simpleHandler.EventsListPage)
	r.Get("/events/{id}", simpleHandler.EventDetailsPage)
	r.Get("/search", simpleHandler.SearchEvents)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"event-ticketing-platform"}`))
	})

	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s (Environment: %s) - Simple Mode", serverAddr, cfg.Server.Env)
	log.Fatal(http.ListenAndServe(serverAddr, r))
}
