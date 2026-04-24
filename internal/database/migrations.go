package database

import (
	"fmt"

	"github.com/Umairnoor2398/examify-backend/internal/models"
	"gorm.io/gorm"
)

// CreateStoredProcedures installs (or replaces) the two school management procedures.
// Safe to call on every startup — each procedure is dropped before re-creation.
func CreateStoredProcedures(db *gorm.DB) error {
	procs := []struct {
		name string
		body string
	}{
		{
			name: "sp_school_set_status",
			body: `
CREATE PROCEDURE sp_school_set_status(
    IN p_school_id VARCHAR(36),
    IN p_action    VARCHAR(20)
)
BEGIN
    DECLARE v_user_id VARCHAR(36);
    SELECT user_id INTO v_user_id FROM schools WHERE id = p_school_id;

    IF p_action = 'soft_delete' THEN
        UPDATE schools SET deleted_at = NOW() WHERE id = p_school_id AND deleted_at IS NULL;
        UPDATE users SET is_active = FALSE WHERE id = v_user_id;
        UPDATE users SET is_active = FALSE
            WHERE id IN (SELECT user_id FROM teacher_schools WHERE school_id = p_school_id);

    ELSEIF p_action = 'activate' THEN
        UPDATE schools SET deleted_at = NULL WHERE id = p_school_id;
        UPDATE users SET is_active = TRUE WHERE id = v_user_id;
        UPDATE users SET is_active = TRUE
            WHERE id IN (SELECT user_id FROM teacher_schools WHERE school_id = p_school_id);

    ELSEIF p_action = 'deactivate' THEN
        UPDATE users SET is_active = FALSE WHERE id = v_user_id;
        UPDATE users SET is_active = FALSE
            WHERE id IN (SELECT user_id FROM teacher_schools WHERE school_id = p_school_id);
    END IF;
END`,
		},
		{
			name: "sp_school_hard_delete",
			body: `
CREATE PROCEDURE sp_school_hard_delete(
    IN p_school_id VARCHAR(36)
)
BEGIN
    DECLARE v_user_id VARCHAR(36);
    SELECT user_id INTO v_user_id FROM schools WHERE id = p_school_id;

    -- Papers
    DELETE pq FROM paper_questions pq
        INNER JOIN papers p ON pq.paper_id = p.id
        WHERE p.school_id = p_school_id;

    DELETE pc FROM paper_configs pc
        INNER JOIN papers p ON pc.paper_id = p.id
        WHERE p.school_id = p_school_id;

    DELETE pc FROM paper_chapters pc
        INNER JOIN papers p ON pc.paper_id = p.id
        WHERE p.school_id = p_school_id;

    DELETE FROM papers WHERE school_id = p_school_id;

    -- Custom books owned by this school
    DELETE qt FROM question_tags qt
        INNER JOIN questions q  ON qt.question_id = q.id
        INNER JOIN chapters  ch ON q.chapter_id   = ch.id
        INNER JOIN books     b  ON ch.book_id      = b.id
        WHERE b.school_id = p_school_id;

    DELETE qak FROM question_answer_keys qak
        INNER JOIN questions q  ON qak.question_id = q.id
        INNER JOIN chapters  ch ON q.chapter_id    = ch.id
        INNER JOIN books     b  ON ch.book_id       = b.id
        WHERE b.school_id = p_school_id;

    DELETE qo FROM question_options qo
        INNER JOIN questions q  ON qo.question_id = q.id
        INNER JOIN chapters  ch ON q.chapter_id   = ch.id
        INNER JOIN books     b  ON ch.book_id      = b.id
        WHERE b.school_id = p_school_id;

    DELETE q FROM questions q
        INNER JOIN chapters ch ON q.chapter_id = ch.id
        INNER JOIN books    b  ON ch.book_id    = b.id
        WHERE b.school_id = p_school_id;

    DELETE ch FROM chapters ch
        INNER JOIN books b ON ch.book_id = b.id
        WHERE b.school_id = p_school_id;

    DELETE bs  FROM book_subjects  bs  INNER JOIN books b ON bs.book_id  = b.id WHERE b.school_id = p_school_id;
    DELETE bcl FROM book_classes   bcl INNER JOIN books b ON bcl.book_id = b.id WHERE b.school_id = p_school_id;
    DELETE bcu FROM book_curricula bcu INNER JOIN books b ON bcu.book_id = b.id WHERE b.school_id = p_school_id;

    DELETE FROM books WHERE school_id = p_school_id;

    -- Teachers
    DELETE FROM password_reset_tokens
        WHERE user_id IN (SELECT user_id FROM teacher_schools WHERE school_id = p_school_id);

    DELETE FROM users
        WHERE id IN (SELECT user_id FROM teacher_schools WHERE school_id = p_school_id);

    DELETE FROM teacher_schools WHERE school_id = p_school_id;

    -- School
    DELETE FROM school_curricula WHERE school_id = p_school_id;
    DELETE FROM schools WHERE id = p_school_id;

    -- School admin account
    DELETE FROM password_reset_tokens WHERE user_id = v_user_id;
    DELETE FROM users WHERE id = v_user_id;
END`,
		},
	}

	for _, p := range procs {
		if err := db.Exec("DROP PROCEDURE IF EXISTS " + p.name).Error; err != nil {
			return fmt.Errorf("drop procedure %s: %w", p.name, err)
		}
		if err := db.Exec(p.body).Error; err != nil {
			return fmt.Errorf("create procedure %s: %w", p.name, err)
		}
	}
	return nil
}

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
