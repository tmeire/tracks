package mail_test

import (
	"context"
	"testing"

	"github.com/tmeire/tracks/modules/mail"
)

type TestMailer struct {
	mail.Mailer
}

func (m *TestMailer) Welcome(name string) *mail.Message {
	return m.Mail(mail.Options{
		To:      []string{"user@example.com"},
		Subject: "Welcome, " + name,
		// We won't use a real template in this unit test to avoid filesystem dependencies
		// but we can check if the Message is constructed correctly.
	})
}

func TestMailerConstruction(t *testing.T) {
	tm := &TestMailer{
		Mailer: mail.NewMailer(),
	}
	tm.From = "system@example.com"

	msg := tm.Welcome("Thomas")

	if msg.Subject != "Welcome, Thomas" {
		t.Errorf("Expected subject 'Welcome, Thomas', got '%s'", msg.Subject)
	}

	if msg.From != "system@example.com" {
		t.Errorf("Expected from 'system@example.com', got '%s'", msg.From)
	}

	if len(msg.To) != 1 || msg.To[0] != "user@example.com" {
		t.Errorf("Expected to 'user@example.com', got %v", msg.To)
	}
}

func TestDeliverNow(t *testing.T) {
	tm := &TestMailer{
		Mailer: mail.NewMailer(),
	}
	
	msg := tm.Welcome("Thomas")
	err := msg.DeliverNow(context.Background())
	if err != nil {
		t.Errorf("DeliverNow failed: %v", err)
	}
}
