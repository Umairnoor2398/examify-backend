package school

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

type createCustomBookRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Version     string   `json:"version"`
	SubjectIDs  []string `json:"subject_ids"`
	ClassIDs    []string `json:"class_ids"`
}

func ListCustomBooks(c *gin.Context) {
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	pg := utils.GetPagination(c)
	var books []models.Book
	var total int64

	q := database.DB.Model(&models.Book{}).
		Where("school_id = ?", school.ID).
		Preload("Subjects").
		Preload("Classes")

	q = q.Order("books.title ASC")
	q.Count(&total)
	q.Limit(pg.PerPage).Offset(pg.Offset).Find(&books)

	response.Paginated(c, books, response.PaginationMeta{
		Total: total, Page: pg.Page, PerPage: pg.PerPage,
		TotalPages: utils.TotalPages(total, pg.PerPage),
	})
}

func CreateCustomBook(c *gin.Context) {
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	// Check custom book limit
	var count int64
	database.DB.Model(&models.Book{}).Where("school_id = ?", school.ID).Count(&count)
	if int(count) >= school.MaxCustomBooks {
		response.UnprocessableEntity(c, "Custom book limit reached. Contact admin to increase the limit.")
		return
	}

	var req createCustomBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	book := models.Book{
		Base:        models.Base{ID: utils.NewUUID()},
		Title:       req.Title,
		Description: req.Description,
		URL:         req.URL,
		Version:     req.Version,
		SchoolID:    &school.ID,
		CreatedBy:   userID,
	}

	if err := database.DB.Create(&book).Error; err != nil {
		response.InternalError(c, "Failed to create book")
		return
	}

	if len(req.SubjectIDs) > 0 {
		var subjects []models.Subject
		database.DB.Where("id IN ?", req.SubjectIDs).Find(&subjects)
		database.DB.Model(&book).Association("Subjects").Replace(subjects)
	}
	if len(req.ClassIDs) > 0 {
		var classes []models.Class
		database.DB.Where("id IN ?", req.ClassIDs).Find(&classes)
		database.DB.Model(&book).Association("Classes").Replace(classes)
	}

	database.DB.Preload("Subjects").Preload("Classes").First(&book, "id = ?", book.ID)
	response.Created(c, book, "Custom book created")
}

func GetCustomBook(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	var book models.Book
	if err := database.DB.
		Preload("Subjects").Preload("Classes").Preload("Chapters").
		First(&book, "id = ? AND school_id = ?", id, school.ID).Error; err != nil {
		response.NotFound(c, "Book not found")
		return
	}
	response.Success(c, book, "")
}

func UpdateCustomBook(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	var book models.Book
	if err := database.DB.First(&book, "id = ? AND school_id = ?", id, school.ID).Error; err != nil {
		response.NotFound(c, "Book not found")
		return
	}

	var req createCustomBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	database.DB.Model(&book).Updates(map[string]interface{}{
		"title": req.Title, "description": req.Description,
		"url": req.URL, "version": req.Version,
	})

	if req.SubjectIDs != nil {
		var subjects []models.Subject
		database.DB.Where("id IN ?", req.SubjectIDs).Find(&subjects)
		database.DB.Model(&book).Association("Subjects").Replace(subjects)
	}
	if req.ClassIDs != nil {
		var classes []models.Class
		database.DB.Where("id IN ?", req.ClassIDs).Find(&classes)
		database.DB.Model(&book).Association("Classes").Replace(classes)
	}

	database.DB.Preload("Subjects").Preload("Classes").First(&book, "id = ?", id)
	response.Success(c, book, "Book updated")
}

func DeleteCustomBook(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
		return
	}

	var book models.Book
	if err := database.DB.First(&book, "id = ? AND school_id = ?", id, school.ID).Error; err != nil {
		response.NotFound(c, "Book not found")
		return
	}

	database.DB.Delete(&book)
	response.Success(c, nil, "Book deleted")
}

func ImportCustomBooksCSV(c *gin.Context) {
	userID := c.GetString("user_id")
	school, err := getSchoolByUserID(userID)
	if err != nil {
		response.NotFound(c, "School not found")
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
			SchoolID:  &school.ID,
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
