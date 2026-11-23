package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/saas-tenant/project-service/internal/events"
	"github.com/saas-tenant/project-service/internal/model"
	"github.com/saas-tenant/project-service/internal/repository"
)

type ProjectService struct {
	repo        *repository.ProjectRepository
	redisClient *redis.Client
	publisher   *events.RabbitMQPublisher
}

func NewProjectService(
	repo *repository.ProjectRepository,
	redisClient *redis.Client,
	publisher *events.RabbitMQPublisher,
) *ProjectService {
	return &ProjectService{
		repo:        repo,
		redisClient: redisClient,
		publisher:   publisher,
	}
}

func (s *ProjectService) Create(
	ctx context.Context,
	tenantID string,
	input model.CreateProjectInput,
) (*model.Project, error) {
	// 1. Persist the project in PostgreSQL.
	project, err := s.repo.Create(ctx, tenantID, input)
	if err != nil {
		return nil, err
	}

	// 2. Invalidate this tenant's project-list cache.
	cacheKey := "projects:tenant:" + tenantID
	if err := s.redisClient.Del(ctx, cacheKey).Err(); err != nil {
		log.Printf("Failed to invalidate project cache: %v", err)
	}

	// 3. Publish the project.created event.
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	event := events.ProjectCreatedEvent{
		Event:     "project.created",
		TenantID:  project.TenantID,
		ProjectID: project.ID,
		Name:      project.Name,
		CreatedAt: project.CreatedAt,
	}

	if err := s.publisher.PublishProjectCreated(publishCtx, event); err != nil {
		log.Printf("Failed to publish project.created event: %v", err)
	}

	return project, nil
}

func (s *ProjectService) FindAll(
	ctx context.Context,
	tenantID string,
) ([]model.Project, error) {
	cacheKey := "projects:tenant:" + tenantID

	// 1. Try Redis first.
	cached, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var projects []model.Project

		if err := json.Unmarshal([]byte(cached), &projects); err == nil {
			log.Println("Projects loaded from Redis")
			return projects, nil
		}

		log.Printf("Invalid project cache; loading from database: %v", err)
	} else if err != redis.Nil {
		log.Printf("Redis cache read failed: %v", err)
	}

	// 2. Cache miss: load from PostgreSQL.
	projects, err := s.repo.FindAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 3. Save the result in Redis for five minutes.
	data, err := json.Marshal(projects)
	if err != nil {
		log.Printf("Failed to serialize projects for cache: %v", err)
		return projects, nil
	}

	if err := s.redisClient.Set(
		ctx,
		cacheKey,
		data,
		5*time.Minute,
	).Err(); err != nil {
		log.Printf("Failed to write project cache: %v", err)
	}

	return projects, nil
}
