package mail

import (
	"strings"
	"testing"
	"time"
)

func TestBuildMessageStripsHeaderInjection(t *testing.T) {
	msg := &Message{
		From:     "noreply@example.com",
		To:       []string{"a@example.com (x\r\nBcc: evil@example.com)"},
		Subject:  "Hello\r\nBcc: evil@example.com\n\r\n<script>",
		HTMLBody: "<p>body</p>",
	}
	out := buildMessage(msg, time.Now())

	head, body, ok := strings.Cut(out, "\r\n\r\n")
	if !ok {
		t.Fatalf("no header/body separator in %q", out)
	}
	if body != "<p>body</p>" {
		t.Errorf("body = %q", body)
	}
	for _, line := range strings.Split(head, "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), "bcc:") {
			t.Errorf("injected header line %q", line)
		}
	}
	if !strings.Contains(head, "Subject: Hello Bcc: evil@example.com <script>\r\n") {
		t.Errorf("subject not flattened onto one line:\n%s", head)
	}
	if strings.ContainsAny(strings.ReplaceAll(head, "\r\n", ""), "\r\n") {
		t.Errorf("stray CR/LF in headers:\n%q", head)
	}
}

func TestBuildMessageAddsDateMessageIDAndCustomHeaders(t *testing.T) {
	now := time.Date(2026, 11, 15, 10, 30, 0, 0, time.UTC)
	msg := &Message{
		From:    "FloraLynx <noreply@floralynx.com>",
		To:      []string{"a@example.com"},
		Subject: "Hi",
		Headers: map[string]string{
			"list-unsubscribe":      "<https://floralynx.com/u?x=1>",
			"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
			"Subject":               "overridden?",
			"Bad Name":              "dropped",
			"X-Evil\r\nBcc":         "dropped",
		},
		TextBody: "body",
	}
	head, _, _ := strings.Cut(buildMessage(msg, now), "\r\n\r\n")
	lines := strings.Split(head, "\r\n")
	has := func(prefix string) bool {
		for _, l := range lines {
			if strings.HasPrefix(l, prefix) {
				return true
			}
		}
		return false
	}
	for _, want := range []string{
		"Date: Sun, 15 Nov 2026 10:30:00 +0000",
		"Message-Id: <",
		"List-Unsubscribe: <https://floralynx.com/u?x=1>",
		"List-Unsubscribe-Post: List-Unsubscribe=One-Click",
		"Subject: Hi",
	} {
		if !has(want) {
			t.Errorf("missing %q in:\n%s", want, head)
		}
	}
	if !strings.Contains(head, "@floralynx.com>") {
		t.Errorf("Message-Id not on sender's domain:\n%s", head)
	}
	if has("Bad Name") || has("X-Evil") || has("Bcc") || strings.Contains(head, "overridden") {
		t.Errorf("invalid or reserved custom header leaked:\n%s", head)
	}
	if a, b := messageID(msg.From), messageID(msg.From); a == b {
		t.Errorf("Message-Ids not unique: %s", a)
	}
}
