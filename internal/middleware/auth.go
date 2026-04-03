package middleware

import (
	"strings"

	"github.com/Umairnoor2398/examify-backend/internal/config"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/Umairnoor2398/examify-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.Unauthorized(c, "Authentication required")
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := utils.ParseToken(tokenStr, cfg.JWT.Secret)
		if err != nil || claims.Type != "access" {
			response.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RequireRole(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.Unauthorized(c, "Authentication required")
			c.Abort()
			return
		}

		userRole := models.Role(role.(string))
		for _, r := range roles {
			if userRole == r {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "You don't have permission to access this resource")
		c.Abort()
	}
}
