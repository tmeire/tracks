package tracks

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"html/template"
	"net/http"
	"strings"

	"github.com/tmeire/tracks/i18n"
	"github.com/tmeire/tracks/session"
)

// Action is a function that processes an HTTP request and returns either:
// 1. An opaque data object or a Response object with status and message, and
// 2. An error object that satisfies the Go error interface
//
// If the first return value is an opaque data object (not a Response), the status will be set to OK.
type Action func(r *http.Request) (any, error)

func (a Action) wrap(controllerName, actionName string, tpl *template.Template, translator *i18n.Translator) *action {
	return &action{
		name:       controllerName + "#" + actionName,
		template:   tpl,
		impl:       a,
		translator: translator,
	}
}

type action struct {
	name       string
	template   *template.Template
	impl       Action
	translator *i18n.Translator
}

func (a *action) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(r.Context(), a.name)
	defer span.End()

	r = r.WithContext(ctx)

	data, err := a.impl(r)

	var resp *Response

	// If there's an error, create an error response
	if err != nil {
		panic(err)
		resp = &Response{
			StatusCode: http.StatusInternalServerError,
			Data: map[string]string{
				"message": err.Error(),
			},
		}
	} else if response, ok := data.(*Response); ok {
		// If data is already a Response, use it directly
		resp = response
	} else {
		// If data is an opaque object, create a Response with StatusOK
		resp = &Response{
			StatusCode: http.StatusOK,
			Data:       data,
		}
	}

	a.write(w, r, resp)
}

// WriteResponse writes the response to the http.ResponseWriter based on the
// content type requested by the client
func (a *action) write(w http.ResponseWriter, r *http.Request, resp *Response) {
	// Set the default status code if not provided
	if resp.StatusCode == 0 {
		resp.StatusCode = http.StatusOK
	}

	// Determine content type based on Accept header
	contentTypes := determineContentType(r)

	var render renderer
	for _, contentType := range contentTypes {
		switch contentType {
		case "application/json":
			render = a.renderJSON
		case "application/xml":
			render = a.renderXML
		case "text/html":
			render = a.renderHTML
		case "text/plain":
			render = a.renderText
		case "*/*":
			render = a.renderHTML
		}
		if render != nil {
			break
		}
	}

	if render == nil {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	err := render(r, w, resp)
	if err == nil {
		return
	}

	trace.SpanFromContext(r.Context()).RecordError(err)

	// If template rendering fails, fallback to JSON
	w.Header().Set("Content-Type", "application/json")
	jsonErr := json.NewEncoder(w).Encode(err)
	if jsonErr != nil {
		w.Write([]byte("Error rendering template and marshaling JSON"))
		return
	}
}

type renderer func(r *http.Request, w http.ResponseWriter, resp *Response) error

// renderHTML renders an HTML template with the given data
func (a *action) renderHTML(r *http.Request, w http.ResponseWriter, resp *Response) error {
	if resp.Location != "" {
		// TODO: Not really a fan of hardcoding support for HTMX in here. This feels like we need some kind of hook
		// system here so we can also support libraries like Turbo JS.
		if r.Header.Get("hx-request") == "true" {
			w.Header().Set("HX-Redirect", resp.Location)
			w.WriteHeader(http.StatusAccepted)
			return nil
		} else {
			w.Header().Set("Location", resp.Location)
			w.WriteHeader(http.StatusSeeOther)
			return nil
		}
	}

	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(r.Context(), "action.renderhtml")
	defer span.End()

	if a.template == nil {
		err := fmt.Errorf("template not found")
		span.RecordError(err)
		return err
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(resp.StatusCode)

	lang := i18n.LanguageFromContext(ctx)

	tpl, err := a.template.Clone()
	if err != nil {
		return err
	}
	tpl.Funcs(template.FuncMap{
		"t": func(key string, args ...interface{}) string {
			if len(args) == 0 {
				return a.translator.Translate(lang, key)
			}
			return a.translator.TranslateWithParams(lang, key, args...)
		},
	})

	// TODO: Write to a buffer and only write to the response on success
	return tpl.ExecuteTemplate(w, "application.gohtml", struct {
		Title   string
		Session session.Session
		Flash   map[string]string
		Content any
	}{
		Title:   resp.Title,
		Session: session.FromRequest(r),
		Flash:   session.FlashMessages(r),
		Content: resp.Data,
	})
}

func (a *action) renderJSON(r *http.Request, w http.ResponseWriter, resp *Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)

	return json.NewEncoder(w).Encode(resp.Data)
}

func (a *action) renderXML(r *http.Request, w http.ResponseWriter, resp *Response) error {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(resp.StatusCode)

	return xml.NewEncoder(w).Encode(resp.Data)
}

func (a *action) renderText(r *http.Request, w http.ResponseWriter, resp *Response) error {
	// For plain text, try to convert to string if possible
	if str, ok := resp.Data.(string); ok {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(resp.StatusCode)

		_, err := w.Write([]byte(str))
		return err
	} else if bs, ok := resp.Data.([]byte); ok {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(resp.StatusCode)

		_, err := w.Write(bs)
		return err
	}

	// Fallback to JSON for complex objects but keep the text/plain content type
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)

	return json.NewEncoder(w).Encode(resp.Data)
}

// determineContentType determines the content type based on the file extension in the URL path.
// It returns one of: "application/json", "application/xml", "text/html", or "text/plain" (default).
// The function examines the file extension in the URL path and maps it to the appropriate MIME type.
// If no recognized extension is found, it defaults to "text/plain".
func determineContentType(r *http.Request) []string {
	accept := strings.TrimSpace(r.Header.Get("Accept"))
	if accept == "" {
		return []string{"text/html"}
	}

	var res []string
	for _, t := range strings.Split(accept, ",") {
		res = append(res, strings.TrimSpace(strings.Split(t, ";")[0]))
	}
	return res
}
