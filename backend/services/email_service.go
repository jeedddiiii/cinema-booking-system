package services

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/smtp"
	"os"
	"time"
)

type EmailService struct {
	host     string
	port     string
	username string
	password string
	from     string
	enabled  bool
}

func NewEmailService() *EmailService {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")

	enabled := host != "" && port != "" && username != "" && password != ""

	if enabled {
		log.Printf("✅ Email service initialized (host: %s:%s)", host, port)
	} else {
		log.Println("⚠️ Email service not configured (missing SMTP_* environment variables)")
	}

	return &EmailService{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		enabled:  enabled,
	}
}

type BookingConfirmationData struct {
	UserName    string
	BookingID   string
	MovieTitle  string
	Theater     string
	Seats       []string
	TotalAmount float64
	BookingDate string
}

func (s *EmailService) SendBookingConfirmation(to string, data BookingConfirmationData) error {
	if !s.enabled {
		log.Printf("📧 Email not sent (not configured): Booking confirmation for %s", to)
		return nil
	}

	subject := fmt.Sprintf("🎬 Booking Confirmed - %s", data.MovieTitle)

	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #e11d48, #9333ea); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: #f9fafb; padding: 30px; border-radius: 0 0 10px 10px; }
        .booking-details { background: white; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .detail-row { display: flex; justify-content: space-between; padding: 10px 0; border-bottom: 1px solid #eee; }
        .detail-label { color: #666; }
        .detail-value { font-weight: bold; }
        .seats { background: #fef3c7; padding: 15px; border-radius: 8px; text-align: center; margin: 20px 0; }
        .total { font-size: 24px; color: #e11d48; text-align: center; margin: 20px 0; }
        .footer { text-align: center; color: #666; font-size: 12px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎬 Booking Confirmed!</h1>
            <p>Thank you for your purchase</p>
        </div>
        <div class="content">
            <p>Hi {{.UserName}},</p>
            <p>Your booking has been confirmed! Here are your ticket details:</p>
            
            <div class="booking-details">
                <div class="detail-row">
                    <span class="detail-label">Booking ID:&nbsp;</span><span class="detail-value">{{.BookingID}}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Movie:&nbsp;</span><span class="detail-value">{{.MovieTitle}}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Theater:&nbsp;</span><span class="detail-value">{{.Theater}}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Date:&nbsp;</span><span class="detail-value">{{.BookingDate}}</span>
                </div>
            </div>
            
            <div class="seats">
                <strong>Your Seats:</strong><br>
                <span style="font-size: 24px;">{{range $i, $seat := .Seats}}{{if $i}}, {{end}}{{$seat}}{{end}}</span>
            </div>
            
            <div class="total">
                Total: ฿{{printf "%.2f" .TotalAmount}}
            </div>
            
            <p>Please arrive 15 minutes before showtime. Show this email or your booking ID at the counter.</p>
            
            <div class="footer">
                <p>Cinema Booking System</p>
                <p>This is an automated email. Please do not reply.</p>
            </div>
        </div>
    </div>
</body>
</html>
`

	tmpl, err := template.New("booking").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	return s.sendWithTLS(to, subject, body.String())
}

func (s *EmailService) sendWithTLS(to, subject, htmlBody string) error {
	fromAddr := s.from
	if fromAddr == "" {
		fromAddr = s.username
	}

	headers := make(map[string]string)
	headers["From"] = fromAddr
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var message bytes.Buffer
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(htmlBody)

	addr := s.host + ":" + s.port
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	tlsConfig := &tls.Config{
		ServerName: s.host,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	if err := client.Mail(s.username); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}
	if _, err := w.Write(message.Bytes()); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("failed to quit: %w", err)
	}

	log.Printf("📧 Booking confirmation email sent to: %s", to)
	return nil
}
