package mail

import (
	"context"
	"fmt"
	"strings"
)

type LogDriver struct{}

func NewLogDriver() *LogDriver {
	return &LogDriver{}
}

func (d *LogDriver) Send(ctx context.Context, msg *Message) error {
	fmt.Println("--- EMAIL SENT (Log Driver) ---")
	fmt.Printf("From:    %s\n", msg.From)
	fmt.Printf("To:      %s\n", strings.Join(msg.To, ", "))
	if len(msg.Cc) > 0 {
		fmt.Printf("Cc:      %s\n", strings.Join(msg.Cc, ", "))
	}
	fmt.Printf("Subject: %s\n", msg.Subject)
	if msg.TextBody != "" {
		fmt.Println("\nText Body:")
		fmt.Println(msg.TextBody)
	}
	if msg.HTMLBody != "" {
		fmt.Println("\nHTML Body:")
		fmt.Println(msg.HTMLBody)
	}
	if len(msg.Attachments) > 0 {
		fmt.Printf("\nAttachments: %d files\n", len(msg.Attachments))
		for _, a := range msg.Attachments {
			fmt.Printf("- %s (%s, %d bytes)\n", a.Filename, a.ContentType, len(a.Content))
		}
	}
	fmt.Println("-------------------------------")
	return nil
}
