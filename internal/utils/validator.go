package utils

import (
	"fmt"
	"net/url"
	"strings"
)

func ValidatorURL(rawURL string) error {
	if rawURL == "" {
		fmt.Errorf("URL cannot be empty")
	}

	if len(rawURL) > 2048 {
		return fmt.Errorf("URL is too long (max 2048 characters)")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must start with http:// or https://")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL must comtain a valid host")
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
