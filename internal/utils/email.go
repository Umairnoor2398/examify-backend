package utils

import (
	"fmt"
	"log"
	"net/smtp"

	"github.com/Umairnoor2398/examify-backend/internal/config"
)

func SendPasswordResetEmail(cfg *config.Config, toEmail, resetURL string) {
	if cfg.SMTP.Host == "" {
		log.Printf("[EMAIL] Password reset link for %s: %s", toEmail, resetURL)
		return
	}

	subject := "Examify - Password Reset Request"
	body := fmt.Sprintf(`
Hello,

You requested a password reset for your Examify account.

Click the link below to reset your password (valid for 1 hour):
%s

If you did not request this, please ignore this email.

Best regards,
Examify Team
`, resetURL)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		cfg.SMTP.From, toEmail, subject, body)

	addr := fmt.Sprintf("%s:%d", cfg.SMTP.Host, cfg.SMTP.Port)
	auth := smtp.PlainAuth("", cfg.SMTP.User, cfg.SMTP.Password, cfg.SMTP.Host)

	if err := smtp.SendMail(addr, auth, cfg.SMTP.From, []string{toEmail}, []byte(msg)); err != nil {
		log.Printf("[EMAIL] Failed to send reset email to %s: %v", toEmail, err)
		return
	}
	log.Printf("[EMAIL] Password reset email sent to %s", toEmail)
}
