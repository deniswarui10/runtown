package auth

import (
	"net/http"
	"net/url"
	"time"

	"github.com/aarondl/authboss/v3"
)

// CookieStorer implements authboss.ClientStateReadWriter interface for cookies
type CookieStorer struct {
	cookieName string
	secure     bool
	httpOnly   bool
	sameSite   http.SameSite
	domain     string
	path       string
}

// NewCookieStorer creates a new cookie storer
func NewCookieStorer(cookieName string, secure bool) *CookieStorer {
	return &CookieStorer{
		cookieName: cookieName,
		secure:     secure,
		httpOnly:   true,
		sameSite:   http.SameSiteLaxMode,
		path:       "/",
	}
}

// CookieState implements authboss.ClientState interface
type CookieState struct {
	values map[string]string
}

// Get implements authboss.ClientState interface
func (cs *CookieState) Get(key string) (string, bool) {
	value, exists := cs.values[key]
	return value, exists
}

// ReadState reads state from cookies
func (c *CookieStorer) ReadState(r *http.Request) (authboss.ClientState, error) {
	values := make(map[string]string)

	// Read all cookies and extract authboss-related ones
	for _, cookie := range r.Cookies() {
		if len(cookie.Name) > len(c.cookieName) && 
		   cookie.Name[:len(c.cookieName)] == c.cookieName {
			
			key := cookie.Name[len(c.cookieName)+1:] // +1 for the separator
			value, err := url.QueryUnescape(cookie.Value)
			if err != nil {
				continue // Skip malformed cookies
			}
			values[key] = value
		}
	}

	return &CookieState{values: values}, nil
}

// WriteState writes state to cookies
func (c *CookieStorer) WriteState(w http.ResponseWriter, state authboss.ClientState, events []authboss.ClientStateEvent) error {
	// For now, we'll implement a basic version
	// In a full implementation, you'd process the events to determine what to set/delete
	
	// This is a simplified implementation that would need to be enhanced
	// based on the specific events received
	
	return nil
}

// DelState deletes state from cookies (helper method)
func (c *CookieStorer) DelState(w http.ResponseWriter, r *http.Request, keys []string) error {
	for _, key := range keys {
		cookieName := c.cookieName + "_" + key

		cookie := &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Path:     c.path,
			Domain:   c.domain,
			Secure:   c.secure,
			HttpOnly: c.httpOnly,
			SameSite: c.sameSite,
			Expires:  time.Unix(0, 0), // Expire immediately
			MaxAge:   -1,
		}

		http.SetCookie(w, cookie)
	}

	return nil
}

// RememberCookieStorer handles remember me cookies specifically
type RememberCookieStorer struct {
	*CookieStorer
}

// NewRememberCookieStorer creates a cookie storer for remember me functionality
func NewRememberCookieStorer(secure bool) *RememberCookieStorer {
	return &RememberCookieStorer{
		CookieStorer: NewCookieStorer("authboss_rm", secure),
	}
}

// WriteState writes remember me state with longer expiration
func (r *RememberCookieStorer) WriteState(w http.ResponseWriter, state authboss.ClientState, events []authboss.ClientStateEvent) error {
	// Simplified implementation for remember me cookies
	// In a full implementation, you'd process the events to set appropriate cookies
	
	return nil
}