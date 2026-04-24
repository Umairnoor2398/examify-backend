package middleware

import (
	"log"

	"github.com/Umairnoor2398/examify-backend/internal/database"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// Audit logs mutating requests (POST/PUT/PATCH/DELETE) to the audit_logs table.
// Must be applied after the Auth middleware so user_id and user_role are set.
func Audit() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == "GET" || method == "OPTIONS" || method == "HEAD" {
			c.Next()
			return
		}

		c.Next()

		userID := c.GetString("user_id")
		userRole := c.GetString("role")
		if userID == "" {
			return
		}

		entry := models.AuditLog{
			ID:         utils.NewUUID(),
			UserID:     userID,
			UserRole:   userRole,
			Action:     method,
			Resource:   c.FullPath(),
			StatusCode: c.Writer.Status(),
		}

		if err := database.DB.Create(&entry).Error; err != nil {
			log.Printf("[AUDIT] failed to write audit log: %v", err)
		}
	}
}
