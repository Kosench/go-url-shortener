package service

import (
	"context"
	"fmt"
	"github.com/Kosench/go-url-shortener/internal/model"
	"github.com/Kosench/go-url-shortener/internal/repository"
	"github.com/Kosench/go-url-shortener/internal/utils"
	"time"
)

type URLService struct {
	urlRepo    repository.URLRepository
	baseURL    string
	maxRetries int
}

func NewURLService(urlRepo repository.URLRepository, baseURL string) *URLService {
	return &URLService{
		urlRepo:    urlRepo,
		baseURL:    baseURL,
		maxRetries: 5,
	}
}

func (s *URLService) CreateShortURL(ctx context.Context, req *model.CreateURLRequest) (*model.URLResponse, error) {
	if err := utils.ValidatorURL(req.URL); err != nil {
		return nil, fmt.Errorf("validate error: %w", err)
	}

	sanitizedURL := utils.SanitizeInput(req.URL)

	shortCode, err := s.generateUniqueShortCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate short code: %w", err)
	}

	url := &model.URL{
		OriginalURL: sanitizedURL,
		ShortCode:   shortCode,
		ClickCount:  0,
		CreatedAt:   time.Now(),
	}

	if err := s.urlRepo.Create(ctx, url); err != nil {
		return nil, fmt.Errorf("failed to create URL: %w", err)
	}

	return &model.URLResponse{
		ID:          shortCode,
		OriginalURL: url.OriginalURL,
		ShortURL:    s.buildShortURL(shortCode),
		CreatedAt:   url.CreatedAt,
	}, nil
}

func (s *URLService) GetURL(ctx context.Context, shortCode string) (*model.URLResponse, error) {
	if shortCode == "" {
		return nil, fmt.Errorf("short code cannot be empty")
	}

	url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, fmt.Errorf("URL not found")
	}

	return &model.URLResponse{
		ID:          url.ShortCode,
		OriginalURL: url.OriginalURL,
		ShortURL:    s.buildShortURL(url.ShortCode),
		CreatedAt:   url.CreatedAt,
	}, nil
}

func (s *URLService) generateUniqueShortCode(ctx context.Context) (string, error) {
	for attempt := 0; attempt < s.maxRetries; attempt++ {
		code, err := utils.GenerateShortCode()
		if err != nil {
			return "", fmt.Errorf("failed to generate code: %w", err)
		}

		// Проверяем уникальность
		exists, err := s.urlRepo.ExistsByShortCode(ctx, code)
		if err != nil {
			continue // пробуем еще раз
		}

		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique short code after %d attempts", s.maxRetries)
}

func (s *URLService) buildShortURL(shortCode string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, shortCode)
}
