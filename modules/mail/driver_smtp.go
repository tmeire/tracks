package mail

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

func init() {
	RegisterDriver("smtp", func(conf json.RawMessage) (Driver, error) {
		var c Config
		if err := json.Unmarshal(conf, &c); err != nil {
			return nil, err
		}
		return NewSMTPDriver(c.SMTP), nil
	})
}

type SMTPDriver struct {
	config SMTPConfig
}

func NewSMTPDriver(config SMTPConfig) *SMTPDriver {
	return &SMTPDriver{config: config}
}

func (d *SMTPDriver) Send(ctx context.Context, msg *Message) error {
	addr := fmt.Sprintf("%s:%d", d.config.Address, d.config.Port)
	slog.Info("sending email via SMTP", "address", addr, "from", msg.From, "to", msg.To, "subject", msg.Subject)
	
	// Prepare headers
	header := make(map[string]string)
	header["From"] = msg.From
	header["To"] = strings.Join(msg.To, ", ")
	header["Subject"] = msg.Subject
	header["MIME-Version"] = "1.0"
	
	var body string
	if msg.HTMLBody != "" && msg.TextBody != "" {
		// Multi-part mixed/alternative would be better here, 
		// but let's start simple with just HTML if available, else Text.
		header["Content-Type"] = "text/html; charset=\"UTF-8\""
		body = msg.HTMLBody
	} else if msg.HTMLBody != "" {
		header["Content-Type"] = "text/html; charset=\"UTF-8\""
		body = msg.HTMLBody
	} else {
		header["Content-Type"] = "text/plain; charset=\"UTF-8\""
		body = msg.TextBody
	}

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	var auth smtp.Auth
	if d.config.UserName != "" {
		slog.Info("using PLAIN authentication", "user", d.config.UserName)
		auth = smtp.PlainAuth("", d.config.UserName, d.config.Password, d.config.Address)
	} else {
		slog.Info("no authentication provided")
	}
	
	err := smtp.SendMail(addr, auth, msg.From, msg.To, []byte(message))
	if err != nil {
		slog.Error("SMTP SendMail failed", "error", err)
	} else {
		slog.Info("SMTP email sent successfully")
	}
	return err
}
