package handler

import (
	"context"
	"errors"
	apperrors "github.com/Kosench/go-url-shortener/internal/errors"
	"github.com/Kosench/go-url-shortener/internal/model"
	"github.com/Kosench/go-url-shortener/internal/service"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

type URLHandler struct {
	urlService *service.URLService
}

func NewURLHandler(urlService *service.URLService) *URLHandler {
	return &URLHandler{
		urlService: urlService,
	}
}

func (h *URLHandler) CreateURL(c *gin.Context) {
	var req model.CreateURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid JSON format",
		})
		return
	}

	// Создаем URL
	response, err := h.urlService.CreateShortURL(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *URLHandler) GetURL(c *gin.Context) {
	shortCode := c.Param("shortCode")
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Short code is required",
		})
		return
	}

	response, err := h.urlService.GetURL(c.Request.Context(), shortCode)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// handleError обрабатывает ошибки и возвращает соответствующие HTTP коды
func (h *URLHandler) handleError(c *gin.Context, err error) {
	// Проверяем ValidationError
	if apperrors.IsValidationError(err) {
		validationErr := apperrors.GetValidationError(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": validationErr.Message,
			"field":   validationErr.Field,
		})
		return
	}

	// Проверяем URL not found
	if errors.Is(err, apperrors.ErrURLNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "url_not_found",
			"message": "URL not found",
		})
		return
	}

	// Проверяем BusinessError
	if apperrors.IsBusinessError(err) {
		businessErr := apperrors.GetBusinessError(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "business_error",
			"message": businessErr.Message,
			"code":    businessErr.Code,
		})
		return
	}

	// Неизвестная ошибка
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "internal_error",
		"message": "An unexpected error occurred",
	})
}
func (h *URLHandler) RedirectURL(c *gin.Context) {
	shortCode := c.Param("shortCode")

	// Gin всегда вернет какое-то значение для :shortCode, но проверим на всякий случай
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Short code is required",
		})
		return
	}

	// Получаем оригинальный URL
	originalURL, err := h.urlService.GetOriginalURL(c.Request.Context(), shortCode)
	if err != nil {
		h.handleError(c, err)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := h.urlService.RecordClick(ctx, shortCode); err != nil {
			log.Printf("Failed to record click for shortCode %s: %v", shortCode, err)
		}
	}()

	// Выполняем редирект (HTTP 302 - Found)
	c.Redirect(http.StatusFound, originalURL)
}
