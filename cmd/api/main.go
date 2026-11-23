package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/saas-tenant/project-service/internal/cache"
	"github.com/saas-tenant/project-service/internal/config"
	"github.com/saas-tenant/project-service/internal/database"
	"github.com/saas-tenant/project-service/internal/events"
	"github.com/saas-tenant/project-service/internal/handler"
	"github.com/saas-tenant/project-service/internal/middleware"
	"github.com/saas-tenant/project-service/internal/repository"
	"github.com/saas-tenant/project-service/internal/service"
)

func main() {
	// 1. Load application configuration.
	cfg := config.Load()

	// 2. Connect to PostgreSQL.
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to PostgreSQL")

	// 2. Connect to Redis.
	redisClient, err := cache.NewRedisClient(
		cfg.RedisAddr,
		cfg.RedisPassword,
		cfg.RedisDB,
	)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	log.Println("Connected to Redis")

	// 3. Connect to RabbitMQ.

	publisher, err := events.NewRabbitMQPublisher(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer publisher.Close()

	log.Println("Connected to RabbitMQ")

	// 3. Initialize application layers.
	projectRepo := repository.NewProjectRepository(db)
	projectService := service.NewProjectService(
		projectRepo,
		redisClient,
		publisher,
	)
	projectHandler := handler.NewProjectHandler(projectService)

	// 4. Initialize JWT authentication middleware.
	jwtMiddleware, err := middleware.RequireInternalJWT(
		cfg.InternalJWTPublicKeyPath,
	)
	if err != nil {
		log.Fatalf("Failed to initialize JWT middleware: %v", err)
	}

	// 5. Initialize Gin router.
	router := gin.Default()

	// 6. Public health endpoints.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "project-service",
		})
	})

	router.GET("/health/db", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "error",
				"database": "unavailable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"database": "connected",
		})
	})

	// 7. Redis health endpoint.
	router.GET("/health/redis", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "error",
				"redis":  "unavailable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"redis":  "connected",
		})
	})

	// 7. API v1 routes.
	v1 := router.Group("/api/v1")
	{
		projects := v1.Group("/projects")

		// All project routes require a valid internal JWT.
		projects.Use(jwtMiddleware)

		{
			projects.POST("", projectHandler.Create)
			projects.GET("", projectHandler.FindAll)
		}
	}

	// 8. Start HTTP server.
	log.Printf("Project service listening on port %s", cfg.AppPort)

	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
