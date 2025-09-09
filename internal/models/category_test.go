package models

import (
	"testing"
)

func TestCategory_Validate(t *testing.T) {
	tests := []struct {
		name     string
		category Category
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid category",
			category: Category{
				Name:        "Music",
				Slug:        "music",
				Description: "Musical events and concerts",
			},
			wantErr: false,
		},
		{
			name: "invalid name - empty",
			category: Category{
				Name: "",
				Slug: "music",
			},
			wantErr: true,
			errMsg:  "category name is required",
		},
		{
			name: "invalid slug - empty",
			category: Category{
				Name: "Music",
				Slug: "",
			},
			wantErr: true,
			errMsg:  "category slug is required",
		},
		{
			name: "invalid slug - uppercase",
			category: Category{
				Name: "Music",
				Slug: "Music",
			},
			wantErr: true,
			errMsg:  "category slug can only contain lowercase letters, numbers, and hyphens",
		},
		{
			name: "invalid slug - starts with hyphen",
			category: Category{
				Name: "Music",
				Slug: "-music",
			},
			wantErr: true,
			errMsg:  "category slug cannot start or end with a hyphen",
		},
		{
			name: "invalid slug - ends with hyphen",
			category: Category{
				Name: "Music",
				Slug: "music-",
			},
			wantErr: true,
			errMsg:  "category slug cannot start or end with a hyphen",
		},
		{
			name: "invalid slug - consecutive hyphens",
			category: Category{
				Name: "Music",
				Slug: "music--events",
			},
			wantErr: true,
			errMsg:  "category slug cannot contain consecutive hyphens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.category.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Category.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Category.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name string
		input string
		want string
	}{
		{
			name:  "simple name",
			input: "Music",
			want:  "music",
		},
		{
			name:  "name with spaces",
			input: "Arts & Culture",
			want:  "arts-culture",
		},
		{
			name:  "name with special characters",
			input: "Food & Drink!",
			want:  "food-drink",
		},
		{
			name:  "name with multiple spaces",
			input: "Health   &   Wellness",
			want:  "health-wellness",
		},
		{
			name:  "name with leading/trailing spaces",
			input: "  Technology  ",
			want:  "technology",
		},
		{
			name:  "complex name",
			input: "Business & Professional Development",
			want:  "business-professional-development",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateSlug(tt.input); got != tt.want {
				t.Errorf("GenerateSlug() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCategory_HasDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        bool
	}{
		{
			name:        "has description",
			description: "Musical events and concerts",
			want:        true,
		},
		{
			name:        "empty description",
			description: "",
			want:        false,
		},
		{
			name:        "whitespace only description",
			description: "   ",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := Category{Description: tt.description}
			if got := category.HasDescription(); got != tt.want {
				t.Errorf("Category.HasDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateCategorySlug(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid slug",
			slug:    "music",
			wantErr: false,
		},
		{
			name:    "valid slug with numbers",
			slug:    "music2024",
			wantErr: false,
		},
		{
			name:    "valid slug with hyphens",
			slug:    "arts-culture",
			wantErr: false,
		},
		{
			name:    "empty slug",
			slug:    "",
			wantErr: true,
			errMsg:  "category slug is required",
		},
		{
			name:    "slug with uppercase",
			slug:    "Music",
			wantErr: true,
			errMsg:  "category slug can only contain lowercase letters, numbers, and hyphens",
		},
		{
			name:    "slug with spaces",
			slug:    "music events",
			wantErr: true,
			errMsg:  "category slug can only contain lowercase letters, numbers, and hyphens",
		},
		{
			name:    "slug with special characters",
			slug:    "music&events",
			wantErr: true,
			errMsg:  "category slug can only contain lowercase letters, numbers, and hyphens",
		},
		{
			name:    "slug starting with hyphen",
			slug:    "-music",
			wantErr: true,
			errMsg:  "category slug cannot start or end with a hyphen",
		},
		{
			name:    "slug ending with hyphen",
			slug:    "music-",
			wantErr: true,
			errMsg:  "category slug cannot start or end with a hyphen",
		},
		{
			name:    "slug with consecutive hyphens",
			slug:    "music--events",
			wantErr: true,
			errMsg:  "category slug cannot contain consecutive hyphens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCategorySlug(tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCategorySlug() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("validateCategorySlug() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}