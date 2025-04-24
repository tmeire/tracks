package tracks

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type action struct {
	template *template.Template
	impl     Action
}

func wrap(controllerName, actionName string, a Action) *action {
	return &action{
		template: template.Must(load(controllerName, actionName)),
		impl:     a,
	}
}

func load(controller, action string) (*template.Template, error) {
	// Construct the template path
	templatePath := strings.ToLower(filepath.Join("views", controller, action+".gohtml"))

	// Check if template exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, nil
	}

	layout, err := template.ParseFiles("./views/layouts/application.gohtml")
	if err != nil {
		return nil, err
	}

	// Parse and execute template
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, err
	} else {
		_, err = layout.AddParseTree("yield", tmpl.Tree)
		if err != nil {
			return nil, err
		}
	}
	return layout, nil
}

func (a *action) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := a.impl(r)

	var resp *Response

	// If there's an error, create an error response
	if err != nil {
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
	contentType := determineContentType(r)

	var render renderer
	switch contentType {
	case "application/json":
		render = a.renderJSON
	case "application/xml":
		render = a.renderXML
	case "text/html":
		render = a.renderHTML
	case "text/plain":
		render = a.renderText
	}

	err := render(w, resp)
	if err == nil {
		return
	}

	// If template rendering fails, fallback to JSON
	w.Header().Set("Content-Type", "application/json")
	jsonErr := json.NewEncoder(w).Encode(err)
	if jsonErr != nil {
		w.Write([]byte("Error rendering template and marshaling JSON"))
		return
	}
}

type renderer func(w http.ResponseWriter, resp *Response) error

// renderHTML renders an HTML template with the given data
func (a *action) renderHTML(w http.ResponseWriter, resp *Response) error {
	if resp.Location != "" && (resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNoContent) {
		w.Header().Set("Location", resp.Location)
		w.WriteHeader(http.StatusSeeOther)
		return nil
	}
	if a.template == nil {
		return fmt.Errorf("template not found")
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(resp.StatusCode)

	return a.template.ExecuteTemplate(w, "application.gohtml", struct {
		PageTitle string
		Content   any
	}{
		PageTitle: "",
		Content:   resp.Data,
	})
}

func (a *action) renderJSON(w http.ResponseWriter, resp *Response) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)

	return json.NewEncoder(w).Encode(resp.Data)
}

func (a *action) renderXML(w http.ResponseWriter, resp *Response) error {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(resp.StatusCode)

	return xml.NewEncoder(w).Encode(resp.Data)
}

func (a *action) renderText(w http.ResponseWriter, resp *Response) error {
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)

	return json.NewEncoder(w).Encode(resp.Data)
}

// determineContentType determines the content type based on the file extension in the URL path.
// It returns one of: "application/json", "application/xml", "text/html", or "text/plain" (default).
// The function examines the file extension in the URL path and maps it to the appropriate MIME type.
// If no recognized extension is found, it defaults to "text/plain".
func determineContentType(r *http.Request) string {
	if dot := strings.LastIndex(r.URL.Path, "."); dot >= 0 {
		switch strings.ToLower(r.URL.Path[dot+1:]) {
		case "json":
			return "application/json"
		case "xml":
			return "application/xml"
		case "html", "htm":
			return "text/html"
		case "txt":
			return "text/plain"
		}
	}
	// if it has the JSON header, return JSON
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		return "application/json"
	}
	return "text/html"
}
