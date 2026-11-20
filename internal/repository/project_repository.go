
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saas-tenant/project-service/internal/model"
)

type ProjectRepository struct {
	db *pgxpool.Pool
}

func NewProjectRepository(db *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create inserts a project belonging to a specific tenant.
func (r *ProjectRepository) Create(
	ctx context.Context,
	tenantID string,
	input model.CreateProjectInput,
) (*model.Project, error) {
	query := `
		INSERT INTO projects.projects (
			tenant_id,
			name,
			description
		)
		VALUES ($1, $2, $3)
		RETURNING
			id::text,
			tenant_id,
			name,
			description,
			status,
			created_at,
			updated_at
	`

	var project model.Project

	err := r.db.QueryRow(
		ctx,
		query,
		tenantID,
		input.Name,
		input.Description,
	).Scan(
		&project.ID,
		&project.TenantID,
		&project.Name,
		&project.Description,
		&project.Status,
		&project.CreatedAt,
		&project.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &project, nil
}

// FindAll returns projects belonging only to the requested tenant.
func (r *ProjectRepository) FindAll(
	ctx context.Context,
	tenantID string,
) ([]model.Project, error) {
	query := `
		SELECT
			id::text,
			tenant_id,
			name,
			description,
			status,
			created_at,
			updated_at
		FROM projects.projects
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]model.Project, 0)

	for rows.Next() {
		var project model.Project

		err := rows.Scan(
			&project.ID,
			&project.TenantID,
			&project.Name,
			&project.Description,
			&project.Status,
			&project.CreatedAt,
			&project.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return projects, nil
}
