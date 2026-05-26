package util

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"booking-system-api/internal/config"
)

func sendEmail(to, subject, body string) {
	if config.C.SMTPHost == "" {
		log.Printf("[EMAIL] To: %s | Subject: %s\n%s", to, subject, body)
		return
	}

	auth := smtp.PlainAuth("", config.C.SMTPUser, config.C.SMTPPass, config.C.SMTPHost)
	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", config.C.SMTPFrom),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-version: 1.0;",
		`Content-Type: text/html; charset="UTF-8"`,
		"",
		body,
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", config.C.SMTPHost, config.C.SMTPPort)
	if err := smtp.SendMail(addr, auth, config.C.SMTPFrom, []string{to}, []byte(msg)); err != nil {
		log.Printf("[EMAIL ERROR] %v", err)
	}
}

func SendOTPEmail(to, name, otp string) {
	body := fmt.Sprintf(`<p>Hi %s,</p>
<p>Your password reset OTP is: <strong>%s</strong></p>
<p>This code will expire in %d minutes.</p>`, name, otp, config.C.OTPExpireMinutes)
	sendEmail(to, "Password Reset OTP", body)
}

func SendBookingStatusEmail(to, name string, bookingID int, resource, status, note string) {
	body := fmt.Sprintf(`<p>Hi %s,</p>
<p>Your booking #%d for <strong>%s</strong> has been <strong>%s</strong>.</p>`,
		name, bookingID, resource, status)
	if note != "" {
		body += fmt.Sprintf(`<p>Note: %s</p>`, note)
	}
	sendEmail(to, fmt.Sprintf("Booking #%d %s", bookingID, status), body)
}
