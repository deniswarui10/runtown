# Event Ticketing Platform Setup Guide

## Quick Start

1. **Clone and Install Dependencies**
   ```bash
   git clone <repository-url>
   cd event-ticketing-platform
   go mod download
   npm install
   ```

2. **Environment Configuration**
   ```bash
   cp .env.example .env.local
   ```
   Edit `.env.local` with your configuration values.

3. **Run the Application**
   ```bash
   # Development mode with hot reload
   make dev
   
   # Or run directly
   go run cmd/server/main.go
   ```

## Authentication Setup

The application includes a working mock authentication system with default users:

- **Test User**: `test@example.com` / `password123`
- **Organizer**: `organizer@example.com` / `organizer123`

You can register new users through the `/auth/register` page.

## Email Integration with Resend

### 1. Get Resend API Key
1. Sign up at [resend.com](https://resend.com)
2. Create a new API key in your dashboard
3. Add your domain and verify it

### 2. Configure Environment Variables
```bash
RESEND_API_KEY=re_your_api_key_here
RESEND_FROM_EMAIL=noreply@yourdomain.com
RESEND_FROM_NAME=Event Ticketing Platform
```

### 3. Features Enabled with Resend
- Password reset emails
- Welcome emails for new users
- Order confirmation emails
- Event reminder emails

## Payment Integration with Pesapal

### 1. Get Pesapal Credentials
1. Sign up at [pesapal.com](https://www.pesapal.com)
2. Get your Consumer Key and Consumer Secret from the dashboard
3. Set up your callback URLs

### 2. Configure Environment Variables
```bash
PESAPAL_CONSUMER_KEY=your_consumer_key
PESAPAL_CONSUMER_SECRET=your_consumer_secret
PESAPAL_ENVIRONMENT=sandbox  # or "production"
PESAPAL_CALLBACK_URL=http://localhost:8080/payment/callback
PESAPAL_IPN_URL=http://localhost:8080/payment/ipn
```

### 3. Features Enabled with Pesapal
- Secure payment processing
- Multiple payment methods (Mobile Money, Cards, etc.)
- Real-time payment status updates
- Automatic refund processing

## Database Setup (Optional)

The application works with mock data by default. For production use:

1. **Install PostgreSQL**
2. **Create Database**
   ```sql
   CREATE DATABASE event_ticketing;
   ```
3. **Configure Environment**
   ```bash
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=your_password
   DB_NAME=event_ticketing
   DB_SSLMODE=disable
   ```
4. **Run Migrations**
   ```bash
   go run cmd/migrate/main.go
   ```

## Development Commands

```bash
# Run tests
make test

# Build application
make build

# Generate templates
make templ

# Run with hot reload
make dev

# Format code
make fmt
```

## Production Deployment

1. **Set Environment Variables**
   - Use production values for all configurations
   - Set `ENV=production`
   - Use strong `SESSION_SECRET`

2. **Build Application**
   ```bash
   make build
   ```

3. **Run Application**
   ```bash
   ./tmp/server
   ```

## Troubleshooting

### Authentication Issues
- Check that mock auth service is properly initialized
- Verify session configuration
- Test with default users first

### Email Issues
- Verify Resend API key is correct
- Check domain verification in Resend dashboard
- Test with mock mode first (no API key)

### Payment Issues
- Verify Pesapal credentials
- Check environment (sandbox vs production)
- Test with mock mode first (no credentials)

### Database Issues
- Ensure PostgreSQL is running
- Check connection parameters
- Run migrations if using database

## Support

For issues and questions:
1. Check the logs for error messages
2. Verify environment configuration
3. Test with mock services first
4. Check API credentials and permissions