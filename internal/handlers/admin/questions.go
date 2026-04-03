package admin

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/Umairnoor2398/examify-backend/internal/database"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/Umairnoor2398/examify-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type questionOption struct {
	Text      string `json:"text" binding:"required"`
	IsCorrect bool   `json:"is_correct"`
}

type createQuestionRequest struct {
	Type       string           `json:"type" binding:"required,oneof=mcq short descriptive"`
	Text       string           `json:"text" binding:"required"`
	Difficulty string           `json:"difficulty" binding:"required,oneof=easy medium hard"`
	Tags       []string         `json:"tags"`
	Options    []questionOption `json:"options"`
	AnswerKey  string           `json:"answer_key"`
}

func ListQuestions(c *gin.Context) {
	chapterID := c.Param("chapterId")
	qType := c.Query("type")
	difficulty := c.Query("difficulty")
	tag := c.Query("tag")
	pg := utils.GetPagination(c)

	var questions []models.Question
	var total int64

	q := database.DB.Model(&models.Question{}).
		Where("chapter_id = ?", chapterID).
		Preload("Options").
		Preload("AnswerKey").
		Preload("Tags")

	if qType != "" {
		q = q.Where("type = ?", qType)
	}
	if difficulty != "" {
		q = q.Where("difficulty = ?", difficulty)
	}
	if tag != "" {
		q = q.Joins("JOIN question_tags ON question_tags.question_id = questions.id").
			Where("question_tags.tag = ?", tag)
	}

	q.Count(&total)
	q.Limit(pg.PerPage).Offset(pg.Offset).Find(&questions)

	response.Paginated(c, questions, response.PaginationMeta{
		Total: total, Page: pg.Page, PerPage: pg.PerPage,
		TotalPages: utils.TotalPages(total, pg.PerPage),
	})
}

func CreateQuestion(c *gin.Context) {
	chapterID := c.Param("chapterId")
	var req createQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := validateQuestion(req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var chapter models.Chapter
	if err := database.DB.First(&chapter, "id = ?", chapterID).Error; err != nil {
		response.NotFound(c, "Chapter not found")
		return
	}

	question := models.Question{
		Base:       models.Base{ID: utils.NewUUID()},
		ChapterID:  chapterID,
		Type:       models.QuestionType(req.Type),
		Text:       req.Text,
		Difficulty: models.Difficulty(req.Difficulty),
	}

	if err := database.DB.Create(&question).Error; err != nil {
		response.InternalError(c, "Failed to create question")
		return
	}

	saveQuestionRelations(question.ID, req)
	loadQuestionRelations(c, question.ID)
}

func GetQuestion(c *gin.Context) {
	id := c.Param("id")
	var question models.Question
	if err := database.DB.Preload("Options").Preload("AnswerKey").Preload("Tags").
		First(&question, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Question not found")
		return
	}
	response.Success(c, question, "")
}

func UpdateQuestion(c *gin.Context) {
	id := c.Param("id")
	var req createQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var question models.Question
	if err := database.DB.First(&question, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Question not found")
		return
	}

	if err := validateQuestion(req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	database.DB.Model(&question).Updates(map[string]interface{}{
		"type": req.Type, "text": req.Text, "difficulty": req.Difficulty,
	})

	// Replace relations
	database.DB.Where("question_id = ?", id).Delete(&models.QuestionOption{})
	database.DB.Where("question_id = ?", id).Delete(&models.QuestionAnswerKey{})
	database.DB.Where("question_id = ?", id).Delete(&models.QuestionTag{})
	saveQuestionRelations(id, req)
	loadQuestionRelations(c, id)
}

func DeleteQuestion(c *gin.Context) {
	id := c.Param("id")
	var question models.Question
	if err := database.DB.First(&question, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Question not found")
		return
	}
	database.DB.Where("question_id = ?", id).Delete(&models.QuestionOption{})
	database.DB.Where("question_id = ?", id).Delete(&models.QuestionAnswerKey{})
	database.DB.Where("question_id = ?", id).Delete(&models.QuestionTag{})
	database.DB.Delete(&question)
	response.Success(c, nil, "Question deleted")
}

func ImportQuestionsCSV(c *gin.Context) {
	chapterID := c.Param("chapterId")
	var chapter models.Chapter
	if err := database.DB.First(&chapter, "id = ?", chapterID).Error; err != nil {
		response.NotFound(c, "Chapter not found")
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "CSV file required")
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		response.BadRequest(c, "Invalid CSV. Expected: type,text,difficulty,answer_key_or_options,tags")
		return
	}

	created := 0
	for _, row := range records[1:] {
		if len(row) < 3 {
			continue
		}
		qType := strings.TrimSpace(row[0])
		text := strings.TrimSpace(row[1])
		difficulty := strings.TrimSpace(row[2])

		if text == "" || (qType != "mcq" && qType != "short" && qType != "descriptive") {
			continue
		}

		question := models.Question{
			Base:       models.Base{ID: utils.NewUUID()},
			ChapterID:  chapterID,
			Type:       models.QuestionType(qType),
			Text:       text,
			Difficulty: models.Difficulty(difficulty),
		}

		if database.DB.Create(&question).Error != nil {
			continue
		}

		// Col 3: for MCQ = "opt1|opt2|opt3|opt4|correct_index", for short/descriptive = answer key
		if len(row) > 3 && strings.TrimSpace(row[3]) != "" {
			col := strings.TrimSpace(row[3])
			if qType == "mcq" {
				parts := strings.Split(col, "|")
				correctIdx := 0
				if len(parts) >= 5 {
					fmt.Sscanf(strings.TrimSpace(parts[4]), "%d", &correctIdx)
				}
				for i, p := range parts[:min(4, len(parts))] {
					database.DB.Create(&models.QuestionOption{
						ID: utils.NewUUID(), QuestionID: question.ID,
						OptionText: strings.TrimSpace(p), IsCorrect: i == correctIdx,
					})
				}
			} else {
				database.DB.Create(&models.QuestionAnswerKey{
					ID: utils.NewUUID(), QuestionID: question.ID, AnswerText: col,
				})
			}
		}

		// Col 4: tags (comma separated)
		if len(row) > 4 && strings.TrimSpace(row[4]) != "" {
			for _, t := range strings.Split(row[4], ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					database.DB.Create(&models.QuestionTag{
						ID: utils.NewUUID(), QuestionID: question.ID, Tag: t,
					})
				}
			}
		}
		created++
	}

	response.Success(c, gin.H{"created": created}, fmt.Sprintf("%d questions imported", created))
}

func validateQuestion(req createQuestionRequest) error {
	if req.Type == "mcq" {
		if len(req.Options) != 4 {
			return fmt.Errorf("MCQ questions must have exactly 4 options")
		}
		correctCount := 0
		for _, o := range req.Options {
			if o.IsCorrect {
				correctCount++
			}
		}
		if correctCount != 1 {
			return fmt.Errorf("MCQ questions must have exactly 1 correct option")
		}
	} else {
		if strings.TrimSpace(req.AnswerKey) == "" {
			return fmt.Errorf("answer_key is required for short/descriptive questions")
		}
	}
	return nil
}

func saveQuestionRelations(questionID string, req createQuestionRequest) {
	if req.Type == "mcq" {
		for _, o := range req.Options {
			database.DB.Create(&models.QuestionOption{
				ID: utils.NewUUID(), QuestionID: questionID,
				OptionText: o.Text, IsCorrect: o.IsCorrect,
			})
		}
	} else if req.AnswerKey != "" {
		database.DB.Create(&models.QuestionAnswerKey{
			ID: utils.NewUUID(), QuestionID: questionID, AnswerText: req.AnswerKey,
		})
	}
	for _, tag := range req.Tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			database.DB.Create(&models.QuestionTag{
				ID: utils.NewUUID(), QuestionID: questionID, Tag: tag,
			})
		}
	}
}

func loadQuestionRelations(c *gin.Context, questionID string) {
	var question models.Question
	database.DB.Preload("Options").Preload("AnswerKey").Preload("Tags").
		First(&question, "id = ?", questionID)
	response.Success(c, question, "")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
