package utils

import (
	"strings"
	"testing"

	apperrors "github.com/Kosench/go-url-shortener/internal/errors"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errType string
	}{
		{
			name:    "valid http URL",
			url:     "http://example.com",
			wantErr: false,
		},
		{
			name:    "valid https URL",
			url:     "https://google.com/search?q=test",
			wantErr: false,
		},
		{
			name:    "valid URL with path and query",
			url:     "https://api.github.com/repos/user/repo?sort=updated",
			wantErr: false,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
			errType: "validation",
		},
		{
			name:    "URL without scheme",
			url:     "example.com",
			wantErr: true,
			errType: "validation",
		},
		{
			name:    "URL with invalid scheme",
			url:     "ftp://example.com",
			wantErr: true,
			errType: "validation",
		},
		{
			name:    "URL without host",
			url:     "https://",
			wantErr: true,
			errType: "validation",
		},
		{
			name:    "invalid URL format",
			url:     "not-a-url",
			wantErr: true,
			errType: "validation",
		},
		{
			name:    "URL too long",
			url:     "https://example.com/" + strings.Repeat("a", 2100),
			wantErr: true,
			errType: "validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateURL() expected error, got nil")
					return
				}
				
				if tt.errType == "validation" && !apperrors.IsValidationError(err) {
					t.Errorf("ValidateURL() expected validation error, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateURL() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "string with spaces",
			input:    "  https://example.com  ",
			expected: "https://example.com",
		},
		{
			name:     "string with control characters",
			input:    "https://example.com\x00\x01\x02",
			expected: "https://example.com",
		},
		{
			name:     "string with tabs and newlines",
			input:    "https://example.com\t\n\r",
			expected: "https://example.com",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only spaces",
			input:    "   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeInput() = %q, want %q", result, tt.expected)
			}
		})
	}
}