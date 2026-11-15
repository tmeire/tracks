package mail

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"sort"
	"strings"
	"time"
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

	message := buildMessage(msg, time.Now())

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

// headerValue makes a value safe to place in a mail header: CR, LF and other
// control characters would otherwise let it end the header and start new ones
// (or the body).
func headerValue(s string) string {
	return strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)), " ")
}

// messageID returns a new, globally unique Message-ID for mail sent from
// the address from.
func messageID(from string) string {
	domain := "localhost"
	if addr, err := mail.ParseAddress(from); err == nil {
		if at := strings.LastIndexByte(addr.Address, '@'); at != -1 {
			domain = addr.Address[at+1:]
		}
	}
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "<" + hex.EncodeToString(b) + "@" + domain + ">"
}

// headerName canonicalises a header field name, or returns "" if it is not a
// valid one (RFC 5322 section 3.6.8: printable ASCII except colon).
func headerName(k string) string {
	if k == "" {
		return ""
	}
	for i := 0; i < len(k); i++ {
		if k[i] <= ' ' || k[i] >= 0x7f || k[i] == ':' {
			return ""
		}
	}
	return textproto.CanonicalMIMEHeaderKey(k)
}

// buildMessage renders msg as an RFC 5322 message: headers, blank line, body.
// Custom headers in msg.Headers are added first, so they can set Date or
// Message-ID but cannot override From, To, Subject or the content headers.
func buildMessage(msg *Message, now time.Time) string {
	header := make(map[string]string)
	for k, v := range msg.Headers {
		if k = headerName(k); k != "" {
			header[k] = v
		}
	}
	if header["Date"] == "" {
		header["Date"] = now.Format(time.RFC1123Z)
	}
	if header["Message-Id"] == "" {
		header["Message-Id"] = messageID(msg.From)
	}
	header["From"] = msg.From
	header["To"] = strings.Join(msg.To, ", ")
	header["Subject"] = msg.Subject
	header["MIME-Version"] = "1.0"

	var body string
	if msg.HTMLBody != "" {
		// Multi-part alternative would be better here when both bodies are
		// set, but for now HTML wins if available, else Text.
		header["Content-Type"] = "text/html; charset=\"UTF-8\""
		body = msg.HTMLBody
	} else {
		header["Content-Type"] = "text/plain; charset=\"UTF-8\""
		body = msg.TextBody
	}

	keys := make([]string, 0, len(header))
	for k := range header {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s: %s\r\n", k, headerValue(header[k]))
	}
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}
