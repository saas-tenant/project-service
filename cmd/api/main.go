
package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/saas-tenant/project-service/internal/config"
)

func main() {
	// Load application configuration
	cfg := config.Load()

	// Configure Gin mode
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "project-service",
		})
	})

	v1 := router.Group("/api/v1")
	{
		v1.GET("/hello", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Welcome to project-service API",
			})
		})
	}

	// Start HTTP server
	log.Printf("Starting project-service on port %s", cfg.AppPort)

	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}
