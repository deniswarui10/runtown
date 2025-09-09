package models

import (
	"testing"
)

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid user",
			user: User{
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Role:      RoleAttendee,
			},
			wantErr: false,
		},
		{
			name: "invalid email - empty",
			user: User{
				Email:     "",
				FirstName: "John",
				LastName:  "Doe",
				Role:      RoleAttendee,
			},
			wantErr: true,
			errMsg:  "email is required",
		},
		{
			name: "invalid email - format",
			user: User{
				Email:     "invalid-email",
				FirstName: "John",
				LastName:  "Doe",
				Role:      RoleAttendee,
			},
			wantErr: true,
			errMsg:  "email format is invalid",
		},
		{
			name: "invalid first name - empty",
			user: User{
				Email:     "test@example.com",
				FirstName: "",
				LastName:  "Doe",
				Role:      RoleAttendee,
			},
			wantErr: true,
			errMsg:  "first name is required",
		},
		{
			name: "invalid last name - empty",
			user: User{
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "",
				Role:      RoleAttendee,
			},
			wantErr: true,
			errMsg:  "last name is required",
		},
		{
			name: "invalid role",
			user: User{
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Role:      "invalid",
			},
			wantErr: true,
			errMsg:  "invalid user role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("User.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("User.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestUserCreateRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UserCreateRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: UserCreateRequest{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
				Role:      RoleAttendee,
			},
			wantErr: false,
		},
		{
			name: "invalid password - too short",
			req: UserCreateRequest{
				Email:     "test@example.com",
				Password:  "short",
				FirstName: "John",
				LastName:  "Doe",
				Role:      RoleAttendee,
			},
			wantErr: true,
			errMsg:  "password must be at least 8 characters long",
		},
		{
			name: "invalid password - empty",
			req: UserCreateRequest{
				Email:     "test@example.com",
				Password:  "",
				FirstName: "John",
				LastName:  "Doe",
				Role:      RoleAttendee,
			},
			wantErr: true,
			errMsg:  "password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("UserCreateRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("UserCreateRequest.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestUser_FullName(t *testing.T) {
	user := User{
		FirstName: "John",
		LastName:  "Doe",
	}
	
	expected := "John Doe"
	if got := user.FullName(); got != expected {
		t.Errorf("User.FullName() = %v, want %v", got, expected)
	}
}

func TestUser_RoleChecks(t *testing.T) {
	tests := []struct {
		name string
		role UserRole
		checks map[string]bool
	}{
		{
			name: "admin user",
			role: RoleAdmin,
			checks: map[string]bool{
				"IsAdmin":        true,
				"IsOrganizer":    false,
				"IsAttendee":     false,
				"CanCreateEvents": true,
				"CanManageUsers": true,
			},
		},
		{
			name: "organizer user",
			role: RoleOrganizer,
			checks: map[string]bool{
				"IsAdmin":        false,
				"IsOrganizer":    true,
				"IsAttendee":     false,
				"CanCreateEvents": true,
				"CanManageUsers": false,
			},
		},
		{
			name: "attendee user",
			role: RoleAttendee,
			checks: map[string]bool{
				"IsAdmin":        false,
				"IsOrganizer":    false,
				"IsAttendee":     true,
				"CanCreateEvents": false,
				"CanManageUsers": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := User{Role: tt.role}
			
			if got := user.IsAdmin(); got != tt.checks["IsAdmin"] {
				t.Errorf("User.IsAdmin() = %v, want %v", got, tt.checks["IsAdmin"])
			}
			if got := user.IsOrganizer(); got != tt.checks["IsOrganizer"] {
				t.Errorf("User.IsOrganizer() = %v, want %v", got, tt.checks["IsOrganizer"])
			}
			if got := user.IsAttendee(); got != tt.checks["IsAttendee"] {
				t.Errorf("User.IsAttendee() = %v, want %v", got, tt.checks["IsAttendee"])
			}
			if got := user.CanCreateEvents(); got != tt.checks["CanCreateEvents"] {
				t.Errorf("User.CanCreateEvents() = %v, want %v", got, tt.checks["CanCreateEvents"])
			}
			if got := user.CanManageUsers(); got != tt.checks["CanManageUsers"] {
				t.Errorf("User.CanManageUsers() = %v, want %v", got, tt.checks["CanManageUsers"])
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid email",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
			errMsg:  "email is required",
		},
		{
			name:    "invalid format - no @",
			email:   "testexample.com",
			wantErr: true,
			errMsg:  "email format is invalid",
		},
		{
			name:    "invalid format - no domain",
			email:   "test@",
			wantErr: true,
			errMsg:  "email format is invalid",
		},
		{
			name:    "invalid format - no TLD",
			email:   "test@example",
			wantErr: true,
			errMsg:  "email format is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("validateEmail() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
			errMsg:   "password is required",
		},
		{
			name:     "too short password",
			password: "short",
			wantErr:  true,
			errMsg:   "password must be at least 8 characters long",
		},
		{
			name:     "minimum length password",
			password: "12345678",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("validatePassword() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}