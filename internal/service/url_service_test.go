package service

import (
	"context"
	"errors"
	"testing"
	"time"

	apperrors "github.com/Kosench/go-url-shortener/internal/errors"
	"github.com/Kosench/go-url-shortener/internal/model"
)

type mockURLRepository struct {
	urls       map[string]*model.URL
	shouldFail bool
	failCount  int
	callCount  int
}

func newMockURLRepository() *mockURLRepository {
	return &mockURLRepository{
		urls: make(map[string]*model.URL),
	}
}

func (m *mockURLRepository) Create(ctx context.Context, url *model.URL) error {
	if m.shouldFail {
		return errors.New("database error")
	}

	if m.failCount > 0 && m.callCount < m.failCount {
		m.callCount++
		return apperrors.ErrShortCodeExists
	}

	if _, exists := m.urls[url.ShortCode]; exists {
		return apperrors.ErrShortCodeExists
	}

	url.ID = int64(len(m.urls) + 1)
	m.urls[url.ShortCode] = url
	return nil
}

func (m *mockURLRepository) GetByShortCode(ctx context.Context, shortCode string) (*model.URL, error) {
	if m.shouldFail {
		return nil, errors.New("database error")
	}

	url, exists := m.urls[shortCode]
	if !exists {
		return nil, apperrors.ErrURLNotFound
	}

	return url, nil
}

func (m *mockURLRepository) ExistsByShortCode(ctx context.Context, shortCode string) (bool, error) {
	if m.shouldFail {
		return false, errors.New("database error")
	}

	_, exists := m.urls[shortCode]
	return exists, nil
}

func (m *mockURLRepository) IncrementClickCount(ctx context.Context, urlID int64) error {
	if m.shouldFail {
		return errors.New("database error")
	}

	for _, url := range m.urls {
		if url.ID == urlID {
			url.ClickCount++
			return nil
		}
	}

	return apperrors.ErrURLNotFound
}

func TestNewURLService(t *testing.T) {
	repo := newMockURLRepository()
	baseURL := "http://localhost:8080"

	service := NewURLService(repo, baseURL)

	if service.urlRepo == nil {
		t.Error("URLService.urlRepo not set correctly")
	}

	if service.baseURL != baseURL {
		t.Error("URLService.baseURL not set correctly")
	}

	if service.maxRetries != 5 {
		t.Error("URLService.maxRetries should default to 5")
	}
}

func TestURLService_CreateShortURL(t *testing.T) {
	tests := []struct {
		name    string
		request *model.CreateURLRequest
		wantErr bool
		errType string
	}{
		{
			name:    "valid URL",
			request: &model.CreateURLRequest{URL: "https://example.com"},
			wantErr: false,
		},
		{
			name:    "empty URL",
			request: &model.CreateURLRequest{URL: ""},
			wantErr: true,
			errType: "validation",
		},
		{
			name:    "invalid URL",
			request: &model.CreateURLRequest{URL: "not-a-url"},
			wantErr: true,
			errType: "validation",
		},
		{
			name:    "URL without scheme",
			request: &model.CreateURLRequest{URL: "example.com"},
			wantErr: true,
			errType: "validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockURLRepository()
			service := NewURLService(repo, "http://localhost:8080")

			response, err := service.CreateShortURL(context.Background(), tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("CreateShortURL() expected error, got nil")
					return
				}

				if tt.errType == "validation" && !apperrors.IsValidationError(err) {
					t.Errorf("CreateShortURL() expected validation error, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("CreateShortURL() unexpected error = %v", err)
					return
				}

				if response == nil {
					t.Error("CreateShortURL() response is nil")
					return
				}

				if response.ShortCode == "" {
					t.Error("CreateShortURL() response.ShortCode is empty")
				}

				if response.OriginalURL != tt.request.URL {
					t.Errorf("CreateShortURL() response.OriginalURL = %s, want %s", response.OriginalURL, tt.request.URL)
				}

				expectedShortURL := "http://localhost:8080/" + response.ShortCode
				if response.ShortURL != expectedShortURL {
					t.Errorf("CreateShortURL() response.ShortURL = %s, want %s", response.ShortURL, expectedShortURL)
				}

				if response.ClickCount != 0 {
					t.Errorf("CreateShortURL() response.ClickCount = %d, want 0", response.ClickCount)
				}
			}
		})
	}
}

func TestURLService_GetURL(t *testing.T) {
	repo := newMockURLRepository()
	service := NewURLService(repo, "http://localhost:8080")

	url := &model.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		ClickCount:  5,
		CreatedAt:   time.Now(),
	}
	repo.urls["abc123"] = url

	t.Run("existing URL", func(t *testing.T) {
		response, err := service.GetURL(context.Background(), "abc123")
		if err != nil {
			t.Errorf("GetURL() unexpected error = %v", err)
			return
		}

		if response.ShortCode != "abc123" {
			t.Errorf("GetURL() response.ShortCode = %s, want abc123", response.ShortCode)
		}

		if response.OriginalURL != "https://example.com" {
			t.Errorf("GetURL() response.OriginalURL = %s, want https://example.com", response.OriginalURL)
		}
	})

	t.Run("non-existing URL", func(t *testing.T) {
		_, err := service.GetURL(context.Background(), "notfound")
		if err == nil {
			t.Error("GetURL() expected error, got nil")
		}

		if !errors.Is(err, apperrors.ErrURLNotFound) {
			t.Errorf("GetURL() expected ErrURLNotFound, got %v", err)
		}
	})

	t.Run("empty shortCode", func(t *testing.T) {
		_, err := service.GetURL(context.Background(), "")
		if err == nil {
			t.Error("GetURL() expected error for empty shortCode")
		}

		if !apperrors.IsValidationError(err) {
			t.Errorf("GetURL() expected validation error, got %T", err)
		}
	})
}

func TestURLService_GetOriginalURL(t *testing.T) {
	repo := newMockURLRepository()
	service := NewURLService(repo, "http://localhost:8080")

	url := &model.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		ClickCount:  0,
		CreatedAt:   time.Now(),
	}
	repo.urls["abc123"] = url

	t.Run("existing URL", func(t *testing.T) {
		originalURL, err := service.GetOriginalURL(context.Background(), "abc123")
		if err != nil {
			t.Errorf("GetOriginalURL() unexpected error = %v", err)
			return
		}

		if originalURL != "https://example.com" {
			t.Errorf("GetOriginalURL() = %s, want https://example.com", originalURL)
		}
	})

	t.Run("non-existing URL", func(t *testing.T) {
		_, err := service.GetOriginalURL(context.Background(), "notfound")
		if err == nil {
			t.Error("GetOriginalURL() expected error, got nil")
		}

		if !errors.Is(err, apperrors.ErrURLNotFound) {
			t.Errorf("GetOriginalURL() expected ErrURLNotFound, got %v", err)
		}
	})
}

func TestURLService_RecordClick(t *testing.T) {
	repo := newMockURLRepository()
	service := NewURLService(repo, "http://localhost:8080")

	url := &model.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		ClickCount:  0,
		CreatedAt:   time.Now(),
	}
	repo.urls["abc123"] = url

	t.Run("existing URL", func(t *testing.T) {
		err := service.RecordClick(context.Background(), "abc123")
		if err != nil {
			t.Errorf("RecordClick() unexpected error = %v", err)
		}

		if url.ClickCount != 1 {
			t.Errorf("RecordClick() ClickCount = %d, want 1", url.ClickCount)
		}
	})

	t.Run("non-existing URL", func(t *testing.T) {
		err := service.RecordClick(context.Background(), "notfound")
		if err == nil {
			t.Error("RecordClick() expected error, got nil")
		}

		if !errors.Is(err, apperrors.ErrURLNotFound) {
			t.Errorf("RecordClick() expected ErrURLNotFound, got %v", err)
		}
	})

	t.Run("empty shortCode", func(t *testing.T) {
		err := service.RecordClick(context.Background(), "")
		if err == nil {
			t.Error("RecordClick() expected error for empty shortCode")
		}

		if !apperrors.IsValidationError(err) {
			t.Errorf("RecordClick() expected validation error, got %T", err)
		}
	})
}

func TestURLService_CreateShortURL_RetryLogic(t *testing.T) {
	repo := newMockURLRepository()
	repo.failCount = 2 // Fail first 2 attempts, succeed on 3rd
	service := NewURLService(repo, "http://localhost:8080")

	request := &model.CreateURLRequest{URL: "https://example.com"}
	response, err := service.CreateShortURL(context.Background(), request)

	if err != nil {
		t.Errorf("CreateShortURL() with retry logic failed: %v", err)
		return
	}

	if response == nil {
		t.Error("CreateShortURL() response is nil")
		return
	}

	if response.ShortCode == "" {
		t.Error("CreateShortURL() response.ShortCode is empty")
	}
}