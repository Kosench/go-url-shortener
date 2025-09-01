package main

import (
	"context"
	"github.com/Kosench/go-url-shortener/internal/config"
	"github.com/Kosench/go-url-shortener/internal/database"
	"github.com/Kosench/go-url-shortener/internal/handler"
	"github.com/Kosench/go-url-shortener/internal/repository"
	"github.com/Kosench/go-url-shortener/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config", err)
	}

	db, err := database.Connect(
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName)
	if err != nil {
		log.Fatal("Failed to connect database:", err)
	}
	defer db.Close()

	log.Println("Successfully connected to database")

	baseURL := cfg.GetBaseURL()

	urlRepo := repository.NewPostgresURLRepository(db)
	urlService := service.NewURLService(urlRepo, baseURL)
	urlHandler := handler.NewURLHandler(urlService)

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // В продакшене указать конкретные домены
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))
	router.Use(RateLimitMiddleware(100, time.Minute)) // 100 запросов в минуту

	// Статические файлы и HTML шаблоны
	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/static/*")

	// Базовый endpoint - HTML страница
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		response := gin.H{
			"status": "healthy",
		}

		// Проверяем БД
		if err := database.HealthCheck(db); err != nil {
			response["status"] = "unhealthy"
			response["database"] = "unhealthy"
			response["error"] = err.Error()
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}

		response["database"] = "healthy"
		c.JSON(http.StatusOK, response)
	})

	router.GET("/info", func(c *gin.Context) {
		version, _ := database.GetVersion(db)
		c.JSON(http.StatusOK, gin.H{
			"service":          "URL Shortener",
			"version":          "0.4.1",
			"database_driver":  "pgx",
			"database_version": version,
		})
	})

	apiV1 := router.Group("/api")
	{
		// URL операции
		apiV1.POST("/urls", urlHandler.CreateURL)
		apiV1.GET("/urls/:shortCode", urlHandler.GetURL)
	}

	router.GET("/:shortCode", urlHandler.RedirectURL)

	srv := &http.Server{
		Addr:           cfg.GetServerAddress(),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		log.Printf("Server starting on %s", cfg.GetServerAddress())
		log.Printf("API endpoints: POST/GET /api/urls")
		log.Printf("Redirect endpoint: GET /{shortCode}")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

// RateLimitMiddleware - простой rate limiter (в продакшене использовать Redis)
func RateLimitMiddleware(maxRequests int, window time.Duration) gin.HandlerFunc {
	// Простая in-memory реализация
	// TODO: Заменить на Redis-based rate limiter
	requests := make(map[string][]time.Time)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		// Очищаем старые записи
		if times, exists := requests[clientIP]; exists {
			validTimes := []time.Time{}
			for _, t := range times {
				if now.Sub(t) < window {
					validTimes = append(validTimes, t)
				}
			}
			requests[clientIP] = validTimes
		}

		// Проверяем лимит
		if len(requests[clientIP]) >= maxRequests {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests",
			})
			c.Abort()
			return
		}

		// Добавляем текущий запрос
		requests[clientIP] = append(requests[clientIP], now)

		c.Next()
	}
}
