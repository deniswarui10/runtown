package repositories

import (
	"testing"

	"event-ticketing-platform/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestUserRepository_New(t *testing.T) {
	repo := NewUserRepository(nil)
	assert.NotNil(t, repo)
}

func TestUserModel_FullName(t *testing.T) {
	user := &models.User{
		FirstName: "John",
		LastName:  "Doe",
	}
	assert.Equal(t, "John Doe", user.FullName())
}

func TestUserModel_RoleChecks(t *testing.T) {
	t.Run("organizer role", func(t *testing.T) {
		user := &models.User{
			Role: models.RoleOrganizer,
		}
		assert.True(t, user.IsOrganizer())
		assert.False(t, user.IsAdmin())
		assert.False(t, user.IsAttendee())
		assert.True(t, user.CanCreateEvents())
		assert.False(t, user.CanManageUsers())
	})
	
	t.Run("admin role", func(t *testing.T) {
		user := &models.User{
			Role: models.RoleAdmin,
		}
		assert.False(t, user.IsOrganizer())
		assert.True(t, user.IsAdmin())
		assert.False(t, user.IsAttendee())
		assert.True(t, user.CanCreateEvents())
		assert.True(t, user.CanManageUsers())
	})
	
	t.Run("attendee role", func(t *testing.T) {
		user := &models.User{
			Role: models.RoleAttendee,
		}
		assert.False(t, user.IsOrganizer())
		assert.False(t, user.IsAdmin())
		assert.True(t, user.IsAttendee())
		assert.False(t, user.CanCreateEvents())
		assert.False(t, user.CanManageUsers())
	})
}

func TestUserCreateRequest_Validation(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &models.UserCreateRequest{
			Email:     "test@example.com",
			Password:  "validpassword123",
			FirstName: "John",
			LastName:  "Doe",
			Role:      models.RoleAttendee,
		}
		
		err := req.Validate()
		assert.NoError(t, err)
	})
	
	t.Run("invalid email", func(t *testing.T) {
		req := &models.UserCreateRequest{
			Email:     "invalid-email",
			Password:  "validpassword123",
			FirstName: "John",
			LastName:  "Doe",
			Role:      models.RoleAttendee,
		}
		
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email format is invalid")
	})
	
	t.Run("short password", func(t *testing.T) {
		req := &models.UserCreateRequest{
			Email:     "test@example.com",
			Password:  "short",
			FirstName: "John",
			LastName:  "Doe",
			Role:      models.RoleAttendee,
		}
		
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password must be at least 8 characters")
	})
	
	t.Run("empty first name", func(t *testing.T) {
		req := &models.UserCreateRequest{
			Email:     "test@example.com",
			Password:  "validpassword123",
			FirstName: "",
			LastName:  "Doe",
			Role:      models.RoleAttendee,
		}
		
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "first name is required")
	})
}

func TestUserUpdateRequest_Validation(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &models.UserUpdateRequest{
			FirstName: "Jane",
			LastName:  "Smith",
			Role:      models.RoleOrganizer,
		}
		
		err := req.Validate()
		assert.NoError(t, err)
	})
	
	t.Run("empty last name", func(t *testing.T) {
		req := &models.UserUpdateRequest{
			FirstName: "Jane",
			LastName:  "",
			Role:      models.RoleOrganizer,
		}
		
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "last name is required")
	})
	
	t.Run("invalid role", func(t *testing.T) {
		req := &models.UserUpdateRequest{
			FirstName: "Jane",
			LastName:  "Smith",
			Role:      "invalid_role",
		}
		
		err := req.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user role")
	})
}

func TestUserSearchFilters_Structure(t *testing.T) {
	filters := UserSearchFilters{
		Role:     models.RoleOrganizer,
		Email:    "test@example.com",
		Name:     "John",
		Limit:    10,
		Offset:   0,
		SortBy:   "created_at",
		SortDesc: true,
	}
	
	assert.Equal(t, models.RoleOrganizer, filters.Role)
	assert.Equal(t, "test@example.com", filters.Email)
	assert.Equal(t, "John", filters.Name)
	assert.Equal(t, 10, filters.Limit)
	assert.Equal(t, 0, filters.Offset)
	assert.Equal(t, "created_at", filters.SortBy)
	assert.True(t, filters.SortDesc)
}