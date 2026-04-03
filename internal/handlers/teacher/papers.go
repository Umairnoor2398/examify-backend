package teacher

import (
	"fmt"
	"math/rand"

	"github.com/Umairnoor2398/examify-backend/internal/database"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/Umairnoor2398/examify-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type paperConfigInput struct {
	MCQCount         int     `json:"mcq_count"`
	ShortCount       int     `json:"short_count"`
	DescriptiveCount int     `json:"descriptive_count"`
	MCQMarks         float64 `json:"mcq_marks"`
	ShortMarks       float64 `json:"short_marks"`
	DescriptiveMarks float64 `json:"descriptive_marks"`
	Difficulty       string  `json:"difficulty"`
	Tags             []string `json:"tags"`
}

type createPaperRequest struct {
	Name        string           `json:"name" binding:"required"`
	BookID      string           `json:"book_id" binding:"required"`
	ChapterIDs  []string         `json:"chapter_ids" binding:"required,min=1"`
	Config      paperConfigInput `json:"config" binding:"required"`
	QuestionIDs []string         `json:"question_ids"`
}

func getSchoolForTeacher(userID string) (*models.School, error) {
	var school models.School
	err := database.DB.Raw(`
		SELECT s.* FROM schools s
		JOIN teacher_schools ts ON ts.school_id = s.id
		WHERE ts.user_id = ?
	`, userID).Scan(&school).Error
	if err != nil || school.ID == "" {
		return nil, fmt.Errorf("school not found")
	}
	return &school, nil
}

func ListBooks(c *gin.Context) {
	userID := c.GetString("user_id")
	school, err := getSchoolForTeacher(userID)
	if err != nil {
		response.NotFound(c, "School not found for this teacher")
		return
	}

	var curriculaIDs []string
	database.DB.Raw("SELECT curriculum_id FROM school_curricula WHERE school_id = ?", school.ID).
		Scan(&curriculaIDs)

	var books []models.Book
	if len(curriculaIDs) > 0 {
		database.DB.Raw(`
			SELECT DISTINCT b.* FROM books b
			LEFT JOIN book_curricula bc ON bc.book_id = b.id
			WHERE (b.school_id IS NULL AND bc.curriculum_id IN ?) OR b.school_id = ?
		`, curriculaIDs, school.ID).
			Preload("Subjects").Preload("Classes").Preload("Curricula").
			Scan(&books)
	} else {
		database.DB.Where("school_id = ?", school.ID).
			Preload("Subjects").Preload("Classes").
			Find(&books)
	}

	response.Success(c, books, "")
}

func ListBookChapters(c *gin.Context) {
	bookID := c.Param("bookId")
	var chapters []models.Chapter
	database.DB.Where("book_id = ?", bookID).Order("order_index asc").Find(&chapters)
	response.Success(c, chapters, "")
}

func GetFilteredQuestions(c *gin.Context) {
	chapterIDs := c.QueryArray("chapter_ids")
	difficulty := c.Query("difficulty")
	tags := c.QueryArray("tags")
	pg := utils.GetPagination(c)

	var questions []models.Question
	var total int64

	q := database.DB.Model(&models.Question{}).
		Where("chapter_id IN ?", chapterIDs).
		Preload("Options").
		Preload("AnswerKey").
		Preload("Tags")

	if difficulty != "" && difficulty != "random" {
		q = q.Where("difficulty = ?", difficulty)
	}
	if len(tags) > 0 {
		q = q.Joins("JOIN question_tags qt ON qt.question_id = questions.id").
			Where("qt.tag IN ?", tags)
	}

	q.Count(&total)
	q.Limit(pg.PerPage).Offset(pg.Offset).Find(&questions)

	response.Paginated(c, questions, response.PaginationMeta{
		Total: total, Page: pg.Page, PerPage: pg.PerPage,
		TotalPages: utils.TotalPages(total, pg.PerPage),
	})
}

func ListPapers(c *gin.Context) {
	userID := c.GetString("user_id")
	pg := utils.GetPagination(c)
	status := c.Query("status")

	var papers []models.Paper
	var total int64

	q := database.DB.Model(&models.Paper{}).
		Where("created_by = ?", userID).
		Preload("Book").
		Preload("Config")

	if status != "" {
		q = q.Where("status = ?", status)
	}

	q.Count(&total)
	q.Limit(pg.PerPage).Offset(pg.Offset).Order("created_at desc").Find(&papers)

	response.Paginated(c, papers, response.PaginationMeta{
		Total: total, Page: pg.Page, PerPage: pg.PerPage,
		TotalPages: utils.TotalPages(total, pg.PerPage),
	})
}

func CreatePaper(c *gin.Context) {
	userID := c.GetString("user_id")
	school, err := getSchoolForTeacher(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	var req createPaperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	paper := models.Paper{
		Base:      models.Base{ID: utils.NewUUID()},
		Name:      req.Name,
		SchoolID:  school.ID,
		CreatedBy: userID,
		Status:    models.PaperDraft,
		BookID:    req.BookID,
	}

	if err := database.DB.Create(&paper).Error; err != nil {
		response.InternalError(c, "Failed to create paper")
		return
	}

	// Associate chapters
	var chapters []models.Chapter
	database.DB.Where("id IN ?", req.ChapterIDs).Find(&chapters)
	database.DB.Model(&paper).Association("Chapters").Replace(chapters)

	// Save config
	cfg := models.PaperConfig{
		ID:               utils.NewUUID(),
		PaperID:          paper.ID,
		MCQCount:         req.Config.MCQCount,
		ShortCount:       req.Config.ShortCount,
		DescriptiveCount: req.Config.DescriptiveCount,
		MCQMarks:         req.Config.MCQMarks,
		ShortMarks:       req.Config.ShortMarks,
		DescriptiveMarks: req.Config.DescriptiveMarks,
		Difficulty:       req.Config.Difficulty,
	}
	if cfg.MCQMarks == 0 {
		cfg.MCQMarks = 1
	}
	if cfg.ShortMarks == 0 {
		cfg.ShortMarks = 2
	}
	if cfg.DescriptiveMarks == 0 {
		cfg.DescriptiveMarks = 5
	}
	database.DB.Create(&cfg)

	// Add questions
	addQuestionsToPublisher(&paper, req, userID)

	loadFullPaper(c, paper.ID)
}

func GetPaper(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	var paper models.Paper
	if err := database.DB.Where("id = ? AND created_by = ?", id, userID).First(&paper).Error; err != nil {
		response.NotFound(c, "Paper not found")
		return
	}
	loadFullPaper(c, id)
}

func UpdatePaper(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	var paper models.Paper
	if err := database.DB.Where("id = ? AND created_by = ? AND status = ?", id, userID, models.PaperDraft).First(&paper).Error; err != nil {
		response.NotFound(c, "Draft paper not found")
		return
	}

	var req createPaperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	database.DB.Model(&paper).Updates(map[string]interface{}{
		"name": req.Name, "book_id": req.BookID,
	})

	var chapters []models.Chapter
	database.DB.Where("id IN ?", req.ChapterIDs).Find(&chapters)
	database.DB.Model(&paper).Association("Chapters").Replace(chapters)

	database.DB.Model(&models.PaperConfig{}).Where("paper_id = ?", id).Updates(map[string]interface{}{
		"mcq_count": req.Config.MCQCount, "short_count": req.Config.ShortCount,
		"descriptive_count": req.Config.DescriptiveCount,
		"mcq_marks": req.Config.MCQMarks, "short_marks": req.Config.ShortMarks,
		"descriptive_marks": req.Config.DescriptiveMarks, "difficulty": req.Config.Difficulty,
	})

	database.DB.Where("paper_id = ?", id).Delete(&models.PaperQuestion{})
	addQuestionsToPublisher(&paper, req, userID)
	loadFullPaper(c, id)
}

func DeletePaper(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	var paper models.Paper
	if err := database.DB.Where("id = ? AND created_by = ? AND status = ?", id, userID, models.PaperDraft).First(&paper).Error; err != nil {
		response.NotFound(c, "Draft paper not found")
		return
	}

	database.DB.Where("paper_id = ?", id).Delete(&models.PaperQuestion{})
	database.DB.Where("paper_id = ?", id).Delete(&models.PaperConfig{})
	database.DB.Model(&paper).Association("Chapters").Clear()
	database.DB.Delete(&paper)

	response.Success(c, nil, "Paper deleted")
}

func SubmitPaper(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	var paper models.Paper
	if err := database.DB.Where("id = ? AND created_by = ? AND status = ?", id, userID, models.PaperDraft).First(&paper).Error; err != nil {
		response.NotFound(c, "Draft paper not found")
		return
	}

	database.DB.Model(&paper).Update("status", models.PaperSubmitted)
	response.Success(c, nil, "Paper submitted successfully")
}

func addQuestionsToPublisher(paper *models.Paper, req createPaperRequest, userID string) {
	manualIDs := map[string]bool{}
	orderIdx := 0

	// Add manually selected questions first
	for _, qid := range req.QuestionIDs {
		var q models.Question
		if database.DB.First(&q, "id = ?", qid).Error != nil {
			continue
		}
		marks := getMarks(q.Type, req.Config)
		database.DB.Create(&models.PaperQuestion{
			ID: utils.NewUUID(), PaperID: paper.ID, QuestionID: qid,
			Marks: marks, OrderIndex: orderIdx,
		})
		manualIDs[qid] = true
		orderIdx++
	}

	// Auto-fill remaining with random selection per type
	fillRandom(paper.ID, req.ChapterIDs, models.QuestionMCQ, req.Config.MCQCount, req.Config.MCQMarks, req.Config.Difficulty, req.Config.Tags, manualIDs, &orderIdx)
	fillRandom(paper.ID, req.ChapterIDs, models.QuestionShort, req.Config.ShortCount, req.Config.ShortMarks, req.Config.Difficulty, req.Config.Tags, manualIDs, &orderIdx)
	fillRandom(paper.ID, req.ChapterIDs, models.QuestionDescriptive, req.Config.DescriptiveCount, req.Config.DescriptiveMarks, req.Config.Difficulty, req.Config.Tags, manualIDs, &orderIdx)
}

func fillRandom(paperID string, chapterIDs []string, qType models.QuestionType, needed int, marks float64, difficulty string, tags []string, exclude map[string]bool, orderIdx *int) {
	// Count how many of this type are already manually selected
	alreadyHave := 0
	for id := range exclude {
		var q models.Question
		if database.DB.First(&q, "id = ? AND type = ?", id, qType).Error == nil {
			alreadyHave++
		}
	}
	remaining := needed - alreadyHave
	if remaining <= 0 {
		return
	}

	q := database.DB.Where("chapter_id IN ? AND type = ?", chapterIDs, qType)
	if difficulty != "" && difficulty != "random" {
		q = q.Where("difficulty = ?", difficulty)
	}
	if len(tags) > 0 {
		q = q.Joins("JOIN question_tags qt ON qt.question_id = questions.id").
			Where("qt.tag IN ?", tags)
	}

	var questions []models.Question
	q.Find(&questions)

	// Exclude already selected
	var available []models.Question
	for _, quest := range questions {
		if !exclude[quest.ID] {
			available = append(available, quest)
		}
	}

	// Shuffle
	rand.Shuffle(len(available), func(i, j int) { available[i], available[j] = available[j], available[i] })

	count := remaining
	if count > len(available) {
		count = len(available)
	}

	for i := 0; i < count; i++ {
		if marks == 0 {
			marks = 1
		}
		database.DB.Create(&models.PaperQuestion{
			ID: utils.NewUUID(), PaperID: paperID, QuestionID: available[i].ID,
			Marks: marks, OrderIndex: *orderIdx,
		})
		*orderIdx++
	}
}

func getMarks(qType models.QuestionType, cfg paperConfigInput) float64 {
	switch qType {
	case models.QuestionMCQ:
		if cfg.MCQMarks > 0 {
			return cfg.MCQMarks
		}
		return 1
	case models.QuestionShort:
		if cfg.ShortMarks > 0 {
			return cfg.ShortMarks
		}
		return 2
	case models.QuestionDescriptive:
		if cfg.DescriptiveMarks > 0 {
			return cfg.DescriptiveMarks
		}
		return 5
	}
	return 1
}

func loadFullPaper(c *gin.Context, paperID string) {
	var paper models.Paper
	database.DB.
		Preload("Book").
		Preload("Chapters").
		Preload("Questions.Question.Options").
		Preload("Questions.Question.AnswerKey").
		Preload("Questions.Question.Tags").
		Preload("Config").
		Preload("Creator").
		First(&paper, "id = ?", paperID)
	response.Success(c, paper, "")
}
