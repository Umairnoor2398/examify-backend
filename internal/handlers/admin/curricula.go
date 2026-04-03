package admin

import (
	"github.com/Umairnoor2398/examify-backend/internal/database"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/Umairnoor2398/examify-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type nameRequest struct {
	Name string `json:"name" binding:"required,min=1"`
}

// Curricula
func ListCurricula(c *gin.Context) {
	var curricula []models.Curriculum
	database.DB.Order("name asc").Find(&curricula)
	response.Success(c, curricula, "")
}

func CreateCurriculum(c *gin.Context) {
	var req nameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	curriculum := models.Curriculum{Base: models.Base{ID: utils.NewUUID()}, Name: req.Name}
	if err := database.DB.Create(&curriculum).Error; err != nil {
		response.Conflict(c, "Curriculum with this name already exists")
		return
	}
	response.Created(c, curriculum, "Curriculum created")
}

func UpdateCurriculum(c *gin.Context) {
	id := c.Param("id")
	var req nameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var curriculum models.Curriculum
	if err := database.DB.First(&curriculum, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Curriculum not found")
		return
	}

	database.DB.Model(&curriculum).Update("name", req.Name)
	response.Success(c, curriculum, "Curriculum updated")
}

func DeleteCurriculum(c *gin.Context) {
	id := c.Param("id")
	var curriculum models.Curriculum
	if err := database.DB.First(&curriculum, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Curriculum not found")
		return
	}
	database.DB.Delete(&curriculum)
	response.Success(c, nil, "Curriculum deleted")
}

// Subjects
func ListSubjects(c *gin.Context) {
	var subjects []models.Subject
	database.DB.Order("name asc").Find(&subjects)
	response.Success(c, subjects, "")
}

func CreateSubject(c *gin.Context) {
	var req nameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	subject := models.Subject{Base: models.Base{ID: utils.NewUUID()}, Name: req.Name}
	if err := database.DB.Create(&subject).Error; err != nil {
		response.Conflict(c, "Subject with this name already exists")
		return
	}
	response.Created(c, subject, "Subject created")
}

func UpdateSubject(c *gin.Context) {
	id := c.Param("id")
	var req nameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var subject models.Subject
	if err := database.DB.First(&subject, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Subject not found")
		return
	}

	database.DB.Model(&subject).Update("name", req.Name)
	response.Success(c, subject, "Subject updated")
}

func DeleteSubject(c *gin.Context) {
	id := c.Param("id")
	var subject models.Subject
	if err := database.DB.First(&subject, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Subject not found")
		return
	}
	database.DB.Delete(&subject)
	response.Success(c, nil, "Subject deleted")
}

// Classes
func ListClasses(c *gin.Context) {
	var classes []models.Class
	database.DB.Order("name asc").Find(&classes)
	response.Success(c, classes, "")
}

func CreateClass(c *gin.Context) {
	var req nameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	class := models.Class{Base: models.Base{ID: utils.NewUUID()}, Name: req.Name}
	if err := database.DB.Create(&class).Error; err != nil {
		response.Conflict(c, "Class with this name already exists")
		return
	}
	response.Created(c, class, "Class created")
}

func UpdateClass(c *gin.Context) {
	id := c.Param("id")
	var req nameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var class models.Class
	if err := database.DB.First(&class, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Class not found")
		return
	}

	database.DB.Model(&class).Update("name", req.Name)
	response.Success(c, class, "Class updated")
}

func DeleteClass(c *gin.Context) {
	id := c.Param("id")
	var class models.Class
	if err := database.DB.First(&class, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Class not found")
		return
	}
	database.DB.Delete(&class)
	response.Success(c, nil, "Class deleted")
}
