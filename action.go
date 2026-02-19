package tracks

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/tmeire/tracks/i18n"
	"github.com/tmeire/tracks/session"
)

type Action struct {
	// Method is the uppercase HTTP verb for this action
	Method string

	// Path is the url path where this action is served
	Path string

	// Controller is the name of the controller this action is part of
	Controller string

	// Name is the name of the action
	Name string

	// Func is the action that will be executed when this endpoint is invoked
	Func ActionFunc

	// Layout is the name of the base layout for this response
	Layout string

	// Middlewares is a list of middlewares that need to be applied to this action only
	Middlewares []MiddlewareBuilder
}

// ActionFunc is a function that processes an HTTP request and returns either:
// 1. An opaque data object or a Response object with status and message, and
// 2. An error object that satisfies the Go error interface
//
// If the first return value is an opaque data object (not a Response), the status will be set to OK.
type ActionFunc func(r *http.Request) (any, error)

func (a ActionFunc) wrap(controllerName, actionName string, tpl Template, translator *i18n.Translator) *action {
	return &action{
		name:       controllerName + "#" + actionName,
		template:   tpl,
		impl:       a,
		translator: translator,
	}
}

type action struct {
	name       string
	template   Template
	impl       ActionFunc
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
		if appErr, ok := err.(AppError); ok {
			resp = &Response{
				StatusCode: appErr.StatusCode,
				Data: ErrorData{
					Success: false,
					Message: appErr.Message,
					Code:    appErr.Code,
				},
			}
		} else if appErrPtr, ok := err.(*AppError); ok {
			resp = &Response{
				StatusCode: appErrPtr.StatusCode,
				Data: ErrorData{
					Success: false,
					Message: appErrPtr.Message,
					Code:    appErrPtr.Code,
				},
			}
		} else {
			resp = &Response{
				StatusCode: http.StatusInternalServerError,
				Data: ErrorData{
					Success: false,
					Message: err.Error(),
					Code:    "INTERNAL_SERVER_ERROR",
				},
			}
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
	// Set any cookies provided in the response
	for _, cookie := range resp.Cookies {
		http.SetCookie(w, cookie)
	}

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
	slog.Warn("failed to render response", "error", err)

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
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(r.Context(), "action.renderhtml")
	defer span.End()

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

	if resp.StatusCode != http.StatusOK {
		err := a.template.ExecuteTemplate(w, strconv.Itoa(resp.StatusCode), resp)
		if err != nil {
			span.RecordError(err)
			// If status-specific template fails, try a generic error template if it exists
			// or just return the error to fallback to JSON
			return err
		}
		return nil
	}

	if a.template == nil {
		err := fmt.Errorf("template not found")
		span.RecordError(err)
		return err
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(resp.StatusCode)

	lang := i18n.LanguageFromContext(ctx)

	vars := ViewVars(ctx)

	// Resolve the correct template (potentially domain-specific)
	var tpl *template.Template
	if dt, ok := a.template.(*dynamicTemplate); ok {
		tpl = dt.resolve(r)
	} else {
		var err error
		tpl, err = a.template.Clone()
		if err != nil {
			return err
		}
	}

	tpl.Funcs(template.FuncMap{
		"t": func(key string, args ...interface{}) string {
			if len(args) == 0 {
				return a.translator.Translate(lang, key)
			}
			return a.translator.TranslateWithParams(lang, key, args...)
		},
		"v": func(key string) any {
			return vars[key]
		},
		"csrf_token": func() string {
			return CSRFTokenFromContext(r)
		},
		"csrf_field": func() template.HTML {
			return CSRFField(r)
		},
	})

	// TODO: Write to a buffer and only write to the response on success
	sessionData := session.FromRequest(r)
	var flash map[string]string
	if sessionData != nil {
		flash = session.FlashMessages(r)
	}

	return tpl.ExecuteTemplate(w, "page", struct {
		Title   string
		Session session.Session
		Flash   map[string]string
		Content any
		Vars    map[string]any
	}{
		Title:   resp.Title,
		Session: sessionData,
		Flash:   flash,
		Content: resp.Data,
		Vars:    vars,
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
