package database

import (
	"fmt"
	"log"

	"github.com/Umairnoor2398/examify-backend/internal/config"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) (*gorm.DB, error) {
	logLevel := logger.Silent
	if cfg.App.Env == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(mysql.Open(cfg.Database.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	DB = db
	log.Println("Database connected successfully")
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.PasswordResetToken{},
		&models.School{},
		&models.Curriculum{},
		&models.Subject{},
		&models.Class{},
		&models.Book{},
		&models.Chapter{},
		&models.Question{},
		&models.QuestionOption{},
		&models.QuestionAnswerKey{},
		&models.QuestionTag{},
		&models.Paper{},
		&models.PaperQuestion{},
		&models.PaperConfig{},
		&models.TeacherSchool{},
		&models.RevokedToken{},
		&models.AuditLog{},
	)
}
