package auth

import (
	"fmt"
	"net/http"

	"github.com/aarondl/authboss/v3"
	"github.com/gorilla/sessions"
)

// SessionState implements authboss.ClientState interface
type SessionState struct {
	values map[string]string
}

// Get implements authboss.ClientState interface
func (ss *SessionState) Get(key string) (string, bool) {
	value, exists := ss.values[key]
	return value, exists
}



// SessionStateReadWriter provides a complete session state management
type SessionStateReadWriter struct {
	store       sessions.Store
	sessionName string
}

// NewSessionStateReadWriter creates a new session state read writer
func NewSessionStateReadWriter(store sessions.Store, sessionName string) *SessionStateReadWriter {
	return &SessionStateReadWriter{
		store:       store,
		sessionName: sessionName,
	}
}

// ReadState reads state from session
func (s *SessionStateReadWriter) ReadState(r *http.Request) (authboss.ClientState, error) {
	session, err := s.store.Get(r, s.sessionName)
	if err != nil {
		return nil, err
	}

	values := make(map[string]string)
	fmt.Printf("[DEBUG] Session values in ReadState: %+v\n", session.Values)
	
	for key, value := range session.Values {
		if strKey, ok := key.(string); ok {
			// Handle both string and integer values
			if strValue, ok := value.(string); ok {
				values[strKey] = strValue
				fmt.Printf("[DEBUG] Added string value: %s = %s\n", strKey, strValue)
			} else if intValue, ok := value.(int); ok {
				values[strKey] = fmt.Sprintf("%d", intValue)
				fmt.Printf("[DEBUG] Added int value: %s = %d (converted to %s)\n", strKey, intValue, fmt.Sprintf("%d", intValue))
			} else {
				fmt.Printf("[DEBUG] Unhandled value type for key %s: %T = %v\n", strKey, value, value)
			}
		} else {
			fmt.Printf("[DEBUG] Non-string key: %T = %v\n", key, key)
		}
	}
	
	fmt.Printf("[DEBUG] Final session state values: %+v\n", values)
	return &SessionState{values: values}, nil
}

// WriteState writes state to session
func (s *SessionStateReadWriter) WriteState(w http.ResponseWriter, state authboss.ClientState, events []authboss.ClientStateEvent) error {
	// Note: This method is called by Authboss but we handle session writing
	// in the login handler directly. This is a placeholder implementation.
	// In a full implementation, you'd need to store the request context
	// to access the session here.
	return nil
}

// DelState deletes state from session
func (s *SessionStateReadWriter) DelState(w http.ResponseWriter, r *http.Request, keys []string) error {
	session, err := s.store.Get(r, s.sessionName)
	if err != nil {
		return err
	}

	for _, key := range keys {
		delete(session.Values, key)
	}

	return session.Save(r, w)
}

// ClearSession clears the entire session (for logout)
func (s *SessionStateReadWriter) ClearSession(w http.ResponseWriter, r *http.Request) error {
	session, err := s.store.Get(r, s.sessionName)
	if err != nil {
		return err
	}

	// Clear all session values
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1 // Mark for deletion
	return session.Save(r, w)
}