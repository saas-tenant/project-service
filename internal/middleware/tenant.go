
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const TenantContextKey = "tenantID"

func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))

		if tenantID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "X-Tenant-ID header is required",
			})
			return
		}

		// Store tenant identity in Gin's request context.
		c.Set(TenantContextKey, tenantID)

		c.Next()
	}
}
