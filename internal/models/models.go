package models

import (
	"time"
)

type Role string
type QuestionType string
type Difficulty string
type PaperStatus string

const (
	RoleAdmin       Role = "admin"
	RoleSchoolAdmin Role = "school_admin"
	RoleTeacher     Role = "teacher"

	QuestionMCQ         QuestionType = "mcq"
	QuestionShort       QuestionType = "short"
	QuestionDescriptive QuestionType = "descriptive"

	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"

	PaperDraft     PaperStatus = "draft"
	PaperSubmitted PaperStatus = "submitted"
)

type Base struct {
	ID        string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

type AuditLog struct {
	ID         string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID     string    `gorm:"type:varchar(36);index;not null" json:"user_id"`
	UserRole   string    `gorm:"type:varchar(50);not null" json:"user_role"`
	Action     string    `gorm:"type:varchar(10);not null" json:"action"`
	Resource   string    `gorm:"type:varchar(255);not null" json:"resource"`
	StatusCode int       `gorm:"not null" json:"status_code"`
	CreatedAt  time.Time `json:"created_at"`
}

type RevokedToken struct {
	ID        string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Token     string    `gorm:"type:varchar(700);uniqueIndex;not null" json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	Base
	Username     string `gorm:"type:varchar(100);uniqueIndex;not null" json:"username"`
	Email        string `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string `gorm:"type:varchar(255);not null" json:"-"`
	Role         Role   `gorm:"type:enum('admin','school_admin','teacher');not null" json:"role"`
	IsActive     bool   `gorm:"default:true" json:"is_active"`
	ContactEmail string `gorm:"type:varchar(255)" json:"contact_email"`
	AvatarURL    string `gorm:"type:varchar(500)" json:"avatar_url"`
}

type PasswordResetToken struct {
	ID        string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID    string     `gorm:"type:varchar(36);not null" json:"user_id"`
	Token     string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	User      User       `gorm:"foreignKey:UserID" json:"-"`
}

type School struct {
	Base
	UserID          string  `gorm:"type:varchar(36);uniqueIndex;not null" json:"user_id"`
	Name            string  `gorm:"type:varchar(255)" json:"name"`
	Address         string  `gorm:"type:varchar(500)" json:"address"`
	ContactNumber   string  `gorm:"type:varchar(50)" json:"contact_number"`
	ContactPerson   string  `gorm:"type:varchar(255)" json:"contact_person"`
	LogoURL         string  `gorm:"type:varchar(500)" json:"logo_url"`
	MaxTeachers     int     `gorm:"default:5" json:"max_teachers"`
	MaxCustomBooks  int     `gorm:"default:10" json:"max_custom_books"`
	User            User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Curricula       []Curriculum `gorm:"many2many:school_curricula;" json:"curricula,omitempty"`
}

type Curriculum struct {
	Base
	Name    string   `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
	Schools []School `gorm:"many2many:school_curricula;" json:"-"`
	Books   []Book   `gorm:"many2many:book_curricula;" json:"-"`
}

type Subject struct {
	Base
	Name  string `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
	Books []Book `gorm:"many2many:book_subjects;" json:"-"`
}

type Class struct {
	Base
	Name  string `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
	Books []Book `gorm:"many2many:book_classes;" json:"-"`
}

type Book struct {
	Base
	Title       string       `gorm:"type:varchar(500);not null" json:"title"`
	Description string       `gorm:"type:text" json:"description"`
	URL         string       `gorm:"type:varchar(1000)" json:"url"`
	PDFURL      string       `gorm:"type:varchar(1000)" json:"pdf_url"`
	Version     string       `gorm:"type:varchar(50)" json:"version"`
	SchoolID    *string      `gorm:"type:varchar(36)" json:"school_id"`
	CreatedBy   string       `gorm:"type:varchar(36);not null" json:"created_by"`
	School      *School      `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Subjects    []Subject    `gorm:"many2many:book_subjects;" json:"subjects,omitempty"`
	Classes     []Class      `gorm:"many2many:book_classes;" json:"classes,omitempty"`
	Curricula   []Curriculum `gorm:"many2many:book_curricula;" json:"curricula,omitempty"`
	Chapters    []Chapter    `gorm:"foreignKey:BookID" json:"chapters,omitempty"`
}

type Chapter struct {
	Base
	BookID      string      `gorm:"type:varchar(36);not null;index" json:"book_id"`
	Name        string      `gorm:"type:varchar(500);not null" json:"name"`
	Description string      `gorm:"type:text" json:"description"`
	OrderIndex  int         `gorm:"default:0" json:"order_index"`
	Book        Book        `gorm:"foreignKey:BookID" json:"book,omitempty"`
	Questions   []Question  `gorm:"foreignKey:ChapterID" json:"questions,omitempty"`
}

type Question struct {
	Base
	ChapterID  string       `gorm:"type:varchar(36);not null;index" json:"chapter_id"`
	Type       QuestionType `gorm:"type:enum('mcq','short','descriptive');not null" json:"type"`
	Text       string       `gorm:"type:text;not null" json:"text"`
	Difficulty Difficulty   `gorm:"type:enum('easy','medium','hard');not null" json:"difficulty"`
	Chapter    Chapter      `gorm:"foreignKey:ChapterID" json:"chapter,omitempty"`
	Options    []QuestionOption    `gorm:"foreignKey:QuestionID" json:"options,omitempty"`
	AnswerKey  *QuestionAnswerKey  `gorm:"foreignKey:QuestionID" json:"answer_key,omitempty"`
	Tags       []QuestionTag       `gorm:"foreignKey:QuestionID" json:"tags,omitempty"`
}

type QuestionOption struct {
	ID         string `gorm:"type:varchar(36);primaryKey" json:"id"`
	QuestionID string `gorm:"type:varchar(36);not null" json:"question_id"`
	OptionText string `gorm:"type:text;not null" json:"option_text"`
	IsCorrect  bool   `gorm:"default:false" json:"is_correct"`
}

type QuestionAnswerKey struct {
	ID         string `gorm:"type:varchar(36);primaryKey" json:"id"`
	QuestionID string `gorm:"type:varchar(36);uniqueIndex;not null" json:"question_id"`
	AnswerText string `gorm:"type:text;not null" json:"answer_text"`
}

type QuestionTag struct {
	ID         string `gorm:"type:varchar(36);primaryKey" json:"id"`
	QuestionID string `gorm:"type:varchar(36);not null;index" json:"question_id"`
	Tag        string `gorm:"type:varchar(100);not null" json:"tag"`
}

type Paper struct {
	Base
	Name      string      `gorm:"type:varchar(500);not null" json:"name"`
	SchoolID  string      `gorm:"type:varchar(36);not null;index" json:"school_id"`
	CreatedBy string      `gorm:"type:varchar(36);not null;index" json:"created_by"`
	Status    PaperStatus `gorm:"type:enum('draft','submitted');default:'draft';index" json:"status"`
	BookID    string      `gorm:"type:varchar(36);not null;index" json:"book_id"`
	School    School      `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Creator   User        `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Book      Book        `gorm:"foreignKey:BookID" json:"book,omitempty"`
	Chapters  []Chapter   `gorm:"many2many:paper_chapters;" json:"chapters,omitempty"`
	Questions []PaperQuestion `gorm:"foreignKey:PaperID" json:"questions,omitempty"`
	Config    *PaperConfig    `gorm:"foreignKey:PaperID" json:"config,omitempty"`
}

type PaperQuestion struct {
	ID         string   `gorm:"type:varchar(36);primaryKey" json:"id"`
	PaperID    string   `gorm:"type:varchar(36);not null" json:"paper_id"`
	QuestionID string   `gorm:"type:varchar(36);not null" json:"question_id"`
	Marks      float64  `gorm:"not null" json:"marks"`
	OrderIndex int      `gorm:"default:0" json:"order_index"`
	Question   Question `gorm:"foreignKey:QuestionID" json:"question,omitempty"`
}

type PaperConfig struct {
	ID               string  `gorm:"type:varchar(36);primaryKey" json:"id"`
	PaperID          string  `gorm:"type:varchar(36);uniqueIndex;not null" json:"paper_id"`
	MCQCount         int     `gorm:"default:0" json:"mcq_count"`
	ShortCount       int     `gorm:"default:0" json:"short_count"`
	DescriptiveCount int     `gorm:"default:0" json:"descriptive_count"`
	MCQMarks         float64 `gorm:"default:1" json:"mcq_marks"`
	ShortMarks       float64 `gorm:"default:2" json:"short_marks"`
	DescriptiveMarks float64 `gorm:"default:5" json:"descriptive_marks"`
	Difficulty       string  `gorm:"default:'random'" json:"difficulty"`
}
