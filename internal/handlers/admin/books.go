package admin

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/Umairnoor2398/examify-backend/internal/config"
	"github.com/Umairnoor2398/examify-backend/internal/database"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/Umairnoor2398/examify-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type createBookRequest struct {
	Title        string   `json:"title" binding:"required"`
	Description  string   `json:"description"`
	URL          string   `json:"url"`
	Version      string   `json:"version"`
	SubjectIDs   []string `json:"subject_ids"`
	ClassIDs     []string `json:"class_ids"`
	CurriculaIDs []string `json:"curricula_ids"`
}

type createChapterRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	OrderIndex  int    `json:"order_index"`
}

func ListBooks(c *gin.Context) {
	pg := utils.GetPagination(c)
	search := c.Query("search")

	var books []models.Book
	var total int64

	q := database.DB.Model(&models.Book{}).
		Where("school_id IS NULL").
		Preload("Subjects").
		Preload("Classes").
		Preload("Curricula")

	if search != "" {
		q = q.Where("title LIKE ?", "%"+search+"%")
	}

	q.Count(&total)
	q.Limit(pg.PerPage).Offset(pg.Offset).Find(&books)

	response.Paginated(c, books, response.PaginationMeta{
		Total: total, Page: pg.Page, PerPage: pg.PerPage,
		TotalPages: utils.TotalPages(total, pg.PerPage),
	})
}

func CreateBook(c *gin.Context) {
	var req createBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := c.GetString("user_id")
	book := models.Book{
		Base:        models.Base{ID: utils.NewUUID()},
		Title:       req.Title,
		Description: req.Description,
		URL:         req.URL,
		Version:     req.Version,
		CreatedBy:   userID,
	}

	if err := database.DB.Create(&book).Error; err != nil {
		response.InternalError(c, "Failed to create book")
		return
	}

	associateBookRelations(&book, req.SubjectIDs, req.ClassIDs, req.CurriculaIDs)
	database.DB.Preload("Subjects").Preload("Classes").Preload("Curricula").First(&book, "id = ?", book.ID)
	response.Created(c, book, "Book created successfully")
}

func GetBook(c *gin.Context) {
	id := c.Param("id")
	var book models.Book
	if err := database.DB.
		Preload("Subjects").
		Preload("Classes").
		Preload("Curricula").
		Preload("Chapters").
		First(&book, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Book not found")
		return
	}
	response.Success(c, book, "")
}

func UpdateBook(c *gin.Context) {
	id := c.Param("id")
	var req createBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var book models.Book
	if err := database.DB.First(&book, "id = ? AND school_id IS NULL", id).Error; err != nil {
		response.NotFound(c, "Book not found")
		return
	}

	database.DB.Model(&book).Updates(map[string]interface{}{
		"title": req.Title, "description": req.Description,
		"url": req.URL, "version": req.Version,
	})

	if req.SubjectIDs != nil {
		associateBookRelations(&book, req.SubjectIDs, req.ClassIDs, req.CurriculaIDs)
	}

	database.DB.Preload("Subjects").Preload("Classes").Preload("Curricula").First(&book, "id = ?", id)
	response.Success(c, book, "Book updated")
}

func DeleteBook(c *gin.Context) {
	id := c.Param("id")
	var book models.Book
	if err := database.DB.First(&book, "id = ? AND school_id IS NULL", id).Error; err != nil {
		response.NotFound(c, "Book not found")
		return
	}
	database.DB.Delete(&book)
	response.Success(c, nil, "Book deleted")
}

func UploadBookPDF(c *gin.Context, cfg *config.Config) {
	id := c.Param("id")
	var book models.Book
	if err := database.DB.First(&book, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Book not found")
		return
	}

	file, header, err := c.Request.FormFile("pdf")
	if err != nil {
		response.BadRequest(c, "PDF file required")
		return
	}
	defer file.Close()

	if err := utils.ValidateFileMIME(file, utils.AllowedDocMIMEs); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	pdfURL, err := utils.SaveUploadedFile(file, header, cfg.Upload.Dir, "books", cfg.Upload.MaxSize)
	if err != nil {
		response.InternalError(c, "Failed to save file")
		return
	}

	database.DB.Model(&book).Update("pdf_url", pdfURL)
	response.Success(c, gin.H{"pdf_url": pdfURL}, "PDF uploaded")
}

func ImportBooksCSV(c *gin.Context, cfg *config.Config) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "CSV file required")
		return
	}
	defer file.Close()

	userID := c.GetString("user_id")
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		response.BadRequest(c, "Invalid CSV format. Expected headers: title,description,version")
		return
	}

	created := 0
	for _, row := range records[1:] {
		if len(row) < 1 || strings.TrimSpace(row[0]) == "" {
			continue
		}
		book := models.Book{
			Base:      models.Base{ID: utils.NewUUID()},
			Title:     strings.TrimSpace(row[0]),
			CreatedBy: userID,
		}
		if len(row) > 1 {
			book.Description = row[1]
		}
		if len(row) > 2 {
			book.Version = row[2]
		}
		if database.DB.Create(&book).Error == nil {
			created++
		}
	}

	response.Success(c, gin.H{"created": created}, fmt.Sprintf("%d books imported", created))
}

// Chapters
func ListChapters(c *gin.Context) {
	bookID := c.Param("id")
	var chapters []models.Chapter
	database.DB.Where("book_id = ?", bookID).Order("order_index asc").Find(&chapters)
	response.Success(c, chapters, "")
}

func CreateChapter(c *gin.Context) {
	bookID := c.Param("id")
	var req createChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var book models.Book
	if err := database.DB.First(&book, "id = ?", bookID).Error; err != nil {
		response.NotFound(c, "Book not found")
		return
	}

	chapter := models.Chapter{
		Base:        models.Base{ID: utils.NewUUID()},
		BookID:      bookID,
		Name:        req.Name,
		Description: req.Description,
		OrderIndex:  req.OrderIndex,
	}

	if err := database.DB.Create(&chapter).Error; err != nil {
		response.InternalError(c, "Failed to create chapter")
		return
	}
	response.Created(c, chapter, "Chapter created")
}

func GetChapter(c *gin.Context) {
	id := c.Param("chapterId")
	var chapter models.Chapter
	if err := database.DB.First(&chapter, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Chapter not found")
		return
	}
	response.Success(c, chapter, "")
}

func UpdateChapter(c *gin.Context) {
	id := c.Param("chapterId")
	var req createChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var chapter models.Chapter
	if err := database.DB.First(&chapter, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Chapter not found")
		return
	}

	database.DB.Model(&chapter).Updates(map[string]interface{}{
		"name": req.Name, "description": req.Description, "order_index": req.OrderIndex,
	})
	response.Success(c, chapter, "Chapter updated")
}

func DeleteChapter(c *gin.Context) {
	id := c.Param("chapterId")
	var chapter models.Chapter
	if err := database.DB.First(&chapter, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Chapter not found")
		return
	}
	database.DB.Delete(&chapter)
	response.Success(c, nil, "Chapter deleted")
}

func ImportChaptersCSV(c *gin.Context) {
	bookID := c.Param("id")
	var book models.Book
	if err := database.DB.First(&book, "id = ?", bookID).Error; err != nil {
		response.NotFound(c, "Book not found")
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
		response.BadRequest(c, "Invalid CSV format. Expected headers: name,description")
		return
	}

	created := 0
	for i, row := range records[1:] {
		if len(row) < 1 || strings.TrimSpace(row[0]) == "" {
			continue
		}
		desc := ""
		if len(row) > 1 {
			desc = row[1]
		}
		chapter := models.Chapter{
			Base:        models.Base{ID: utils.NewUUID()},
			BookID:      bookID,
			Name:        strings.TrimSpace(row[0]),
			Description: desc,
			OrderIndex:  i + 1,
		}
		if database.DB.Create(&chapter).Error == nil {
			created++
		}
	}

	response.Success(c, gin.H{"created": created}, fmt.Sprintf("%d chapters imported", created))
}

func associateBookRelations(book *models.Book, subjectIDs, classIDs, curriculaIDs []string) {
	if len(subjectIDs) > 0 {
		var subjects []models.Subject
		database.DB.Where("id IN ?", subjectIDs).Find(&subjects)
		database.DB.Model(book).Association("Subjects").Replace(subjects)
	}
	if len(classIDs) > 0 {
		var classes []models.Class
		database.DB.Where("id IN ?", classIDs).Find(&classes)
		database.DB.Model(book).Association("Classes").Replace(classes)
	}
	if len(curriculaIDs) > 0 {
		var curricula []models.Curriculum
		database.DB.Where("id IN ?", curriculaIDs).Find(&curricula)
		database.DB.Model(book).Association("Curricula").Replace(curricula)
	}
}
