package mail

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/tmeire/tracks"
)

// Mailer is the base struct that users embed in their mailers
type Mailer struct {
	driver    Driver
	templates *tracks.Templates
	From      string
}

// Options contains the settings for a single email
type Options struct {
	To       []string
	Cc       []string
	Bcc      []string
	Subject  string
	From     string
	Template string // e.g. "user_mailer/welcome"
	Layout   string // defaults to "mailer"
	Data     any
}

// Mail prepares a Message for delivery
func (m *Mailer) Mail(opt Options) *Message {
	from := opt.From
	if from == "" {
		from = m.From
	}

	msg := &Message{
		driver:  m.driver,
		From:    from,
		To:      opt.To,
		Cc:      opt.Cc,
		Bcc:     opt.Bcc,
		Subject: opt.Subject,
	}

	if opt.Template != "" {
		layout := opt.Layout
		if layout == "" {
			layout = "mailer"
		}

		// Split template into controller/action (e.g. "user_mailer/welcome")
		// We prepend "mailers/" to keep them separate from web views
		controller := "mailers/" + opt.Template
		action := ""
		if lastSlash := bytes.LastIndexByte([]byte(controller), '/'); lastSlash != -1 {
			action = controller[lastSlash+1:]
			controller = controller[:lastSlash]
		}

		tpl, err := m.templates.Load(layout, controller, action)
		if err != nil {
			// Log error? For now we'll just have an empty body
			fmt.Printf("Error loading mail template %s: %v\n", opt.Template, err)
		} else if tpl != nil {
			var buf bytes.Buffer
			err := tpl.ExecuteTemplate(&buf, "page", opt.Data)
			if err != nil {
				fmt.Printf("Error executing mail template %s: %v\n", opt.Template, err)
			} else {
				msg.HTMLBody = buf.String()
			}
		}
	}

	return msg
}

// FuncMap returns the template functions available to mailers
func (m *Mailer) FuncMap() template.FuncMap {
	return template.FuncMap{}
}
