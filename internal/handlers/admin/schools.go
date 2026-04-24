package admin

import (
	"strings"

	"github.com/Umairnoor2398/examify-backend/internal/database"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/Umairnoor2398/examify-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	schoolIDFilter    = "id = ?"
	msgSchoolNotFound = "School not found"
)

type createSchoolRequest struct {
	Username       string `json:"username" binding:"required,min=3"`
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=8"`
	MaxTeachers    int    `json:"max_teachers" binding:"required,min=1"`
	MaxCustomBooks int    `json:"max_custom_books" binding:"required,min=0"`
}

type updateSchoolRequest struct {
	Name           string   `json:"name"`
	Address        string   `json:"address"`
	ContactNumber  string   `json:"contact_number"`
	ContactPerson  string   `json:"contact_person"`
	CurriculaIDs   []string `json:"curricula_ids"`
	MaxTeachers    *int     `json:"max_teachers"`
	MaxCustomBooks *int     `json:"max_custom_books"`
}

// allowedSchoolSortColumns maps frontend sort_by values to safe SQL column expressions.
var allowedSchoolSortColumns = map[string]string{
	"name":             "schools.name",
	"contact_person":   "schools.contact_person",
	"max_teachers":     "schools.max_teachers",
	"max_custom_books": "schools.max_custom_books",
	"created_at":       "schools.created_at",
	"email":            "users.email",
	"status":           "users.is_active",
}

func ListSchools(c *gin.Context) {
	pg := utils.GetPagination(c)
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort_by", "")
	sortOrder := strings.ToUpper(c.DefaultQuery("sort_order", "ASC"))
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}

	var schools []models.School
	var total int64

	q := database.DB.Model(&models.School{}).
		Joins("JOIN users ON users.id = schools.user_id").
		Preload("User").
		Preload("Curricula")

	if search != "" {
		q = q.Where("schools.name LIKE ? OR schools.contact_person LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if col, ok := allowedSchoolSortColumns[sortBy]; ok {
		q = q.Order(col + " " + sortOrder)
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
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

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
	schoolID := utils.NewUUID()

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		user := models.User{
			Base:         models.Base{ID: userID},
			Username:     req.Username,
			Email:        req.Email,
			PasswordHash: hash,
			Role:         models.RoleSchoolAdmin,
			IsActive:     true,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		school := models.School{
			Base:           models.Base{ID: schoolID},
			UserID:         userID,
			MaxTeachers:    req.MaxTeachers,
			MaxCustomBooks: req.MaxCustomBooks,
		}
		return tx.Create(&school).Error
	})
	if err != nil {
		response.InternalError(c, "Failed to create school account")
		return
	}

	var school models.School
	database.DB.Preload("User").Preload("Curricula").First(&school, schoolIDFilter, schoolID)
	response.Created(c, school, "School account created successfully")
}

func GetSchool(c *gin.Context) {
	id := c.Param("id")
	var school models.School
	if err := database.DB.Preload("User").Preload("Curricula").First(&school, schoolIDFilter, id).Error; err != nil {
		response.NotFound(c, msgSchoolNotFound)
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
	if err := database.DB.Preload("Curricula").First(&school, schoolIDFilter, id).Error; err != nil {
		response.NotFound(c, msgSchoolNotFound)
		return
	}

	updates := map[string]interface{}{
		"name":           req.Name,
		"address":        req.Address,
		"contact_number": req.ContactNumber,
		"contact_person": req.ContactPerson,
	}
	if req.MaxTeachers != nil {
		updates["max_teachers"] = *req.MaxTeachers
	}
	if req.MaxCustomBooks != nil {
		updates["max_custom_books"] = *req.MaxCustomBooks
	}
	database.DB.Model(&school).Updates(updates)

	if req.CurriculaIDs != nil {
		if len(school.Curricula) > 0 {
			response.UnprocessableEntity(c, "Curriculum cannot be changed after initial assignment")
			return
		}
		var curricula []models.Curriculum
		database.DB.Where("id IN ?", req.CurriculaIDs).Find(&curricula)
		database.DB.Model(&school).Association("Curricula").Replace(curricula)
	}

	database.DB.Preload("User").Preload("Curricula").First(&school, schoolIDFilter, id)
	response.Success(c, school, "School updated successfully")
}

func DeleteSchool(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.First(&models.School{}, schoolIDFilter, id).Error; err != nil {
		response.NotFound(c, msgSchoolNotFound)
		return
	}

	if err := database.DB.Exec("CALL sp_school_set_status(?, ?)", id, "soft_delete").Error; err != nil {
		response.InternalError(c, "Failed to delete school")
		return
	}

	response.Success(c, nil, "School deleted successfully")
}

func HardDeleteSchool(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Unscoped().First(&models.School{}, schoolIDFilter, id).Error; err != nil {
		response.NotFound(c, msgSchoolNotFound)
		return
	}

	if err := database.DB.Exec("CALL sp_school_hard_delete(?)", id).Error; err != nil {
		response.InternalError(c, "Failed to permanently delete school")
		return
	}

	response.Success(c, nil, "School permanently deleted")
}

func ToggleSchoolStatus(c *gin.Context) {
	id := c.Param("id")
	var school models.School
	if err := database.DB.Preload("User").First(&school, schoolIDFilter, id).Error; err != nil {
		response.NotFound(c, msgSchoolNotFound)
		return
	}

	action := "activate"
	msg := "School activated"
	if school.User.IsActive {
		action = "deactivate"
		msg = "School deactivated"
	}

	if err := database.DB.Exec("CALL sp_school_set_status(?, ?)", id, action).Error; err != nil {
		response.InternalError(c, "Failed to update school status")
		return
	}

	response.Success(c, nil, msg)
}
