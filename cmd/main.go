package main

import (
	"context"
	"github.com/Kosench/go-url-shortener/internal/cache"
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

	// Подключаемся к Redis
	redisClient, err := cache.NewRedisClient(cache.RedisConfig{
		Host:         cfg.Redis.Host,
		Port:         cfg.Redis.Port,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		CacheTTL:     cfg.Redis.CacheTTL,
	})
	if err != nil {
		log.Printf("⚠️  Failed to connect to Redis (running without cache): %v", err)
		// Продолжаем без кэша
		redisClient = nil
	} else {
		defer redisClient.Close()
		log.Println("✅ Successfully connected to Redis")
	}

	var urlRepo repository.URLRepository
	if redisClient != nil {
		urlRepo = repository.NewCachedURLRepository(db, redisClient)

		// Прогреваем кэш популярными URL
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if cachedRepo, ok := urlRepo.(*repository.CachedURLRepository); ok {
				if err := cachedRepo.WarmupCache(ctx, 100); err != nil {
					log.Printf("Failed to warmup cache: %v", err)
				}
			}
		}()
	} else {
		urlRepo = repository.NewPostgresURLRepository(db)
	}

	baseURL := cfg.GetBaseURL()
	urlService := service.NewURLService(urlRepo, baseURL)
	urlHandler := handler.NewURLHandler(urlService)

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.GetAllowedOrigins(),
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// Rate limiting с Redis (если доступен)
	if redisClient != nil {
		router.Use(RedisRateLimitMiddleware(redisClient, 100, time.Minute))
	} else {
		router.Use(InMemoryRateLimitMiddleware(100, time.Minute))
	}

	// Статические файлы и HTML шаблоны
	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/static/*")

	// Базовый endpoint - HTML страница
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Health checks
	router.GET("/health", func(c *gin.Context) {
		response := gin.H{
			"status": "healthy",
			"services": gin.H{
				"database": "checking",
				"cache":    "checking",
			},
		}

		// Проверяем БД
		if err := database.HealthCheck(db); err != nil {
			response["services"].(gin.H)["database"] = "unhealthy"
			response["status"] = "degraded"
		} else {
			response["services"].(gin.H)["database"] = "healthy"
		}

		// Проверяем Redis
		if redisClient != nil {
			if err := redisClient.HealthCheck(c.Request.Context()); err != nil {
				response["services"].(gin.H)["cache"] = "unhealthy"
				response["status"] = "degraded"
			} else {
				response["services"].(gin.H)["cache"] = "healthy"
			}
		} else {
			response["services"].(gin.H)["cache"] = "disabled"
		}

		statusCode := http.StatusOK
		if response["status"] == "degraded" {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, response)
	})

	router.GET("/info", func(c *gin.Context) {
		version, _ := database.GetVersion(db)
		info := gin.H{
			"service":          "URL Shortener",
			"version":          "1.0.0",
			"database_driver":  "pgx",
			"database_version": version,
			"cache_enabled":    redisClient != nil,
		}

		if redisClient != nil {
			info["cache_driver"] = "redis"
		}

		c.JSON(http.StatusOK, info)
	})

	// API routes
	apiV1 := router.Group("/api")
	{
		apiV1.POST("/urls", urlHandler.CreateURL)
		apiV1.GET("/urls/:shortCode", urlHandler.GetURL)

		// Stats endpoint (если есть Redis)
		if redisClient != nil {
			apiV1.GET("/stats", StatsHandler(redisClient))
		}
	}

	router.GET("/:shortCode", urlHandler.RedirectURL)

	// HTTP Server
	srv := &http.Server{
		Addr:           cfg.GetServerAddress(),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Запускаем сервер
	go func() {
		log.Printf("🚀 Server starting on %s", cfg.GetServerAddress())
		log.Printf("📝 API endpoints: POST/GET /api/urls")
		log.Printf("🔗 Redirect endpoint: GET /{shortCode}")
		if redisClient != nil {
			log.Printf("⚡ Cache enabled (Redis)")
		}

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Shutdown context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Останавливаем handler workers
	urlHandler.Shutdown()

	// Останавливаем HTTP сервер
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("✅ Server gracefully stopped")
}

// RedisRateLimitMiddleware - rate limiter с использованием Redis
func RedisRateLimitMiddleware(redis *cache.RedisClient, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		clientIP := c.ClientIP()
		key := cache.CacheKeys.RateLimit(clientIP)

		// Используем Redis для подсчета запросов
		count, err := redis.IncrementRateLimit(ctx, key, window)
		if err != nil {
			log.Printf("Rate limit error: %v", err)
			// При ошибке Redis пропускаем запрос
			c.Next()
			return
		}

		if count > int64(maxRequests) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// InMemoryRateLimitMiddleware - fallback rate limiter без Redis
func InMemoryRateLimitMiddleware(maxRequests int, window time.Duration) gin.HandlerFunc {
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
				"message": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		// Добавляем текущий запрос
		requests[clientIP] = append(requests[clientIP], now)

		c.Next()
	}
}

// StatsHandler - endpoint для статистики
func StatsHandler(redis *cache.RedisClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Простая статистика из Redis
		// В будущем можно расширить
		c.JSON(http.StatusOK, gin.H{
			"message": "Stats endpoint",
			"cache":   "enabled",
		})
	}
}
