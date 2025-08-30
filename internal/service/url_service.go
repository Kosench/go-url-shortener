package service

import (
	"context"
	"fmt"
	apperrors "github.com/Kosench/go-url-shortener/internal/errors"
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
	if err := utils.ValidateURL(req.URL); err != nil {
		return nil, err
	}

	sanitizedURL := utils.SanitizeInput(req.URL)

	shortCode, err := s.generateUniqueShortCode(ctx)
	if err != nil {
		return nil, err
	}

	url := &model.URL{
		OriginalURL: sanitizedURL,
		ShortCode:   shortCode,
		ClickCount:  0,
		CreatedAt:   time.Now(),
	}

	if err := s.urlRepo.Create(ctx, url); err != nil {
		return nil, err
	}

	return &model.URLResponse{
		ID:          url.ID,
		ShortCode:   shortCode,
		OriginalURL: url.OriginalURL,
		ShortURL:    s.buildShortURL(shortCode),
		ClickCount:  url.ClickCount,
		CreatedAt:   url.CreatedAt,
	}, nil
}

func (s *URLService) GetURL(ctx context.Context, shortCode string) (*model.URLResponse, error) {
	if shortCode == "" {
		return nil, apperrors.NewValidationError("shortCode", "short code cannot be empty")
	}

	url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	return &model.URLResponse{
		ID:          url.ID,
		ShortCode:   url.ShortCode,
		OriginalURL: url.OriginalURL,
		ShortURL:    s.buildShortURL(url.ShortCode),
		ClickCount:  url.ClickCount,
		CreatedAt:   url.CreatedAt,
	}, nil
}

func (s *URLService) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	if shortCode == "" {
		return "", apperrors.NewValidationError("shortCode", "short code cannot be empty")
	}

	url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return "", err
	}

	return url.OriginalURL, nil
}

func (s *URLService) RecordClick(ctx context.Context, shortCode string) error {
	if shortCode == "" {
		return apperrors.NewValidationError("shortCode", "short code cannot be empty")
	}

	url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return err
	}

	return s.urlRepo.IncrementClickCount(ctx, url.ID)
}

func (s *URLService) generateUniqueShortCode(ctx context.Context) (string, error) {
	for attempt := 0; attempt < s.maxRetries; attempt++ {
		code, err := utils.GenerateShortCode()
		if err != nil {
			continue // пробуем еще раз
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

	return "", apperrors.NewBusinessError(
		"SHORT_CODE_GENERATION",
		fmt.Sprintf("failed to generate unique short code after %d attempts", s.maxRetries),
		nil,
	)
}

func (s *URLService) buildShortURL(shortCode string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, shortCode)
}
