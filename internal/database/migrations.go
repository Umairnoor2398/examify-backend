package database

import (
	"fmt"

	"github.com/Umairnoor2398/examify-backend/internal/models"
	"gorm.io/gorm"
)

// ApplyColumnComments runs ALTER COLUMN for every field that carries a comment tag.
// It is idempotent — safe to call on every startup.
func ApplyColumnComments(db *gorm.DB) error {
	type entry struct {
		model  interface{}
		fields []string
	}

	migrations := []entry{
		{&models.User{}, []string{"Username", "Email", "PasswordHash", "Role", "IsActive", "ContactEmail", "AvatarURL"}},
		{&models.School{}, []string{"UserID", "Name", "Address", "ContactNumber", "ContactPerson", "LogoURL", "MaxTeachers", "MaxCustomBooks"}},
		{&models.Book{}, []string{"Title", "Description", "URL", "PDFURL", "Version", "SchoolID", "CreatedBy"}},
		{&models.Chapter{}, []string{"BookID", "Name", "Description", "OrderIndex"}},
		{&models.Question{}, []string{"ChapterID", "Type", "Text", "Difficulty"}},
		{&models.QuestionOption{}, []string{"ID", "QuestionID", "OptionText", "IsCorrect"}},
		{&models.Paper{}, []string{"Name", "SchoolID", "CreatedBy", "Status", "BookID"}},
		{&models.PaperQuestion{}, []string{"ID", "PaperID", "QuestionID", "Marks", "OrderIndex"}},
		{&models.PaperConfig{}, []string{"ID", "PaperID", "MCQCount", "ShortCount", "DescriptiveCount", "MCQMarks", "ShortMarks", "DescriptiveMarks", "Difficulty"}},
		{&models.AuditLog{}, []string{"ID", "UserID", "UserRole", "Action", "Resource", "StatusCode"}},
		{&models.RevokedToken{}, []string{"ID", "Token", "ExpiresAt"}},
	}

	m := db.Migrator()
	for _, e := range migrations {
		for _, field := range e.fields {
			if err := m.AlterColumn(e.model, field); err != nil {
				return fmt.Errorf("apply comment on %T.%s: %w", e.model, field, err)
			}
		}
	}
	return nil
}
