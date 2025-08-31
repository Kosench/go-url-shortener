package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apperrors "github.com/Kosench/go-url-shortener/internal/errors"
	"github.com/Kosench/go-url-shortener/internal/model"
	"github.com/gin-gonic/gin"
)

type mockURLService struct {
	urls       map[string]*model.URLResponse
	shouldFail bool
	failType   string
}

func newMockURLService() *mockURLService {
	return &mockURLService{
		urls: make(map[string]*model.URLResponse),
	}
}

func (m *mockURLService) CreateShortURL(ctx context.Context, req *model.CreateURLRequest) (*model.URLResponse, error) {
	if m.shouldFail {
		switch m.failType {
		case "validation":
			return nil, apperrors.NewValidationError("url", "invalid URL")
		case "business":
			return nil, apperrors.NewBusinessError("CODE_GEN", "failed to generate code", nil)
		default:
			return nil, errors.New("service error")
		}
	}

	response := &model.URLResponse{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: req.URL,
		ShortURL:    "http://localhost:8080/abc123",
		ClickCount:  0,
		CreatedAt:   time.Now(),
	}

	m.urls["abc123"] = response
	return response, nil
}

func (m *mockURLService) GetURL(ctx context.Context, shortCode string) (*model.URLResponse, error) {
	if m.shouldFail {
		return nil, errors.New("service error")
	}

	if shortCode == "" {
		return nil, apperrors.NewValidationError("shortCode", "short code cannot be empty")
	}

	response, exists := m.urls[shortCode]
	if !exists {
		return nil, apperrors.ErrURLNotFound
	}

	return response, nil
}

func (m *mockURLService) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	if m.shouldFail {
		return "", errors.New("service error")
	}

	if shortCode == "" {
		return "", apperrors.NewValidationError("shortCode", "short code cannot be empty")
	}

	response, exists := m.urls[shortCode]
	if !exists {
		return "", apperrors.ErrURLNotFound
	}

	return response.OriginalURL, nil
}

func (m *mockURLService) RecordClick(ctx context.Context, shortCode string) error {
	if m.shouldFail {
		return errors.New("service error")
	}

	response, exists := m.urls[shortCode]
	if !exists {
		return apperrors.ErrURLNotFound
	}

	response.ClickCount++
	return nil
}

func TestURLHandler_CreateURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*mockURLService)
		expectedStatus int
		expectedFields []string
	}{
		{
			name:        "valid request",
			requestBody: map[string]string{"url": "https://example.com"},
			mockSetup: func(m *mockURLService) {
				// Default success behavior
			},
			expectedStatus: http.StatusCreated,
			expectedFields: []string{"short_code", "original_url", "short_url"},
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"error", "message"},
		},
		{
			name:        "validation error",
			requestBody: map[string]string{"url": "invalid-url"},
			mockSetup: func(m *mockURLService) {
				m.shouldFail = true
				m.failType = "validation"
			},
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"error", "message", "field"},
		},
		{
			name:        "business error",
			requestBody: map[string]string{"url": "https://example.com"},
			mockSetup: func(m *mockURLService) {
				m.shouldFail = true
				m.failType = "business"
			},
			expectedStatus: http.StatusInternalServerError,
			expectedFields: []string{"error", "message", "code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := newMockURLService()
			if tt.mockSetup != nil {
				tt.mockSetup(mockService)
			}

			handler := &URLHandler{urlService: mockService}
			router := gin.New()
			router.POST("/api/urls", handler.CreateURL)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/api/urls", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("CreateURL() status = %d, want %d", w.Code, tt.expectedStatus)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			for _, field := range tt.expectedFields {
				if _, exists := response[field]; !exists {
					t.Errorf("CreateURL() response missing field: %s", field)
				}
			}
		})
	}
}

func TestURLHandler_GetURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := newMockURLService()
	mockService.urls["abc123"] = &model.URLResponse{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		ShortURL:    "http://localhost:8080/abc123",
		ClickCount:  5,
		CreatedAt:   time.Now(),
	}

	handler := &URLHandler{urlService: mockService}
	router := gin.New()
	router.GET("/api/urls/:shortCode", handler.GetURL)

	t.Run("existing URL", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/urls/abc123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GetURL() status = %d, want %d", w.Code, http.StatusOK)
		}

		var response model.URLResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.ShortCode != "abc123" {
			t.Errorf("GetURL() response.ShortCode = %s, want abc123", response.ShortCode)
		}
	})

	t.Run("non-existing URL", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/urls/notfound", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("GetURL() status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestURLHandler_RedirectURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := newMockURLService()
	mockService.urls["abc123"] = &model.URLResponse{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		ShortURL:    "http://localhost:8080/abc123",
		ClickCount:  0,
		CreatedAt:   time.Now(),
	}

	handler := &URLHandler{urlService: mockService}
	router := gin.New()
	router.GET("/:shortCode", handler.RedirectURL)

	t.Run("successful redirect", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/abc123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusFound {
			t.Errorf("RedirectURL() status = %d, want %d", w.Code, http.StatusFound)
		}

		location := w.Header().Get("Location")
		if location != "https://example.com" {
			t.Errorf("RedirectURL() Location = %s, want https://example.com", location)
		}
	})

	t.Run("non-existing URL", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/notfound", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("RedirectURL() status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}