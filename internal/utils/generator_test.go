package utils

import (
	"strings"
	"testing"
)

func TestGenerateShortCode(t *testing.T) {
	code, err := GenerateShortCode()
	if err != nil {
		t.Fatalf("GenerateShortCode() error = %v", err)
	}

	if len(code) != DefaultShortCodeLength {
		t.Errorf("GenerateShortCode() length = %d, want %d", len(code), DefaultShortCodeLength)
	}

	for _, char := range code {
		if !strings.ContainsRune(alphabet, char) {
			t.Errorf("GenerateShortCode() contains invalid character: %c", char)
		}
	}
}

func TestGenerateShortCodeWithLength(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"length 1", 1},
		{"length 4", 4},
		{"length 8", 8},
		{"length 12", 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := GenerateShortCodeWithLength(tt.length)
			if err != nil {
				t.Errorf("GenerateShortCodeWithLength(%d) error = %v", tt.length, err)
				return
			}

			if len(code) != tt.length {
				t.Errorf("GenerateShortCodeWithLength(%d) length = %d, want %d", tt.length, len(code), tt.length)
			}

			for _, char := range code {
				if !strings.ContainsRune(alphabet, char) {
					t.Errorf("GenerateShortCodeWithLength(%d) contains invalid character: %c", tt.length, char)
				}
			}
		})
	}
}

func TestGenerateShortCodeUniqueness(t *testing.T) {
	generated := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		code, err := GenerateShortCode()
		if err != nil {
			t.Fatalf("GenerateShortCode() error = %v", err)
		}

		if generated[code] {
			t.Errorf("GenerateShortCode() generated duplicate: %s", code)
		}
		generated[code] = true
	}
}