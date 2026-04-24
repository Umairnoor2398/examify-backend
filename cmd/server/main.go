package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Umairnoor2398/examify-backend/internal/config"
	"github.com/Umairnoor2398/examify-backend/internal/database"
	adminHandler "github.com/Umairnoor2398/examify-backend/internal/handlers/admin"
	authHandler "github.com/Umairnoor2398/examify-backend/internal/handlers/auth"
	schoolHandler "github.com/Umairnoor2398/examify-backend/internal/handlers/school"
	teacherHandler "github.com/Umairnoor2398/examify-backend/internal/handlers/teacher"
	"github.com/Umairnoor2398/examify-backend/internal/middleware"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	if err := database.ApplyColumnComments(db); err != nil {
		log.Fatalf("Failed to apply column comments: %v", err)
	}

	// Create default admin if not exists
	seedAdmin(cfg)

	// Ensure upload dirs exist
	os.MkdirAll(cfg.Upload.Dir+"/books", 0755)
	os.MkdirAll(cfg.Upload.Dir+"/logos", 0755)
	os.MkdirAll(cfg.Upload.Dir+"/avatars", 0755)

	r := gin.Default()

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Serve uploaded files without directory listing
	r.GET("/uploads/*fp", func(c *gin.Context) {
		filePath := filepath.Join(cfg.Upload.Dir, c.Param("fp"))
		info, err := os.Stat(filePath)
		if err != nil || info.IsDir() {
			c.Status(http.StatusNotFound)
			return
		}
		c.File(filePath)
	})

	// HTTPS redirect in production (behind a reverse proxy that sets X-Forwarded-Proto)
	if cfg.App.Env == "production" {
		r.Use(func(c *gin.Context) {
			if c.Request.Header.Get("X-Forwarded-Proto") == "http" {
				c.Redirect(http.StatusMovedPermanently, "https://"+c.Request.Host+c.Request.RequestURI)
				c.Abort()
				return
			}
			c.Next()
		})
	}

	auth := authHandler.NewHandler(cfg)

	v1 := r.Group("/api/v1")
	{
		// Public auth routes (rate-limited)
		authRoutes := v1.Group("/auth")
		authRoutes.Use(middleware.RateLimit())
		{
			authRoutes.POST("/login", auth.Login)
			authRoutes.POST("/refresh", auth.RefreshToken)
			authRoutes.POST("/logout", auth.Logout)
			authRoutes.POST("/forgot-password", auth.ForgotPassword)
			authRoutes.POST("/reset-password", auth.ResetPassword)
		}

		// Protected auth routes
		protectedAuth := v1.Group("/auth")
		protectedAuth.Use(middleware.Auth(cfg))
		{
			protectedAuth.GET("/me", auth.Me)
			protectedAuth.GET("/permissions", auth.Permissions)
		}

		// Admin routes
		adminRoutes := v1.Group("/admin")
		adminRoutes.Use(middleware.Auth(cfg), middleware.RequireRole(models.RoleAdmin), middleware.Audit())
		{
			// Profile
			adminRoutes.GET("/profile", adminHandler.GetProfile)
			adminRoutes.PUT("/profile", adminHandler.UpdateProfile)
			adminRoutes.POST("/profile/avatar", func(c *gin.Context) { adminHandler.UploadAvatar(c, cfg) })

			// Schools
			adminRoutes.GET("/schools", adminHandler.ListSchools)
			adminRoutes.POST("/schools", adminHandler.CreateSchool)
			adminRoutes.GET("/schools/:id", adminHandler.GetSchool)
			adminRoutes.PUT("/schools/:id", adminHandler.UpdateSchool)
			adminRoutes.DELETE("/schools/:id", adminHandler.DeleteSchool)
			adminRoutes.PATCH("/schools/:id/toggle", adminHandler.ToggleSchoolStatus)

			// Curricula
			adminRoutes.GET("/curricula", adminHandler.ListCurricula)
			adminRoutes.POST("/curricula", adminHandler.CreateCurriculum)
			adminRoutes.PUT("/curricula/:id", adminHandler.UpdateCurriculum)
			adminRoutes.DELETE("/curricula/:id", adminHandler.DeleteCurriculum)

			// Subjects
			adminRoutes.GET("/subjects", adminHandler.ListSubjects)
			adminRoutes.POST("/subjects", adminHandler.CreateSubject)
			adminRoutes.PUT("/subjects/:id", adminHandler.UpdateSubject)
			adminRoutes.DELETE("/subjects/:id", adminHandler.DeleteSubject)

			// Classes
			adminRoutes.GET("/classes", adminHandler.ListClasses)
			adminRoutes.POST("/classes", adminHandler.CreateClass)
			adminRoutes.PUT("/classes/:id", adminHandler.UpdateClass)
			adminRoutes.DELETE("/classes/:id", adminHandler.DeleteClass)

			// Books
			adminRoutes.GET("/books", adminHandler.ListBooks)
			adminRoutes.POST("/books", adminHandler.CreateBook)
			adminRoutes.GET("/books/:id", adminHandler.GetBook)
			adminRoutes.PUT("/books/:id", adminHandler.UpdateBook)
			adminRoutes.DELETE("/books/:id", adminHandler.DeleteBook)
			adminRoutes.POST("/books/import", func(c *gin.Context) { adminHandler.ImportBooksCSV(c, cfg) })
			adminRoutes.POST("/books/:id/pdf", func(c *gin.Context) { adminHandler.UploadBookPDF(c, cfg) })

			// Chapters
			adminRoutes.GET("/books/:id/chapters", adminHandler.ListChapters)
			adminRoutes.POST("/books/:id/chapters", adminHandler.CreateChapter)
			adminRoutes.GET("/books/:id/chapters/:chapterId", adminHandler.GetChapter)
			adminRoutes.PUT("/books/:id/chapters/:chapterId", adminHandler.UpdateChapter)
			adminRoutes.DELETE("/books/:id/chapters/:chapterId", adminHandler.DeleteChapter)
			adminRoutes.POST("/books/:id/chapters/import", adminHandler.ImportChaptersCSV)

			// Questions
			adminRoutes.GET("/chapters/:chapterId/questions", adminHandler.ListQuestions)
			adminRoutes.POST("/chapters/:chapterId/questions", adminHandler.CreateQuestion)
			adminRoutes.GET("/chapters/:chapterId/questions/:id", adminHandler.GetQuestion)
			adminRoutes.PUT("/chapters/:chapterId/questions/:id", adminHandler.UpdateQuestion)
			adminRoutes.DELETE("/chapters/:chapterId/questions/:id", adminHandler.DeleteQuestion)
			adminRoutes.POST("/chapters/:chapterId/questions/import", adminHandler.ImportQuestionsCSV)
		}

		// School-Admin routes
		schoolRoutes := v1.Group("/school")
		schoolRoutes.Use(middleware.Auth(cfg), middleware.RequireRole(models.RoleSchoolAdmin), middleware.Audit())
		{
			schoolRoutes.GET("/profile", schoolHandler.GetProfile)
			schoolRoutes.PUT("/profile", schoolHandler.UpdateProfile)
			schoolRoutes.POST("/profile/logo", func(c *gin.Context) { schoolHandler.UploadLogo(c, cfg) })
			schoolRoutes.POST("/profile/avatar", func(c *gin.Context) { schoolHandler.UploadAvatar(c, cfg) })

			// Teachers
			schoolRoutes.GET("/teachers", schoolHandler.ListTeachers)
			schoolRoutes.POST("/teachers", schoolHandler.CreateTeacher)
			schoolRoutes.PATCH("/teachers/:id/toggle", schoolHandler.ToggleTeacher)
			schoolRoutes.DELETE("/teachers/:id", schoolHandler.DeleteTeacher)

			// Custom books
			schoolRoutes.GET("/books", schoolHandler.ListCustomBooks)
			schoolRoutes.POST("/books", schoolHandler.CreateCustomBook)
			schoolRoutes.GET("/books/:id", schoolHandler.GetCustomBook)
			schoolRoutes.PUT("/books/:id", schoolHandler.UpdateCustomBook)
			schoolRoutes.DELETE("/books/:id", schoolHandler.DeleteCustomBook)
			schoolRoutes.POST("/books/import", schoolHandler.ImportCustomBooksCSV)

			// Chapters & Questions for custom books (reuse admin handlers)
			schoolRoutes.GET("/books/:id/chapters", adminHandler.ListChapters)
			schoolRoutes.POST("/books/:id/chapters", adminHandler.CreateChapter)
			schoolRoutes.PUT("/books/:id/chapters/:chapterId", adminHandler.UpdateChapter)
			schoolRoutes.DELETE("/books/:id/chapters/:chapterId", adminHandler.DeleteChapter)
			schoolRoutes.POST("/books/:id/chapters/import", adminHandler.ImportChaptersCSV)
			schoolRoutes.GET("/chapters/:chapterId/questions", adminHandler.ListQuestions)
			schoolRoutes.POST("/chapters/:chapterId/questions", adminHandler.CreateQuestion)
			schoolRoutes.PUT("/chapters/:chapterId/questions/:id", adminHandler.UpdateQuestion)
			schoolRoutes.DELETE("/chapters/:chapterId/questions/:id", adminHandler.DeleteQuestion)
			schoolRoutes.POST("/chapters/:chapterId/questions/import", adminHandler.ImportQuestionsCSV)

			// Read-only subjects & classes for book creation
			schoolRoutes.GET("/subjects", adminHandler.ListSubjects)
			schoolRoutes.GET("/classes", adminHandler.ListClasses)

			// Papers
			schoolRoutes.GET("/papers", schoolHandler.ListPapers)
			schoolRoutes.GET("/papers/:id", schoolHandler.GetPaper)
			schoolRoutes.DELETE("/papers/:id", schoolHandler.DeletePaper)

			// School admin can also create papers (same as teacher)
			schoolRoutes.POST("/papers", teacherHandler.CreatePaper)
			schoolRoutes.PUT("/papers/:id", teacherHandler.UpdatePaper)
			schoolRoutes.POST("/papers/:id/submit", teacherHandler.SubmitPaper)
		}

		// Teacher routes
		teacherRoutes := v1.Group("/teacher")
		teacherRoutes.Use(middleware.Auth(cfg), middleware.RequireRole(models.RoleTeacher), middleware.Audit())
		{
			// Profile
			teacherRoutes.GET("/profile", teacherHandler.GetProfile)
			teacherRoutes.PUT("/profile", teacherHandler.UpdateProfile)
			teacherRoutes.POST("/profile/avatar", func(c *gin.Context) { teacherHandler.UploadAvatar(c, cfg) })

			teacherRoutes.GET("/classes", teacherHandler.ListFilteredClasses)
			teacherRoutes.GET("/subjects", teacherHandler.ListFilteredSubjects)
			teacherRoutes.GET("/books", teacherHandler.ListBooks)
			teacherRoutes.GET("/books/:bookId/chapters", teacherHandler.ListBookChapters)
			teacherRoutes.GET("/questions", teacherHandler.GetFilteredQuestions)

			teacherRoutes.GET("/papers", teacherHandler.ListPapers)
			teacherRoutes.POST("/papers", teacherHandler.CreatePaper)
			teacherRoutes.GET("/papers/:id", teacherHandler.GetPaper)
			teacherRoutes.PUT("/papers/:id", teacherHandler.UpdatePaper)
			teacherRoutes.DELETE("/papers/:id", teacherHandler.DeletePaper)
			teacherRoutes.POST("/papers/:id/submit", teacherHandler.SubmitPaper)
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "examify-backend"})
	})

	log.Printf("Examify API server starting on port %s", cfg.App.Port)
	if err := r.Run(":" + cfg.App.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func seedAdmin(cfg *config.Config) {
	var count int64
	database.DB.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&count)
	if count > 0 {
		return
	}

	hash, _ := utils.HashPassword("Admin@123456")
	admin := models.User{
		Base:         models.Base{ID: utils.NewUUID()},
		Username:     "admin",
		Email:        "admin@examify.com",
		PasswordHash: hash,
		Role:         models.RoleAdmin,
		IsActive:     true,
	}
	if err := database.DB.Create(&admin).Error; err == nil {
		log.Println("Default admin created: admin@examify.com / Admin@123456")
	}
}
