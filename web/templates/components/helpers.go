package components

import (
	"context"
)

// getCSRFToken gets the CSRF token from the request context
func getCSRFToken(ctx context.Context) string {
	if token, ok := ctx.Value("csrf_token").(string); ok {
		return token
	}
	return ""
}