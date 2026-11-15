package email

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/kushalktamang/blueprint-monorepo-tsgo/assets"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/config"
	"github.com/resend/resend-go/v2"
)

type Client struct {
	client      *resend.Client
	logger      *slog.Logger
	fromAddress string
}

func NewClient(cfg *config.Config, logger *slog.Logger) *Client {
	return &Client{
		client:      resend.NewClient(cfg.Integration.ResendAPIKey),
		logger:      logger,
		fromAddress: fmt.Sprintf("%s <%s>", cfg.Integration.EmailFromName, cfg.Integration.EmailFromAddress),
	}
}

func (c *Client) SendEmail(to, subject string, templateName Template, data map[string]string) error {
	tmpl, err := template.ParseFS(assets.FS, fmt.Sprintf("templates/emails/%s.html", templateName))
	if err != nil {
		return fmt.Errorf("failed to parse email template %s: %w", templateName, err)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute email template %s: %w", templateName, err)
	}

	params := &resend.SendEmailRequest{
		From:    c.fromAddress,
		To:      []string{to},
		Subject: subject,
		Html:    body.String(),
	}

	_, err = c.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
