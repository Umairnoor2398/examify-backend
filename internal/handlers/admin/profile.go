package admin

import (
	"strings"

	"github.com/Umairnoor2398/examify-backend/internal/config"
	"github.com/Umairnoor2398/examify-backend/internal/database"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/Umairnoor2398/examify-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	var user models.User
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		response.NotFound(c, "User not found")
		return
	}
	response.Success(c, gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"role":          user.Role,
		"contact_email": user.ContactEmail,
		"avatar_url":    user.AvatarURL,
	}, "")
}

type updateAdminProfileRequest struct {
	ContactEmail string `json:"contact_email"`
}

func UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	var req updateAdminProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	req.ContactEmail = strings.TrimSpace(req.ContactEmail)
	if err := database.DB.Model(&models.User{}).Where("id = ?", userID).Update("contact_email", req.ContactEmail).Error; err != nil {
		response.InternalError(c, "Failed to update profile")
		return
	}
	response.Success(c, nil, "Profile updated")
}

func UploadAvatar(c *gin.Context, cfg *config.Config) {
	userID := c.GetString("user_id")

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		response.BadRequest(c, "Avatar image required")
		return
	}
	defer file.Close()

	if err := utils.ValidateFileMIME(file, utils.AllowedImageMIMEs); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	avatarURL, err := utils.SaveUploadedFile(file, header, cfg.Upload.Dir, "avatars", cfg.Upload.MaxSize)
	if err != nil {
		response.InternalError(c, "Failed to upload avatar")
		return
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", userID).Update("avatar_url", avatarURL).Error; err != nil {
		response.InternalError(c, "Failed to save avatar")
		return
	}
	response.Success(c, gin.H{"avatar_url": avatarURL}, "Avatar uploaded")
}
