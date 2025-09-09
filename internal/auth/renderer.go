package auth

import (
	"context"
	"fmt"
	"io"

	"github.com/aarondl/authboss/v3"
	"event-ticketing-platform/web/templates/pages"
)

// AuthbossRenderer implements authboss.Renderer interface
type AuthbossRenderer struct{}

// NewAuthbossRenderer creates a new Authboss renderer
func NewAuthbossRenderer() *AuthbossRenderer {
	return &AuthbossRenderer{}
}

// Render renders an Authboss page using our existing templates
func (r *AuthbossRenderer) Render(ctx context.Context, page string, data authboss.HTMLData) (output []byte, contentType string, err error) {
	// Convert authboss data to our template format
	errors := make(map[string][]string)
	formData := make(map[string]string)

	// Extract errors from authboss data
	if errs, ok := data[authboss.DataValidation]; ok {
		if errMap, ok := errs.(map[string][]string); ok {
			errors = errMap
		}
	}

	// Extract form data from authboss data
	if preserved, ok := data[authboss.DataPreserve]; ok {
		if preserveMap, ok := preserved.(map[string]string); ok {
			for field, value := range preserveMap {
				formData[field] = value
			}
		}
	}

	// Add any additional data
	for key, value := range data {
		if strValue, ok := value.(string); ok {
			formData[key] = strValue
		}
	}

	var component interface{}

	// Route to appropriate template based on page
	switch page {
	case "login":
		component = pages.LoginPage(nil, errors, formData)
	case "register":
		component = pages.RegisterPage(nil, errors, formData)
	case "recover_start":
		// For password recovery start page
		component = pages.ForgotPasswordPage(nil, errors, formData, false)
	case "recover_middle":
		// For password recovery email sent confirmation - create a simple success page
		// Since we don't have this template, we'll use a simple message
		return []byte(`
			<!DOCTYPE html>
			<html>
			<head><title>Password Reset Sent</title></head>
			<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; text-align: center;">
				<h1>Check your email</h1>
				<p>We've sent a password reset link to your email address.</p>
				<p><a href="/auth/login">Back to Login</a></p>
			</body>
			</html>
		`), "text/html; charset=utf-8", nil
	case "recover_end":
		// For password reset form
		component = pages.ResetPasswordPage(nil, errors, formData)
	case "confirm":
		// For email confirmation page - create a simple confirmation page
		return []byte(`
			<!DOCTYPE html>
			<html>
			<head><title>Email Verification</title></head>
			<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; text-align: center;">
				<h1>Verify your email</h1>
				<p>Please check your email and click the verification link.</p>
				<p><a href="/auth/login">Back to Login</a></p>
			</body>
			</html>
		`), "text/html; charset=utf-8", nil
	default:
		// Fallback to login page for unknown pages
		component = pages.LoginPage(nil, errors, formData)
	}

	// Render the component
	if templComponent, ok := component.(interface {
		Render(context.Context, io.Writer) error
	}); ok {
		// Create a buffer to capture the output
		var buf []byte
		bufWriter := &bufferWriter{buf: &buf}

		err = templComponent.Render(ctx, bufWriter)
		if err != nil {
			return nil, "", err
		}

		return buf, "text/html; charset=utf-8", nil
	}

	return nil, "", fmt.Errorf("template not found: %s", page)
}

// Load loads data for rendering (not used in our implementation)
func (r *AuthbossRenderer) Load(names ...string) error {
	// We don't need to preload templates since we use Templ
	return nil
}

// bufferWriter implements io.Writer to capture template output
type bufferWriter struct {
	buf *[]byte
}

func (bw *bufferWriter) Write(p []byte) (n int, err error) {
	*bw.buf = append(*bw.buf, p...)
	return len(p), nil
}