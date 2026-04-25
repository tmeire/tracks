package mail

import (
	"context"
	"log"
)

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// Message represents an email message to be sent
type Message struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	HTMLBody    string
	TextBody    string
	Attachments []Attachment
	Headers     map[string]string

	// Internal reference to the driver for delivery
	driver Driver
}

// DeliverNow sends the email immediately using the configured driver
func (m *Message) DeliverNow(ctx context.Context) error {
	if m.driver == nil {
		// Fallback or error? For now, let's assume it should have been set
		return nil
	}
	err := m.driver.Send(ctx, m)
	if err != nil {
		log.Printf("Failed to deliver email: %v", err)
	}
	return err
}

// DeliverLater sends the email asynchronously (currently using a goroutine)
func (m *Message) DeliverLater(ctx context.Context) {
	go func() {
		_ = m.DeliverNow(context.Background())
	}()
}
