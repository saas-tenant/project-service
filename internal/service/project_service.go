
package service

import (
	"context"

	"github.com/saas-tenant/project-service/internal/model"
	"github.com/saas-tenant/project-service/internal/repository"
)

type ProjectService struct {
	repo *repository.ProjectRepository
}

func NewProjectService(repo *repository.ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) Create(
	ctx context.Context,
	tenantID string,
	input model.CreateProjectInput,
) (*model.Project, error) {
	return s.repo.Create(ctx, tenantID, input)
}

func (s *ProjectService) FindAll(
	ctx context.Context,
	tenantID string,
) ([]model.Project, error) {
	return s.repo.FindAll(ctx, tenantID)
}
