package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"

	"car-rental-system/internal/models"
)

type EmailSender interface {
	SendVerificationCode(ctx context.Context, toEmail, toName, code string) error
	SendRentalStatusUpdate(ctx context.Context, toEmail, toName string, rental *models.Rental) error
}

type EmailSenderConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	FromName string
	UseTLS   bool
}

type SMTPEmailSender struct {
	cfg    EmailSenderConfig
	logger *slog.Logger
}

type LoggingEmailSender struct {
	logger *slog.Logger
}

func NewEmailSender(cfg EmailSenderConfig, logger *slog.Logger) EmailSender {
	if logger == nil {
		logger = slog.Default()
	}
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Port = strings.TrimSpace(cfg.Port)
	cfg.From = strings.TrimSpace(cfg.From)
	cfg.FromName = strings.TrimSpace(cfg.FromName)
	cfg.Username = strings.TrimSpace(cfg.Username)
	if cfg.Port == "" {
		cfg.Port = "587"
	}
	if cfg.From == "" {
		cfg.From = "no-reply@rentcar.local"
	}
	if cfg.FromName == "" {
		cfg.FromName = "RentCar"
	}
	if cfg.Host == "" {
		return &LoggingEmailSender{logger: logger}
	}

	return &SMTPEmailSender{cfg: cfg, logger: logger}
}

func (s *SMTPEmailSender) SendVerificationCode(ctx context.Context, toEmail, toName, code string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	message := verificationEmailMessage(s.cfg.From, s.cfg.FromName, toEmail, toName, code)
	if err := s.sendMail(ctx, toEmail, []byte(message)); err != nil {
		return err
	}

	s.logger.Info("verification_email_sent", slog.String("email", toEmail))
	return nil
}

func (s *SMTPEmailSender) SendRentalStatusUpdate(ctx context.Context, toEmail, toName string, rental *models.Rental) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	message := rentalStatusEmailMessage(s.cfg.From, s.cfg.FromName, toEmail, toName, rental)
	if err := s.sendMail(ctx, toEmail, []byte(message)); err != nil {
		return err
	}

	status := ""
	rentalID := int64(0)
	if rental != nil {
		status = string(rental.Status)
		rentalID = rental.ID
	}
	s.logger.Info("rental_status_email_sent", slog.String("email", toEmail), slog.Int64("rental_id", rentalID), slog.String("status", status))
	return nil
}

func (s *SMTPEmailSender) sendMail(ctx context.Context, toEmail string, message []byte) error {
	client, err := s.smtpClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	if s.cfg.Username != "" || s.cfg.Password != "" {
		if err := client.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			return err
		}
	}
	if err := client.Mail(s.cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(toEmail); err != nil {
		return err
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func (s *SMTPEmailSender) smtpClient(ctx context.Context) (*smtp.Client, error) {
	addr := net.JoinHostPort(s.cfg.Host, s.cfg.Port)
	tlsConfig := &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	if s.cfg.UseTLS || s.cfg.Port == "465" {
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if err != nil {
			return nil, err
		}
		return smtp.NewClient(conn, s.cfg.Host)
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(tlsConfig); err != nil {
			_ = client.Close()
			return nil, err
		}
	}

	return client, nil
}

func (s *LoggingEmailSender) SendVerificationCode(ctx context.Context, toEmail, toName, code string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger := s.logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("verification_email_code", slog.String("email", toEmail), slog.String("code", code))
	return nil
}

func (s *LoggingEmailSender) SendRentalStatusUpdate(ctx context.Context, toEmail, toName string, rental *models.Rental) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger := s.logger
	if logger == nil {
		logger = slog.Default()
	}

	status := ""
	rentalID := int64(0)
	if rental != nil {
		status = string(rental.Status)
		rentalID = rental.ID
	}
	logger.Info("rental_status_email", slog.String("email", toEmail), slog.Int64("rental_id", rentalID), slog.String("status", status))
	return nil
}

func verificationEmailMessage(from, fromName, toEmail, toName, code string) string {
	if strings.TrimSpace(toName) == "" {
		toName = "there"
	}

	body := fmt.Sprintf(
		"Hi %s,\n\nYour RentCar verification code is: %s\n\nThis code expires soon. If you did not create an account, you can ignore this email.\n",
		toName,
		code,
	)

	headers := map[string]string{
		"From":         fmt.Sprintf("%s <%s>", fromName, from),
		"To":           toEmail,
		"Subject":      "Your RentCar verification code",
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}

	var builder strings.Builder
	for key, value := range headers {
		builder.WriteString(key)
		builder.WriteString(": ")
		builder.WriteString(value)
		builder.WriteString("\r\n")
	}
	builder.WriteString("\r\n")
	builder.WriteString(body)
	return builder.String()
}

func rentalStatusEmailMessage(from, fromName, toEmail, toName string, rental *models.Rental) string {
	if strings.TrimSpace(toName) == "" {
		toName = "there"
	}

	subject := "Your RentCar booking was updated"
	statusText := "updated"
	nextStep := "Please open your RentCar dashboard to view the latest booking details."
	if rental != nil {
		switch rental.Status {
		case models.RentalStatusApproved:
			subject = "Your RentCar booking was approved"
			statusText = "approved"
			nextStep = "You can now open your dashboard and create a payment request."
		case models.RentalStatusRejected:
			subject = "Your RentCar booking was rejected"
			statusText = "rejected"
			nextStep = "You can choose another car or select different rental dates."
		case models.RentalStatusCancelled:
			subject = "Your RentCar booking was cancelled"
			statusText = "cancelled"
			nextStep = "If you have questions about this change, please contact the administrator."
		}
	}

	rentalID := int64(0)
	dateRange := ""
	carLabel := "your selected car"
	if rental != nil {
		rentalID = rental.ID
		dateRange = fmt.Sprintf(" from %s to %s", rental.StartDate.Format(time.DateOnly), rental.EndDate.Format(time.DateOnly))
		if rental.Car != nil {
			carLabel = strings.TrimSpace(fmt.Sprintf("%s %s", rental.Car.Brand, rental.Car.Model))
			if rental.Car.PlateNumber != "" {
				carLabel += " (" + rental.Car.PlateNumber + ")"
			}
		}
	}

	body := fmt.Sprintf(
		"Hi %s,\n\nYour RentCar booking #%d for %s%s was %s by admin.\n\n%s\n\nThank you for using RentCar.\n",
		toName,
		rentalID,
		carLabel,
		dateRange,
		statusText,
		nextStep,
	)

	headers := map[string]string{
		"From":         fmt.Sprintf("%s <%s>", fromName, from),
		"To":           toEmail,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}

	var builder strings.Builder
	for key, value := range headers {
		builder.WriteString(key)
		builder.WriteString(": ")
		builder.WriteString(value)
		builder.WriteString("\r\n")
	}
	builder.WriteString("\r\n")
	builder.WriteString(body)
	return builder.String()
}
