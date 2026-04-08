# Proposal: Mail Module (ActionMailer for Tracks)

This document outlines the plan for a new `mail` module in the Tracks framework, inspired by Ruby on Rails' ActionMailer.

## Goals

- Provide a structured way to define and send emails.
- Support multiple delivery backends (SMTP, SendGrid, Log, Test).
- Integrate with the existing Tracks templating system.
- Support both HTML and Text email versions.
- Easy testing of sent emails.

## 1. Directory Structure

```text
modules/mail/
├── drivers/             # Delivery backend implementations
│   ├── smtp.go         # Standard SMTP delivery
│   ├── log.go          # Logs emails to stdout (dev)
│   ├── test.go         # Collects emails in memory (testing)
│   └── file.go         # Saves emails to .eml files (dev)
├── driver.go            # Driver interface definition
├── mail.go              # Message and Attachment structs
├── mailer.go            # Base Mailer struct and logic
├── module.go            # Module registration and global state
└── config.go            # Configuration structures
```

## 2. Core Components

### `Message` Struct
Represents an email message.

```go
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
}
```

### `Driver` Interface
Abstraction for email delivery.

```go
type Driver interface {
    Send(ctx context.Context, msg *Message) error
}
```

### `Mailer` Base Struct
Users will embed this in their own mailer structs to gain email functionality.

```go
type Mailer struct {
    // Reference to the global mail driver
    driver Driver
    // Default settings
    From string
    // Template helper
    templates *tracks.Templates
}
```

## 3. Usage Example

### Defining a Mailer

```go
type UserMailer struct {
    mail.Mailer
}

func (m *UserMailer) Welcome(user *models.User) *mail.Message {
    return m.Mail(mail.Options{
        To:      []string{user.Email},
        Subject: "Welcome to Tracks!",
        Template: "user_mailer/welcome",
        Data: map[string]any{
            "User": user,
        },
    })
}
```

### Sending an Email

```go
// Immediate delivery
err := mailers.User.Welcome(user).DeliverNow(ctx)

// Asynchronous delivery (initially using goroutines)
mailers.User.Welcome(user).DeliverLater(ctx)
```

## 4. Templating

Mailers will look for templates in `./views/mailers/`.
A mailer named `UserMailer` with an action `Welcome` will look for:
- `./views/mailers/user_mailer/welcome.html.gohtml`
- `./views/mailers/user_mailer/welcome.text.gohtml` (optional)

## 5. Configuration

Configured in `config.json`:

```json
{
  "mail": {
    "delivery_method": "smtp",
    "smtp_settings": {
      "address": "smtp.gmail.com",
      "port": 587,
      "user_name": "...",
      "password": "...",
      "authentication": "plain",
      "enable_starttls_auto": true
    },
    "defaults": {
      "from": "noreply@example.com"
    }
  }
}
```

## 6. Testing

The `test` driver will provide a way to assert that emails were sent.

```go
func TestUserSignup(t *testing.T) {
    // ... signup logic ...
    
    mail.AssertSent(t, 1) // Assert 1 email was sent
    lastEmail := mail.LastSent()
    assert.Equal(t, "Welcome to Tracks!", lastEmail.Subject)
}
```

## 7. Implementation Plan

1.  **Phase 1: Core & Drivers**: Implement `Message`, `Driver` interface, `Log` and `Test` drivers.
2.  **Phase 2: Mailer & Templating**: Implement base `Mailer` and integration with `tracks.Templates`.
3.  **Phase 3: SMTP Driver**: Implement the SMTP delivery backend.
4.  **Phase 4: CLI Integration**: Add `tracks generate mailer Name` to the CLI.
5.  **Phase 5: Previews (Future)**: Add a development route to preview mailer templates in the browser.
