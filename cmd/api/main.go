
package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/saas-tenant/project-service/internal/config"
	"github.com/saas-tenant/project-service/internal/database"
	"github.com/saas-tenant/project-service/internal/handler"
	"github.com/saas-tenant/project-service/internal/repository"
	"github.com/saas-tenant/project-service/internal/service"
	"github.com/saas-tenant/project-service/internal/middleware"
)

func main() {
	cfg := config.Load()

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Database connection
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}
	defer db.Close()

	log.Println("Connected to PostgreSQL successfully")

	// Dependencies
	projectRepo := repository.NewProjectRepository(db)
	projectService := service.NewProjectService(projectRepo)
	projectHandler := handler.NewProjectHandler(projectService)

	// Gin router
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "project-service",
		})
	})

	// Database health check
	router.GET("/health/db", func(c *gin.Context) {
		if err := db.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "database unavailable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "connected",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		projects := v1.Group("/projects")
		  // Require tenant context for every project endpoint.
   		 projects.Use(middleware.RequireTenant())
		{
			projects.POST("", projectHandler.Create)
			projects.GET("", projectHandler.FindAll)
		}
	}

	log.Printf("Starting project-service on port %s", cfg.AppPort)

	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}
