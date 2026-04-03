package school

import (
	"github.com/Umairnoor2398/examify-backend/internal/database"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/Umairnoor2398/examify-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func ListPapers(c *gin.Context) {
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	pg := utils.GetPagination(c)
	var papers []models.Paper
	var total int64

	q := database.DB.Model(&models.Paper{}).
		Where("school_id = ? AND status = ?", school.ID, models.PaperSubmitted).
		Preload("Creator").
		Preload("Book").
		Preload("Chapters").
		Preload("Config")

	q.Count(&total)
	q.Limit(pg.PerPage).Offset(pg.Offset).
		Order("created_at desc").
		Find(&papers)

	response.Paginated(c, papers, response.PaginationMeta{
		Total: total, Page: pg.Page, PerPage: pg.PerPage,
		TotalPages: utils.TotalPages(total, pg.PerPage),
	})
}

func GetPaper(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	var paper models.Paper
	if err := database.DB.
		Where("id = ? AND school_id = ? AND status = ?", id, school.ID, models.PaperSubmitted).
		Preload("Creator").
		Preload("Book").
		Preload("Chapters").
		Preload("Questions.Question.Options").
		Preload("Questions.Question.AnswerKey").
		Preload("Questions.Question.Tags").
		Preload("Config").
		First(&paper).Error; err != nil {
		response.NotFound(c, "Paper not found")
		return
	}

	response.Success(c, paper, "")
}

func DeletePaper(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	var paper models.Paper
	if err := database.DB.
		Where("id = ? AND school_id = ? AND status = ?", id, school.ID, models.PaperSubmitted).
		First(&paper).Error; err != nil {
		response.NotFound(c, "Paper not found")
		return
	}

	database.DB.Where("paper_id = ?", id).Delete(&models.PaperQuestion{})
	database.DB.Where("paper_id = ?", id).Delete(&models.PaperConfig{})
	database.DB.Model(&paper).Association("Chapters").Clear()
	database.DB.Delete(&paper)

	response.Success(c, nil, "Paper deleted")
}
