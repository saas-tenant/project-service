
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/saas-tenant/project-service/internal/middleware"
	"github.com/saas-tenant/project-service/internal/model"
	"github.com/saas-tenant/project-service/internal/service"
)

type ProjectHandler struct {
	service *service.ProjectService
}

func NewProjectHandler(s *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{service: s}
}

// POST /api/v1/projects
func (h *ProjectHandler) Create(c *gin.Context) {
	var input model.CreateProjectInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	tenantID := c.GetString(middleware.TenantContextKey)

	project, err := h.service.Create(
		c.Request.Context(),
		tenantID,
		input,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create project",
		})
		return
	}

	c.JSON(http.StatusCreated, project)
}

// GET /api/v1/projects
func (h *ProjectHandler) FindAll(c *gin.Context) {
	tenantID := c.GetString(middleware.TenantContextKey)

	projects, err := h.service.FindAll(
		c.Request.Context(),
		tenantID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve projects",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": projects,
	})
}
