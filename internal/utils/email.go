package utils

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"github.com/Umairnoor2398/examify-backend/internal/config"
)

func SendPasswordResetEmail(cfg *config.Config, toEmail, resetURL string) {
	if cfg.SMTP.Host == "" {
		log.Printf("[EMAIL] Password reset link for %s: %s", toEmail, resetURL)
		return
	}

	subject := "Examify - Password Reset Request"
	html := buildResetEmailHTML(resetURL)

	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", cfg.SMTP.From),
		fmt.Sprintf("To: %s", toEmail),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		`Content-Type: text/html; charset="UTF-8"`,
		"",
		html,
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", cfg.SMTP.Host, cfg.SMTP.Port)
	auth := smtp.PlainAuth("", cfg.SMTP.User, cfg.SMTP.Password, cfg.SMTP.Host)

	if err := smtp.SendMail(addr, auth, cfg.SMTP.From, []string{toEmail}, []byte(msg)); err != nil {
		log.Printf("[EMAIL] Failed to send reset email to %s: %v", toEmail, err)
		return
	}
	log.Printf("[EMAIL] Password reset email sent to %s", toEmail)
}

func buildResetEmailHTML(resetURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Password Reset</title>
</head>
<body style="margin:0;padding:0;background-color:#f3f4f6;font-family:Arial,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f3f4f6;padding:40px 0;">
  <tr>
    <td align="center">
      <table width="560" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,0.1);">
        <!-- Header -->
        <tr>
          <td style="background-color:#2563eb;padding:32px 40px;text-align:center;">
            <h1 style="margin:0;color:#ffffff;font-size:24px;font-weight:700;letter-spacing:-0.5px;">Examify</h1>
            <p style="margin:4px 0 0;color:#bfdbfe;font-size:14px;">School Exam Portal</p>
          </td>
        </tr>
        <!-- Body -->
        <tr>
          <td style="padding:40px;">
            <h2 style="margin:0 0 16px;color:#111827;font-size:20px;">Reset your password</h2>
            <p style="margin:0 0 24px;color:#4b5563;font-size:15px;line-height:1.6;">
              We received a request to reset the password for your Examify account.
              Click the button below to choose a new password. This link is valid for <strong>1 hour</strong>.
            </p>
            <table cellpadding="0" cellspacing="0" style="margin:0 0 32px;">
              <tr>
                <td style="border-radius:6px;background-color:#2563eb;">
                  <a href="%s"
                     style="display:inline-block;padding:14px 32px;color:#ffffff;font-size:15px;font-weight:600;text-decoration:none;border-radius:6px;">
                    Reset Password
                  </a>
                </td>
              </tr>
            </table>
            <p style="margin:0 0 8px;color:#6b7280;font-size:13px;">
              If the button doesn&apos;t work, copy and paste this link into your browser:
            </p>
            <p style="margin:0 0 24px;word-break:break-all;">
              <a href="%s" style="color:#2563eb;font-size:13px;">%s</a>
            </p>
            <p style="margin:0;color:#6b7280;font-size:13px;line-height:1.6;">
              If you did not request a password reset, you can safely ignore this email.
              Your password will not be changed.
            </p>
          </td>
        </tr>
        <!-- Footer -->
        <tr>
          <td style="background-color:#f9fafb;padding:24px 40px;border-top:1px solid #e5e7eb;text-align:center;">
            <p style="margin:0;color:#9ca3af;font-size:12px;">
              &copy; Examify. All rights reserved.
            </p>
          </td>
        </tr>
      </table>
    </td>
  </tr>
</table>
</body>
</html>`, resetURL, resetURL, resetURL)
}
