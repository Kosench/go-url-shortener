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
	"regexp"
	"sync"
	"time"
)

var (
	// Валидация short code - только разрешенные символы
	shortCodeRegex = regexp.MustCompile("^[a-zA-Z0-9]{4,10}$")
)

type URLServiceInterface interface {
	CreateShortURL(ctx context.Context, req *model.CreateURLRequest) (*model.URLResponse, error)
	GetURL(ctx context.Context, shortCode string) (*model.URLResponse, error)
	GetOriginalURL(ctx context.Context, shortCode string) (string, error)
	RecordClick(ctx context.Context, shortCode string) error
}

type URLHandler struct {
	urlService  URLServiceInterface
	clickWorker *ClickWorkerPool
}

// ClickWorkerPool для обработки записи кликов
type ClickWorkerPool struct {
	workers  int
	jobQueue chan ClickJob
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

type ClickJob struct {
	ShortCode string
	Service   URLServiceInterface
}

func NewClickWorkerPool(workers int) *ClickWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &ClickWorkerPool{
		workers:  workers,
		jobQueue: make(chan ClickJob, workers*10), // Буфер для очереди
		ctx:      ctx,
		cancel:   cancel,
	}

	// Запускаем воркеров
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

func (p *ClickWorkerPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case job := <-p.jobQueue:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := job.Service.RecordClick(ctx, job.ShortCode); err != nil {
				log.Printf("Failed to record click for shortCode %s: %v", job.ShortCode, err)
			}
			cancel()

		case <-p.ctx.Done():
			return
		}
	}
}

func (p *ClickWorkerPool) AddJob(job ClickJob) {
	select {
	case p.jobQueue <- job:
		// Успешно добавлено в очередь
	default:
		// Очередь полна, логируем и пропускаем
		log.Printf("Click queue is full, dropping click for %s", job.ShortCode)
	}
}

func (p *ClickWorkerPool) Shutdown() {
	p.cancel()
	close(p.jobQueue)
	p.wg.Wait()
}

func NewURLHandler(urlService *service.URLService) *URLHandler {
	return &URLHandler{
		urlService:  urlService,
		clickWorker: NewClickWorkerPool(10), // 10 воркеров для записи кликов
	}
}

// Shutdown для graceful shutdown
func (h *URLHandler) Shutdown() {
	if h.clickWorker != nil {
		h.clickWorker.Shutdown()
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

	// Валидация формата short code
	if !isValidShortCode(shortCode) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid short code format",
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

func (h *URLHandler) RedirectURL(c *gin.Context) {
	shortCode := c.Param("shortCode")

	// Валидация формата short code
	if !isValidShortCode(shortCode) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid short code format",
		})
		return
	}

	// Получаем оригинальный URL
	originalURL, err := h.urlService.GetOriginalURL(c.Request.Context(), shortCode)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Добавляем задачу записи клика в очередь (неблокирующе)
	h.clickWorker.AddJob(ClickJob{
		ShortCode: shortCode,
		Service:   h.urlService,
	})

	// Выполняем редирект (HTTP 302 - Found)
	c.Redirect(http.StatusFound, originalURL)
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

		// Определяем HTTP статус на основе кода ошибки
		statusCode := http.StatusInternalServerError
		if businessErr.Code == "SHORT_CODE_EXISTS" {
			statusCode = http.StatusConflict
		}

		c.JSON(statusCode, gin.H{
			"error":   "business_error",
			"message": businessErr.Message,
			"code":    businessErr.Code,
		})
		return
	}

	// Неизвестная ошибка
	log.Printf("Unexpected error: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "internal_error",
		"message": "An unexpected error occurred",
	})
}

// isValidShortCode проверяет формат короткого кода
func isValidShortCode(shortCode string) bool {
	if shortCode == "" {
		return false
	}
	return shortCodeRegex.MatchString(shortCode)
}
