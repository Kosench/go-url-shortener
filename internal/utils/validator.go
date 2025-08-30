package utils

import (
	"fmt"
	"net/url"
	"strings"

	apperrors "github.com/Kosench/go-url-shortener/internal/errors"
)

func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return apperrors.NewValidationError("url", "URL cannot be empty")
	}

	if len(rawURL) > 2048 {
		return apperrors.NewValidationError("url", "URL is too long (max 2048 characters)")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return apperrors.NewValidationError("url", fmt.Sprintf("invalid URL format: %v", err))
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return apperrors.NewValidationError("url", "URL must start with http:// or https://")
	}

	if parsedURL.Host == "" {
		return apperrors.NewValidationError("url", "URL must contain a valid host")
	}

	return nil
}

func SanitizeInput(input string) string {
	// Удаляем управляющие символы и обрезаем пробелы
	result := strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return -1 // удаляем символ
		}
		return r
	}, input)

	return strings.TrimSpace(result)
}
