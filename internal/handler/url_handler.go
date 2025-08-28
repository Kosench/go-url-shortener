package handler

import (
	"github.com/Kosench/go-url-shortener/internal/model"
	"github.com/Kosench/go-url-shortener/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
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

	response, err := h.urlService.CreateShortURL(c.Request.Context(), &req)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "url_not_found",
				"message": "URL not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Failed to get URL info",
			})
		}
		return
	}
	c.JSON(http.StatusOK, response)
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
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "url_not_found",
				"message": "URL not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Failed to get URL info",
			})
		}
		return
	}

	c.JSON(http.StatusOK, response)
}

func isValidationError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	validationKeywords := []string{
		"validation",
		"invalid",
		"required",
		"format",
		"length",
	}

	for _, keyword := range validationKeywords {
		if strings.Contains(strings.ToLower(errStr), keyword) {
			return true
		}
	}

	return false
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(strings.ToLower(err.Error()), "not found")
}
