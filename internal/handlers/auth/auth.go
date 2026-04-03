package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/Umairnoor2398/examify-backend/internal/config"
	"github.com/Umairnoor2398/examify-backend/internal/database"
	"github.com/Umairnoor2398/examify-backend/internal/middleware"
	"github.com/Umairnoor2398/examify-backend/internal/models"
	"github.com/Umairnoor2398/examify-backend/internal/utils"
	"github.com/Umairnoor2398/examify-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	cfg *config.Config
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var user models.User
	if err := database.DB.Where("email = ? AND deleted_at IS NULL", req.Email).First(&user).Error; err != nil {
		response.Unauthorized(c, "Invalid email or password")
		return
	}

	if !user.IsActive {
		response.Unauthorized(c, "Your account has been disabled. Please contact your administrator.")
		return
	}

	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		response.Unauthorized(c, "Invalid email or password")
		return
	}

	accessToken, err := utils.GenerateAccessToken(user.ID, string(user.Role), h.cfg.JWT.Secret, h.cfg.JWT.AccessExpiry)
	if err != nil {
		response.InternalError(c, "Failed to generate token")
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, string(user.Role), h.cfg.JWT.RefreshSecret, h.cfg.JWT.RefreshExpiry)
	if err != nil {
		response.InternalError(c, "Failed to generate token")
		return
	}

	response.Success(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	}, "Login successful")
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	claims, err := utils.ParseToken(req.RefreshToken, h.cfg.JWT.RefreshSecret)
	if err != nil || claims.Type != "refresh" {
		response.Unauthorized(c, "Invalid or expired refresh token")
		return
	}

	var user models.User
	if err := database.DB.Where("id = ? AND is_active = true AND deleted_at IS NULL", claims.UserID).First(&user).Error; err != nil {
		response.Unauthorized(c, "User not found or inactive")
		return
	}

	accessToken, err := utils.GenerateAccessToken(user.ID, string(user.Role), h.cfg.JWT.Secret, h.cfg.JWT.AccessExpiry)
	if err != nil {
		response.InternalError(c, "Failed to generate token")
		return
	}

	response.Success(c, gin.H{"access_token": accessToken}, "Token refreshed")
}

func (h *Handler) Me(c *gin.Context) {
	userID := c.GetString("user_id")

	var user models.User
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		response.NotFound(c, "User not found")
		return
	}

	response.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	}, "")
}

func (h *Handler) Permissions(c *gin.Context) {
	role := models.Role(c.GetString("role"))
	permissions := middleware.GetPermissions(role)
	response.Success(c, gin.H{"permissions": permissions, "role": role}, "")
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var user models.User
	if err := database.DB.Where("email = ? AND deleted_at IS NULL", req.Email).First(&user).Error; err != nil {
		// Return success even if user not found to prevent email enumeration
		response.Success(c, nil, "If that email exists, a reset link has been sent.")
		return
	}

	// Invalidate existing unused tokens
	database.DB.Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND used_at IS NULL", user.ID).
		Update("used_at", time.Now())

	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	resetToken := models.PasswordResetToken{
		ID:        utils.NewUUID(),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	database.DB.Create(&resetToken)

	resetURL := h.cfg.App.URL + "/reset-password?token=" + token
	go utils.SendPasswordResetEmail(h.cfg, user.Email, resetURL)

	response.Success(c, nil, "If that email exists, a reset link has been sent.")
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var resetToken models.PasswordResetToken
	if err := database.DB.Where("token = ? AND used_at IS NULL", req.Token).First(&resetToken).Error; err != nil {
		response.BadRequest(c, "Invalid or expired reset token")
		return
	}

	if time.Now().After(resetToken.ExpiresAt) {
		response.BadRequest(c, "Reset token has expired")
		return
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		response.InternalError(c, "Failed to hash password")
		return
	}

	now := time.Now()
	database.DB.Model(&resetToken).Update("used_at", now)
	database.DB.Model(&models.User{}).Where("id = ?", resetToken.UserID).Update("password_hash", hash)

	response.Success(c, nil, "Password reset successfully")
}
