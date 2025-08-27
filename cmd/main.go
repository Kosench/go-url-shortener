package main

import (
	"github.com/Kosench/go-url-shortener/internal/config"
	"github.com/Kosench/go-url-shortener/internal/database"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
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

	if version, err := database.GetVersion(db); err == nil {
		log.Println("Connected to database: %s", version[:50]+"...")
	} else {
		log.Println("Successfully connected to database")
	}

	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "URL Shortener",
			"version": "0.3",
			"status":  "development",
		})
	})

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
		apiV1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
			})
		})
	}

	log.Printf("Server starting on %s", cfg.GetServerAddress())
	log.Fatal(router.Run(cfg.GetServerAddress()))
}
