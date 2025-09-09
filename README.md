# Event Ticketing Platform

A modern event ticketing platform built with Go, similar to Eventbrite, featuring event creation, ticket sales, user management, and secure authentication.

## Tech Stack

- **Backend**: Go with Chi router
- **Database**: PostgreSQL
- **Authentication**: Authboss v3
- **Frontend**: HTMX + TailwindCSS
- **CSS**: TailwindCSS with Bun
- **Development**: Air (hot reloading)

## Features

- 🔐 **Secure Authentication** - Login, registration, email verification
- 🎫 **Event Management** - Create, edit, and manage events
- 🛒 **Ticket Sales** - Shopping cart and checkout system
- 💳 **Payment Integration** - Paystack and Pesapal support
- 👥 **User Roles** - Admin, Organizer, and User roles
- 📧 **Email System** - Transactional emails and notifications
- 🎨 **Modern UI** - Responsive design with TailwindCSS

## Project Structure

```
├── cmd/
│   ├── server/          # Application entry point
│   └── migrate/         # Database migrations
├── internal/
│   ├── auth/           # Authentication (Authboss integration)
│   ├── config/         # Configuration management
│   ├── database/       # Database connection and utilities
│   ├── handlers/       # HTTP handlers
│   ├── middleware/     # HTTP middleware
│   ├── models/         # Data models
│   ├── repositories/   # Data access layer
│   ├── services/       # Business logic layer
│   └── utils/          # Utility functions
├── web/
│   ├── static/         # Static assets (CSS, JS, images)
│   └── templates/      # HTML templates
├── .air.toml          # Air configuration for hot reloading
├── .env.example       # Environment variables template
├── .env.local         # Local development environment
├── package.json       # Node.js dependencies (Bun)
└── tailwind.config.js # TailwindCSS configuration
```

## Quick Start

### Automated Setup (Recommended)

Run the setup script to automatically install dependencies and configure your environment:

```powershell
# Windows PowerShell
.\setup.ps1
```

### Manual Setup

#### Prerequisites

- Go 1.21 or higher
- PostgreSQL 13 or higher
- Bun (Node.js package manager)
- Air (for development hot reloading)

#### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd event-ticketing-platform
   ```

2. **Install dependencies**
   ```bash
   make install
   # or manually:
   go mod download && go mod tidy
   bun install
   ```

3. **Setup environment**
   ```bash
   cp .env.example .env.local
   # Edit .env.local with your database credentials
   ```

4. **Setup development environment**
   ```bash
   make setup
   ```

## Development

### Start Development Server

```bash
# Start with hot reloading and CSS watching (recommended)
make dev

# Or start Air only
make air

# Or run directly
go run ./cmd/server
```

The server will start on `http://localhost:8080`

### Available Commands

```bash
make help           # Show all available commands
make install        # Install dependencies
make setup          # Setup development environment
make dev            # Start development mode (Air + CSS watching)
make air            # Start Air only
make build          # Build the application
make test           # Run tests
make clean          # Clean build artifacts
make css            # Build CSS for production
make css-watch      # Watch CSS changes
make migrate        # Run database migrations
```

### CSS Development

TailwindCSS is configured to work with Bun:

```bash
# Watch CSS changes during development
make css-watch

# Build CSS for production
make css
```

## Environment Variables

Copy `.env.example` to `.env.local` and configure:

### Server Configuration
- `PORT` - Server port (default: 8080)
- `HOST` - Server host (default: localhost)
- `ENV` - Environment (development/production)

### Database Configuration
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`

### Authentication
- `SESSION_SECRET` - Session encryption key
- `AUTHBOSS_ROOT_URL` - Base URL for authentication

### Email Configuration
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`

### Payment Providers
- `PAYSTACK_SECRET_KEY`, `PAYSTACK_PUBLIC_KEY`
- `PESAPAL_CONSUMER_KEY`, `PESAPAL_CONSUMER_SECRET`

## API Endpoints

### Authentication
- `GET /auth/login` - Login page
- `POST /auth/login` - Login form
- `GET /auth/register` - Registration page
- `POST /auth/register` - Registration form
- `GET /auth/logout` - Logout

### Dashboard
- `GET /dashboard` - User dashboard
- `GET /organizer/dashboard` - Organizer dashboard
- `GET /admin` - Admin dashboard

### Events
- `GET /` - Homepage with events
- `GET /events` - Event listing
- `GET /events/{id}` - Event details
- `POST /events` - Create event (organizer)

### Cart & Checkout
- `GET /cart` - Shopping cart
- `POST /cart/add` - Add to cart
- `GET /checkout` - Checkout page
- `POST /checkout` - Process payment

## Development Notes

- **Hot Reloading**: Air watches Go files and templates for changes
- **CSS Processing**: TailwindCSS processes styles with Bun
- **Database**: Graceful connection handling - starts even if DB unavailable
- **Authentication**: Secure session management with Authboss
- **File Uploads**: Handled in `uploads/` directory
- **Static Assets**: Served from `web/static/`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request

## License

This project is licensed under the MIT License.