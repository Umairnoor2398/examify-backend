package admin

import (
	"github.com/Umairnoor2398/examify-backend/internal/database"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/Umairnoor2398/examify-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type createSchoolRequest struct {
	Username       string `json:"username" binding:"required,min=3"`
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=8"`
	MaxTeachers    int    `json:"max_teachers" binding:"required,min=1"`
	MaxCustomBooks int    `json:"max_custom_books" binding:"required,min=0"`
}

type updateSchoolRequest struct {
	Name          string   `json:"name"`
	Address       string   `json:"address"`
	ContactNumber string   `json:"contact_number"`
	ContactPerson string   `json:"contact_person"`
	CurriculaIDs  []string `json:"curricula_ids"`
}

func ListSchools(c *gin.Context) {
	pg := utils.GetPagination(c)
	search := c.Query("search")

	var schools []models.School
	var total int64

	q := database.DB.Model(&models.School{}).
		Preload("User").
		Preload("Curricula")

	if search != "" {
		q = q.Where("name LIKE ? OR contact_person LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	q.Count(&total)
	q.Limit(pg.PerPage).Offset(pg.Offset).Find(&schools)

	response.Paginated(c, schools, response.PaginationMeta{
		Total:      total,
		Page:       pg.Page,
		PerPage:    pg.PerPage,
		TotalPages: utils.TotalPages(total, pg.PerPage),
	})
}

func CreateSchool(c *gin.Context) {
	var req createSchoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var existing models.User
	if err := database.DB.Where("email = ? OR username = ?", req.Email, req.Username).First(&existing).Error; err == nil {
		response.Conflict(c, "A user with that email or username already exists")
		return
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		response.InternalError(c, "Failed to create account")
		return
	}

	userID := utils.NewUUID()
	user := models.User{
		Base:         models.Base{ID: userID},
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         models.RoleSchoolAdmin,
		IsActive:     true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		response.InternalError(c, "Failed to create account")
		return
	}

	school := models.School{
		Base:           models.Base{ID: utils.NewUUID()},
		UserID:         userID,
		MaxTeachers:    req.MaxTeachers,
		MaxCustomBooks: req.MaxCustomBooks,
	}

	if err := database.DB.Create(&school).Error; err != nil {
		database.DB.Delete(&user)
		response.InternalError(c, "Failed to create school")
		return
	}

	database.DB.Preload("User").Preload("Curricula").First(&school, "id = ?", school.ID)
	response.Created(c, school, "School account created successfully")
}

func GetSchool(c *gin.Context) {
	id := c.Param("id")
	var school models.School
	if err := database.DB.Preload("User").Preload("Curricula").First(&school, "id = ?", id).Error; err != nil {
		response.NotFound(c, "School not found")
		return
	}
	response.Success(c, school, "")
}

func UpdateSchool(c *gin.Context) {
	id := c.Param("id")
	var req updateSchoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var school models.School
	if err := database.DB.First(&school, "id = ?", id).Error; err != nil {
		response.NotFound(c, "School not found")
		return
	}

	updates := map[string]interface{}{
		"name":           req.Name,
		"address":        req.Address,
		"contact_number": req.ContactNumber,
		"contact_person": req.ContactPerson,
	}

	database.DB.Model(&school).Updates(updates)

	if req.CurriculaIDs != nil {
		var curricula []models.Curriculum
		database.DB.Where("id IN ?", req.CurriculaIDs).Find(&curricula)
		database.DB.Model(&school).Association("Curricula").Replace(curricula)
	}

	database.DB.Preload("User").Preload("Curricula").First(&school, "id = ?", id)
	response.Success(c, school, "School updated successfully")
}

func DeleteSchool(c *gin.Context) {
	id := c.Param("id")
	var school models.School
	if err := database.DB.Preload("User").First(&school, "id = ?", id).Error; err != nil {
		response.NotFound(c, "School not found")
		return
	}

	database.DB.Model(&school.User).Update("is_active", false)
	database.DB.Delete(&school)

	response.Success(c, nil, "School deleted successfully")
}

func ToggleSchoolStatus(c *gin.Context) {
	id := c.Param("id")
	var school models.School
	if err := database.DB.Preload("User").First(&school, "id = ?", id).Error; err != nil {
		response.NotFound(c, "School not found")
		return
	}

	newStatus := !school.User.IsActive
	database.DB.Model(&school.User).Update("is_active", newStatus)

	msg := "School activated"
	if !newStatus {
		msg = "School deactivated"
	}
	response.Success(c, nil, msg)
}
