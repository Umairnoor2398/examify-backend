package school

import (
	"strings"

	"github.com/Umairnoor2398/examify-backend/internal/database"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/Umairnoor2398/examify-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createTeacherRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"`
}

type disableTeacherRequest struct {
	DeletePapers bool `json:"delete_papers"`
}

var allowedTeacherSortColumns = map[string]string{
	"username":    "u.username",
	"email":       "u.email",
	"is_active":   "u.is_active",
	"paper_count": "paper_count",
	"created_at":  "u.created_at",
}

func ListTeachers(c *gin.Context) {
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	pg := utils.GetPagination(c)
	sortBy := c.DefaultQuery("sort_by", "")
	sortOrder := strings.ToUpper(c.DefaultQuery("sort_order", "ASC"))
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}

	orderClause := "u.created_at ASC"
	if col, ok := allowedTeacherSortColumns[sortBy]; ok {
		orderClause = col + " " + sortOrder
	}

	type teacherRow struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		Email      string `json:"email"`
		IsActive   bool   `json:"is_active"`
		AvatarURL  string `json:"avatar_url"`
		PaperCount int64  `json:"paper_count"`
	}

	var total int64
	database.DB.Raw(`
		SELECT COUNT(*)
		FROM users u
		JOIN teacher_schools ts ON ts.user_id = u.id
		WHERE ts.school_id = ? AND u.deleted_at IS NULL
	`, school.ID).Scan(&total)

	var teachers []teacherRow
	database.DB.Raw(`
		SELECT u.id, u.username, u.email, u.is_active, u.avatar_url,
		       COUNT(p.id) AS paper_count
		FROM users u
		JOIN teacher_schools ts ON ts.user_id = u.id
		LEFT JOIN papers p ON p.created_by = u.id AND p.deleted_at IS NULL
		WHERE ts.school_id = ? AND u.deleted_at IS NULL
		GROUP BY u.id, u.username, u.email, u.is_active, u.avatar_url, u.created_at
		ORDER BY `+orderClause+`
		LIMIT ? OFFSET ?
	`, school.ID, pg.PerPage, pg.Offset).Scan(&teachers)

	response.Paginated(c, teachers, response.PaginationMeta{
		Total:      total,
		Page:       pg.Page,
		PerPage:    pg.PerPage,
		TotalPages: utils.TotalPages(total, pg.PerPage),
	})
}

func CreateTeacher(c *gin.Context) {
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	// Count current teachers
	var teacherCount int64
	database.DB.Raw(`
		SELECT COUNT(*) FROM users u
		JOIN teacher_schools ts ON ts.user_id = u.id
		WHERE ts.school_id = ? AND u.deleted_at IS NULL
	`, school.ID).Scan(&teacherCount)

	if int(teacherCount) >= school.MaxTeachers {
		response.UnprocessableEntity(c, "Teacher limit reached. Disable an existing teacher or contact admin to increase the limit.")
		return
	}

	var req createTeacherRequest
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

	teacherUserID := utils.NewUUID()
	teacher := models.User{
		Base:         models.Base{ID: teacherUserID},
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         models.RoleTeacher,
		IsActive:     true,
	}

	if err := database.DB.Create(&teacher).Error; err != nil {
		response.InternalError(c, "Failed to create teacher account")
		return
	}

	// Link teacher to school
	database.DB.Exec("INSERT INTO teacher_schools (user_id, school_id) VALUES (?, ?)", teacherUserID, school.ID)

	response.Created(c, gin.H{
		"id": teacher.ID, "username": teacher.Username, "email": teacher.Email, "is_active": teacher.IsActive,
	}, "Teacher account created")
}

func ToggleTeacher(c *gin.Context) {
	teacherID := c.Param("id")
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	// Verify teacher belongs to this school
	var count int64
	database.DB.Raw("SELECT COUNT(*) FROM teacher_schools WHERE user_id = ? AND school_id = ?", teacherID, school.ID).Scan(&count)
	if count == 0 {
		response.Forbidden(c, "Teacher does not belong to your school")
		return
	}

	var teacher models.User
	if err := database.DB.First(&teacher, "id = ?", teacherID).Error; err != nil {
		response.NotFound(c, "Teacher not found")
		return
	}

	var req disableTeacherRequest
	c.ShouldBindJSON(&req)

	newStatus := !teacher.IsActive

	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		if !newStatus && req.DeletePapers {
			if err := tx.Where("created_by = ?", teacherID).Delete(&models.Paper{}).Error; err != nil {
				return err
			}
		} else if !newStatus && !req.DeletePapers {
			if err := tx.Model(&models.Paper{}).Where("created_by = ?", teacherID).Update("created_by", userID).Error; err != nil {
				return err
			}
		}
		return tx.Model(&teacher).Update("is_active", newStatus).Error
	})
	if txErr != nil {
		response.InternalError(c, "Failed to update teacher status")
		return
	}

	msg := "Teacher account enabled"
	if !newStatus {
		msg = "Teacher account disabled"
	}
	response.Success(c, nil, msg)
}

func DeleteTeacher(c *gin.Context) {
	teacherID := c.Param("id")
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	var count int64
	database.DB.Raw("SELECT COUNT(*) FROM teacher_schools WHERE user_id = ? AND school_id = ?", teacherID, school.ID).Scan(&count)
	if count == 0 {
		response.Forbidden(c, "Teacher does not belong to your school")
		return
	}

	var req disableTeacherRequest
	c.ShouldBindJSON(&req)

	if req.DeletePapers {
		database.DB.Where("created_by = ?", teacherID).Delete(&models.Paper{})
	} else {
		database.DB.Model(&models.Paper{}).Where("created_by = ?", teacherID).Update("created_by", userID)
	}

	database.DB.Exec("DELETE FROM teacher_schools WHERE user_id = ? AND school_id = ?", teacherID, school.ID)
	database.DB.Delete(&models.User{}, "id = ?", teacherID)

	response.Success(c, nil, "Teacher deleted")
}
