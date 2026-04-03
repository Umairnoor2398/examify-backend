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
			"id": user.ID, "username": user.Username, "email": user.Email,
		},
	}, "")
}

type updateProfileRequest struct {
	Name          string `json:"name"`
	Address       string `json:"address"`
	ContactNumber string `json:"contact_number"`
	ContactPerson string `json:"contact_person"`
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

	logoURL, err := utils.SaveUploadedFile(file, header, cfg.Upload.Dir, "logos")
	if err != nil {
		response.InternalError(c, "Failed to upload logo")
		return
	}

	database.DB.Model(school).Update("logo_url", logoURL)
	response.Success(c, gin.H{"logo_url": logoURL}, "Logo uploaded")
}
