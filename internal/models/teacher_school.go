package models

// TeacherSchool is the join table linking teachers to their school.
// A teacher can only belong to one school.
type TeacherSchool struct {
	UserID   string `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	SchoolID string `gorm:"type:varchar(36);primaryKey" json:"school_id"`
}
