package auth

import (
	"database/sql"
	"net/http"

	"github.com/aarondl/authboss/v3"
	"github.com/gorilla/sessions"
)

// Storage combines all Authboss storage interfaces
type Storage struct {
	ServerStorer   *ServerStorer
	RememberStorer *RememberStorer
	SessionStorer  *SessionStateReadWriter
	CookieStorer   *CookieStorer
}

// NewStorage creates a new storage instance with all required storers
func NewStorage(db *sql.DB, sessionStore sessions.Store, sessionName string, secure bool) *Storage {
	return &Storage{
		ServerStorer:   NewServerStorer(db),
		RememberStorer: NewRememberStorer(db),
		SessionStorer:  NewSessionStateReadWriter(sessionStore, sessionName),
		CookieStorer:   NewCookieStorer("authboss", secure),
	}
}

// ConfigureAuthboss configures an Authboss instance with the storage
func (s *Storage) ConfigureAuthboss(ab *authboss.Authboss) {
	// Configure server storage
	ab.Config.Storage.Server = s.ServerStorer

	// Configure session storage
	ab.Config.Storage.SessionState = s.SessionStorer

	// Configure cookie storage  
	ab.Config.Storage.CookieState = s.CookieStorer
}

// ClearSession clears the user session (for logout)
func (s *Storage) ClearSession(w http.ResponseWriter, r *http.Request) error {
	// Clear the session using the session storer
	return s.SessionStorer.ClearSession(w, r)
}

// Cleanup runs cleanup operations on all storers
func (s *Storage) Cleanup() error {
	// Cleanup expired remember tokens
	if err := s.RememberStorer.CleanupExpiredRememberTokens(nil); err != nil {
		return err
	}

	// Additional cleanup operations can be added here
	return nil
}



// Close closes any resources held by the storage
func (s *Storage) Close() error {
	// Currently no resources to close, but this provides a hook for future cleanup
	return nil
}