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
	ID        string     `gorm:"type:varchar(36);primaryKey;comment:UUID primary key" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index;comment:Soft-delete timestamp; non-null means record is deleted" json:"deleted_at,omitempty"`
}

type AuditLog struct {
	ID         string    `gorm:"type:varchar(36);primaryKey;comment:UUID primary key" json:"id"`
	UserID     string    `gorm:"type:varchar(36);index;not null;comment:ID of the user who performed the action" json:"user_id"`
	UserRole   string    `gorm:"type:varchar(50);not null;comment:Role of the user at time of action" json:"user_role"`
	Action     string    `gorm:"type:varchar(10);not null;comment:HTTP method: POST | PUT | PATCH | DELETE" json:"action"`
	Resource   string    `gorm:"type:varchar(255);not null;comment:API route path e.g. /api/admin/schools/:id" json:"resource"`
	StatusCode int       `gorm:"not null;comment:HTTP response status code" json:"status_code"`
	CreatedAt  time.Time `json:"created_at"`
}

type RevokedToken struct {
	ID        string    `gorm:"type:varchar(36);primaryKey;comment:UUID primary key" json:"id"`
	Token     string    `gorm:"type:varchar(700);uniqueIndex;not null;comment:JWT string invalidated before its natural expiry" json:"token"`
	ExpiresAt time.Time `gorm:"comment:Original expiry of the token; safe to purge after this date" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	Base
	Username     string `gorm:"type:varchar(100);uniqueIndex;not null;comment:Unique login handle" json:"username"`
	Email        string `gorm:"type:varchar(255);uniqueIndex;not null;comment:Unique login email address" json:"email"`
	PasswordHash string `gorm:"type:varchar(255);not null;comment:bcrypt hash — never returned in API responses" json:"-"`
	Role         Role   `gorm:"type:enum('admin','school_admin','teacher');not null;comment:Account type: admin | school_admin | teacher" json:"role"`
	IsActive     bool   `gorm:"default:true;comment:False suspends the account; login is rejected" json:"is_active"`
	ContactEmail string `gorm:"type:varchar(255);comment:Optional public contact address separate from login email" json:"contact_email"`
	AvatarURL    string `gorm:"type:varchar(500);comment:Path to uploaded profile photo" json:"avatar_url"`
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
	UserID          string  `gorm:"type:varchar(36);uniqueIndex;not null;comment:FK to users — the school-admin account for this school" json:"user_id"`
	Name            string  `gorm:"type:varchar(255);comment:School display name" json:"name"`
	Address         string  `gorm:"type:varchar(500);comment:Physical address of the school" json:"address"`
	ContactNumber   string  `gorm:"type:varchar(50);comment:Primary phone number" json:"contact_number"`
	ContactPerson   string  `gorm:"type:varchar(255);comment:Name of the primary contact at this school" json:"contact_person"`
	LogoURL         string  `gorm:"type:varchar(500);comment:Path to uploaded school logo" json:"logo_url"`
	MaxTeachers     int     `gorm:"default:5;comment:Quota of teacher accounts this school may create" json:"max_teachers"`
	MaxCustomBooks  int     `gorm:"default:10;comment:Quota of custom books this school may upload" json:"max_custom_books"`
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
	Title       string       `gorm:"type:varchar(500);not null;comment:Full title of the book" json:"title"`
	Description string       `gorm:"type:text;comment:Optional summary or overview of the book" json:"description"`
	URL         string       `gorm:"type:varchar(1000);comment:External reference URL for the book" json:"url"`
	PDFURL      string       `gorm:"type:varchar(1000);comment:Path to uploaded PDF file" json:"pdf_url"`
	Version     string       `gorm:"type:varchar(50);comment:Edition or version identifier e.g. 2nd Ed" json:"version"`
	SchoolID    *string      `gorm:"type:varchar(36);comment:NULL for global admin books; set for school-specific custom books" json:"school_id"`
	CreatedBy   string       `gorm:"type:varchar(36);not null;comment:User ID of the admin or school who created this book" json:"created_by"`
	School      *School      `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Subjects    []Subject    `gorm:"many2many:book_subjects;" json:"subjects,omitempty"`
	Classes     []Class      `gorm:"many2many:book_classes;" json:"classes,omitempty"`
	Curricula   []Curriculum `gorm:"many2many:book_curricula;" json:"curricula,omitempty"`
	Chapters    []Chapter    `gorm:"foreignKey:BookID" json:"chapters,omitempty"`
}

type Chapter struct {
	Base
	BookID      string      `gorm:"type:varchar(36);not null;index;comment:FK to books" json:"book_id"`
	Name        string      `gorm:"type:varchar(500);not null;comment:Chapter title" json:"name"`
	Description string      `gorm:"type:text;comment:Optional chapter summary" json:"description"`
	OrderIndex  int         `gorm:"default:0;comment:Display order within the book; lower = earlier" json:"order_index"`
	Book        Book        `gorm:"foreignKey:BookID" json:"book,omitempty"`
	Questions   []Question  `gorm:"foreignKey:ChapterID" json:"questions,omitempty"`
}

type Question struct {
	Base
	ChapterID  string       `gorm:"type:varchar(36);not null;index;comment:FK to chapters" json:"chapter_id"`
	Type       QuestionType `gorm:"type:enum('mcq','short','descriptive');not null;comment:Question format: mcq | short | descriptive" json:"type"`
	Text       string       `gorm:"type:text;not null;comment:The question body text" json:"text"`
	Difficulty Difficulty   `gorm:"type:enum('easy','medium','hard');not null;comment:Difficulty level: easy | medium | hard" json:"difficulty"`
	Chapter    Chapter      `gorm:"foreignKey:ChapterID" json:"chapter,omitempty"`
	Options    []QuestionOption    `gorm:"foreignKey:QuestionID" json:"options,omitempty"`
	AnswerKey  *QuestionAnswerKey  `gorm:"foreignKey:QuestionID" json:"answer_key,omitempty"`
	Tags       []QuestionTag       `gorm:"foreignKey:QuestionID" json:"tags,omitempty"`
}

type QuestionOption struct {
	ID         string `gorm:"type:varchar(36);primaryKey;comment:UUID primary key" json:"id"`
	QuestionID string `gorm:"type:varchar(36);not null;comment:FK to questions" json:"question_id"`
	OptionText string `gorm:"type:text;not null;comment:Display text of the option" json:"option_text"`
	IsCorrect  bool   `gorm:"default:false;comment:True for the option that is the correct answer (only one per question)" json:"is_correct"`
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
	Name      string      `gorm:"type:varchar(500);not null;comment:Paper title shown to students" json:"name"`
	SchoolID  string      `gorm:"type:varchar(36);not null;index;comment:FK to schools — school this paper belongs to" json:"school_id"`
	CreatedBy string      `gorm:"type:varchar(36);not null;index;comment:User ID of the teacher who created this paper" json:"created_by"`
	Status    PaperStatus `gorm:"type:enum('draft','submitted');default:'draft';index;comment:Workflow state: draft = editable | submitted = locked" json:"status"`
	BookID    string      `gorm:"type:varchar(36);not null;index;comment:FK to books — source book for questions" json:"book_id"`
	School    School      `gorm:"foreignKey:SchoolID" json:"school,omitempty"`
	Creator   User        `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Book      Book        `gorm:"foreignKey:BookID" json:"book,omitempty"`
	Chapters  []Chapter   `gorm:"many2many:paper_chapters;" json:"chapters,omitempty"`
	Questions []PaperQuestion `gorm:"foreignKey:PaperID" json:"questions,omitempty"`
	Config    *PaperConfig    `gorm:"foreignKey:PaperID" json:"config,omitempty"`
}

type PaperQuestion struct {
	ID         string   `gorm:"type:varchar(36);primaryKey;comment:UUID primary key" json:"id"`
	PaperID    string   `gorm:"type:varchar(36);not null;comment:FK to papers" json:"paper_id"`
	QuestionID string   `gorm:"type:varchar(36);not null;comment:FK to questions" json:"question_id"`
	Marks      float64  `gorm:"not null;comment:Points awarded for a correct answer to this question" json:"marks"`
	OrderIndex int      `gorm:"default:0;comment:Display order on the printed paper" json:"order_index"`
	Question   Question `gorm:"foreignKey:QuestionID" json:"question,omitempty"`
}

type PaperConfig struct {
	ID               string  `gorm:"type:varchar(36);primaryKey;comment:UUID primary key" json:"id"`
	PaperID          string  `gorm:"type:varchar(36);uniqueIndex;not null;comment:FK to papers (one config per paper)" json:"paper_id"`
	MCQCount         int     `gorm:"default:0;comment:Number of MCQ questions to include" json:"mcq_count"`
	ShortCount       int     `gorm:"default:0;comment:Number of short-answer questions to include" json:"short_count"`
	DescriptiveCount int     `gorm:"default:0;comment:Number of descriptive questions to include" json:"descriptive_count"`
	MCQMarks         float64 `gorm:"default:1;comment:Marks per MCQ question" json:"mcq_marks"`
	ShortMarks       float64 `gorm:"default:2;comment:Marks per short-answer question" json:"short_marks"`
	DescriptiveMarks float64 `gorm:"default:5;comment:Marks per descriptive question" json:"descriptive_marks"`
	Difficulty       string  `gorm:"default:'random';comment:Question difficulty filter: easy | medium | hard | random" json:"difficulty"`
}
