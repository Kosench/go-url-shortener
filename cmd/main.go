package main

import (
	"github.com/Kosench/go-url-shortener/internal/config"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config", err)
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
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
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
