package handler

import (
	"errors"
	apperrors "github.com/Kosench/go-url-shortener/internal/errors"
	"github.com/Kosench/go-url-shortener/internal/model"
	"github.com/Kosench/go-url-shortener/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
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
