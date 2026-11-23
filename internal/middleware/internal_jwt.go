
package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const UserContextKey = "userID"

type InternalClaims struct {
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

func RequireInternalJWT(publicKeyPath string) (gin.HandlerFunc, error) {
	// Load the public key once when the application starts.
	publicKeyPEM, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read JWT public key: %w", err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse JWT public key: %w", err)
	}

	return func(c *gin.Context) {
		// Extract the Bearer token.
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing or invalid Authorization header",
			})
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

		// Verify signature and registered claims.
		claims := &InternalClaims{}

		token, err := jwt.ParseWithClaims(
			tokenString,
			claims,
			func(token *jwt.Token) (any, error) {
				if token.Method != jwt.SigningMethodRS256 {
					return nil, fmt.Errorf("unexpected signing algorithm")
				}
				return publicKey, nil
			},
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithIssuer("saas-auth-service"),
			jwt.WithAudience("project-service"),
			jwt.WithExpirationRequired(),
		)

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired internal token",
			})
			return
		}

		// Require both identities.
		if claims.Subject == "" || claims.TenantID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing user or tenant claims",
			})
			return
		}

		// Store verified identities in Gin context.
		c.Set(UserContextKey, claims.Subject)
		c.Set(TenantContextKey, claims.TenantID)

		c.Next()
	}, nil
}
