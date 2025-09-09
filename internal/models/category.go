package models

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// Category represents an event category
type Category struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// CategoryCreateRequest represents the data needed to create a new category
type CategoryCreateRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// CategoryUpdateRequest represents the data that can be updated for a category
type CategoryUpdateRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

var (
	// Slug validation regex: lowercase letters, numbers, and hyphens only
	slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)
)

// Validate validates the category data
func (c *Category) Validate() error {
	if err := c.validateName(); err != nil {
		return err
	}
	
	if err := c.validateSlug(); err != nil {
		return err
	}
	
	if err := c.validateDescription(); err != nil {
		return err
	}
	
	return nil
}

// ValidateCreate validates category creation data
func (req *CategoryCreateRequest) Validate() error {
	if err := validateCategoryName(req.Name); err != nil {
		return err
	}
	
	if err := validateCategorySlug(req.Slug); err != nil {
		return err
	}
	
	if err := validateCategoryDescription(req.Description); err != nil {
		return err
	}
	
	return nil
}

// ValidateUpdate validates category update data
func (req *CategoryUpdateRequest) Validate() error {
	if err := validateCategoryName(req.Name); err != nil {
		return err
	}
	
	if err := validateCategorySlug(req.Slug); err != nil {
		return err
	}
	
	if err := validateCategoryDescription(req.Description); err != nil {
		return err
	}
	
	return nil
}

// validateName validates the category name
func (c *Category) validateName() error {
	return validateCategoryName(c.Name)
}

// validateSlug validates the category slug
func (c *Category) validateSlug() error {
	return validateCategorySlug(c.Slug)
}

// validateDescription validates the category description
func (c *Category) validateDescription() error {
	return validateCategoryDescription(c.Description)
}

// validateCategoryName validates a category name
func validateCategoryName(name string) error {
	if name == "" {
		return errors.New("category name is required")
	}
	
	if len(name) > 100 {
		return errors.New("category name must be less than 100 characters")
	}
	
	if strings.TrimSpace(name) == "" {
		return errors.New("category name cannot be only whitespace")
	}
	
	return nil
}

// validateCategorySlug validates a category slug
func validateCategorySlug(slug string) error {
	if slug == "" {
		return errors.New("category slug is required")
	}
	
	if len(slug) > 100 {
		return errors.New("category slug must be less than 100 characters")
	}
	
	if !slugRegex.MatchString(slug) {
		return errors.New("category slug can only contain lowercase letters, numbers, and hyphens")
	}
	
	if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") {
		return errors.New("category slug cannot start or end with a hyphen")
	}
	
	if strings.Contains(slug, "--") {
		return errors.New("category slug cannot contain consecutive hyphens")
	}
	
	return nil
}

// validateCategoryDescription validates a category description
func validateCategoryDescription(description string) error {
	// Description is optional, but if provided, it should not be too long
	if len(description) > 500 {
		return errors.New("category description must be less than 500 characters")
	}
	
	return nil
}

// GenerateSlug generates a URL-friendly slug from the category name
func GenerateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)
	
	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")
	
	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")
	
	// Replace multiple consecutive hyphens with single hyphen
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")
	
	return slug
}

// HasDescription returns true if the category has a description
func (c *Category) HasDescription() bool {
	return strings.TrimSpace(c.Description) != ""
}