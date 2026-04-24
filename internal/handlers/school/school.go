package school

import (
	"github.com/Umairnoor2398/examify-backend/internal/config"
	"github.com/Umairnoor2398/examify-backend/internal/database"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/Umairnoor2398/examify-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func getSchoolByUserID(userID string) (*models.School, error) {
	var school models.School
	err := database.DB.Preload("Curricula").First(&school, "user_id = ?", userID).Error
	return &school, err
}

func GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	var user models.User
	database.DB.First(&user, "id = ?", userID)

	response.Success(c, gin.H{
		"school": school,
		"user": gin.H{
			"id":            user.ID,
			"username":      user.Username,
			"email":         user.Email,
			"contact_email": user.ContactEmail,
			"avatar_url":    user.AvatarURL,
		},
	}, "")
}

type updateProfileRequest struct {
	Name          string `json:"name"`
	Address       string `json:"address"`
	ContactNumber string `json:"contact_number"`
	ContactPerson string `json:"contact_person"`
	ContactEmail  string `json:"contact_email"`
}

func UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	database.DB.Model(school).Updates(map[string]interface{}{
		"name": req.Name, "address": req.Address,
		"contact_number": req.ContactNumber, "contact_person": req.ContactPerson,
	})

	// contact_email lives on the user record
	if req.ContactEmail != "" {
		database.DB.Model(&models.User{}).Where("id = ?", userID).Update("contact_email", req.ContactEmail)
	}
	response.Success(c, school, "Profile updated")
}

func UploadLogo(c *gin.Context, cfg *config.Config) {
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	file, header, err := c.Request.FormFile("logo")
	if err != nil {
		response.BadRequest(c, "Logo image required")
		return
	}
	defer file.Close()

	if err := utils.ValidateFileMIME(file, utils.AllowedImageMIMEs); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	logoURL, err := utils.SaveUploadedFile(file, header, cfg.Upload.Dir, "logos", cfg.Upload.MaxSize)
	if err != nil {
		response.InternalError(c, "Failed to upload logo")
		return
	}

	database.DB.Model(school).Update("logo_url", logoURL)
	response.Success(c, gin.H{"logo_url": logoURL}, "Logo uploaded")
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
