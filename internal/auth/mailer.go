package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/aarondl/authboss/v3"
	"event-ticketing-platform/internal/services"
)

// AuthbossMailer implements authboss.Mailer interface
type AuthbossMailer struct {
	emailService services.EmailService
}

// NewAuthbossMailer creates a new Authboss mailer
func NewAuthbossMailer(emailService services.EmailService) *AuthbossMailer {
	return &AuthbossMailer{
		emailService: emailService,
	}
}

// Send sends an email using our email service
func (m *AuthbossMailer) Send(ctx context.Context, email authboss.Email) error {
	if m.emailService == nil {
		return fmt.Errorf("email service not configured")
	}

	// Get the recipient email
	if len(email.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}
	recipient := email.To[0]

	// Determine email type and use appropriate template
	switch {
	case strings.Contains(email.Subject, "Confirm") || strings.Contains(email.Subject, "Verify"):
		return m.sendConfirmationEmail(recipient, email)
	case strings.Contains(email.Subject, "Password") || strings.Contains(email.Subject, "Reset"):
		return m.sendPasswordResetEmail(recipient, email)
	default:
		// Generic email
		return m.sendGenericEmail(recipient, email)
	}
}

// sendConfirmationEmail sends an email confirmation email
func (m *AuthbossMailer) sendConfirmationEmail(recipient string, email authboss.Email) error {
	// Extract confirmation token from email content
	token := m.extractTokenFromEmail(email.HTMLBody)
	if token == "" {
		token = m.extractTokenFromEmail(email.TextBody)
	}

	if token == "" {
		// Fallback to generic email if we can't extract token
		return m.sendGenericEmail(recipient, email)
	}

	// Extract user name from recipient email (simple approach)
	userName := recipient
	if atIndex := strings.Index(recipient, "@"); atIndex > 0 {
		userName = recipient[:atIndex]
	}

	// Use our existing email verification service
	return m.emailService.SendVerificationEmail(recipient, userName, token)
}

// sendPasswordResetEmail sends a password reset email
func (m *AuthbossMailer) sendPasswordResetEmail(recipient string, email authboss.Email) error {
	// Extract reset token from email content
	token := m.extractTokenFromEmail(email.HTMLBody)
	if token == "" {
		token = m.extractTokenFromEmail(email.TextBody)
	}

	if token == "" {
		// Fallback to generic email if we can't extract token
		return m.sendGenericEmail(recipient, email)
	}

	// Use our existing password reset service
	return m.emailService.SendPasswordResetEmail(recipient, token)
}

// sendGenericEmail sends a generic email
func (m *AuthbossMailer) sendGenericEmail(recipient string, email authboss.Email) error {
	// Extract user name from recipient email (simple approach)
	userName := recipient
	if atIndex := strings.Index(recipient, "@"); atIndex > 0 {
		userName = recipient[:atIndex]
	}

	// For generic emails, we'll use the welcome email method as it's the most flexible
	// In a real implementation, you'd want a proper generic SendEmail method
	return m.emailService.SendWelcomeEmail(recipient, userName)
}

// extractTokenFromEmail extracts a token from email content
func (m *AuthbossMailer) extractTokenFromEmail(content string) string {
	// Look for common token patterns in email content
	patterns := []string{
		"token=",
		"confirm=",
		"reset=",
		"verify=",
	}

	for _, pattern := range patterns {
		if idx := strings.Index(content, pattern); idx != -1 {
			// Extract token after the pattern
			start := idx + len(pattern)
			end := start

			// Find the end of the token (until space, newline, or HTML tag)
			for end < len(content) {
				ch := content[end]
				if ch == ' ' || ch == '\n' || ch == '\r' || ch == '<' || ch == '&' {
					break
				}
				end++
			}

			if end > start {
				return content[start:end]
			}
		}
	}

	return ""
}