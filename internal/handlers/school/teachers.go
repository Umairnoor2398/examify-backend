package school

import (
	"github.com/Umairnoor2398/examify-backend/internal/database"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/Umairnoor2398/examify-backend/pkg/response"
	"github.com/gin-gonic/gin"
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

func ListTeachers(c *gin.Context) {
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	type teacherWithPaperCount struct {
		models.User
		PaperCount int64 `json:"paper_count"`
	}

	var teachers []models.User
	database.DB.Raw(`
		SELECT u.* FROM users u
		JOIN teacher_schools ts ON ts.user_id = u.id
		WHERE ts.school_id = ? AND u.deleted_at IS NULL
	`, school.ID).Scan(&teachers)

	result := make([]gin.H, len(teachers))
	for i, t := range teachers {
		var count int64
		database.DB.Model(&models.Paper{}).Where("created_by = ?", t.ID).Count(&count)
		result[i] = gin.H{
			"id": t.ID, "username": t.Username, "email": t.Email,
			"is_active": t.IsActive, "paper_count": count,
		}
	}

	response.Success(c, result, "")
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

	if !newStatus && req.DeletePapers {
		// Delete all papers by this teacher
		database.DB.Where("created_by = ?", teacherID).Delete(&models.Paper{})
	} else if !newStatus && !req.DeletePapers {
		// Re-link papers to school admin
		database.DB.Model(&models.Paper{}).
			Where("created_by = ?", teacherID).
			Update("created_by", userID)
	}

	database.DB.Model(&teacher).Update("is_active", newStatus)

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
