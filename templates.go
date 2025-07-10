package tracks

import (
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type Templates struct {
	fns     template.FuncMap
	basedir string
	layouts map[string]*template.Template
}

// Func adds a new function to templates that are loaded after this call
func (t *Templates) Func(name string, fn any) {
	t.fns[name] = fn
	t.layouts = nil
}

func (t *Templates) loadLayout(name string) (*template.Template, error) {
	filename := fmt.Sprintf("%s.gohtml", name)

	layout, err := template.
		New(filename).
		Funcs(t.fns).
		Option("missingkey=error").
		ParseFiles(filepath.Join(t.basedir, "layouts", filename))

	if err != nil {
		return nil, err
	}

	// Add the layout with a shared name "page" to make sure we don't have to pass the name of the layout file around.
	layout, err = layout.AddParseTree("page", layout.Lookup(filename).Tree)
	if err != nil {
		return nil, err
	}

	// Find and load all partial templates into the layout
	partials, err := filepath.Glob(filepath.Join(t.basedir, "*", "_*.gohtml"))
	if err != nil {
		return nil, err
	}

	for _, partial := range partials {
		filename := filepath.Base(partial)

		partialName := strings.TrimSuffix(strings.TrimPrefix(filename, "_"), ".gohtml")
		controllerName := filepath.Base(filepath.Dir(partial))

		templateName := controllerName + "#" + partialName

		parsed, err := layout.
			New(filename).
			Funcs(t.fns).
			ParseFiles(partial)
		if err != nil {
			return nil, err
		}

		layout, err = layout.AddParseTree(templateName, parsed.Tree)
		if err != nil {
			return nil, err
		}
	}

	return layout, nil
}

// Load loads the view associated with the controller and action from the templates directory. It will load the
// base layouts from "./{{basedir}}/layouts/{{layout}}.gohtml", the view file from
// "./{{basedir}}/{{controller}}/{{action}}.gohtml" and makes sure the two are properly linked together. The resulting
// template has access to all functions that were registered before the call to Load.
//
// Not thread-safe!
func (t *Templates) Load(layout, controller, action string) (*template.Template, error) {
	if _, ok := t.layouts[layout]; !ok {
		_layout, err := t.loadLayout(layout)
		if err != nil {
			slog.Warn("failed to load layout", "name", layout, "error", err)
			// Let's ignore it, could be an API-only app
			return nil, nil
		}
		t.layouts[layout] = _layout
	}

	// Construct the template path
	filename := action + ".gohtml"
	templatePath := filepath.Join(t.basedir, controller, filename)

	// Check if the template exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, nil
	}

	page, err := t.layouts[layout].Clone()
	if err != nil {
		return nil, err
	}

	// Parse and execute the actual page template
	page, err = page.
		New(filename).
		Funcs(t.fns).
		ParseFiles(filepath.Join(t.basedir, controller, filename))
	if err != nil {
		return nil, err
	}

	// Add the same template again, but now with the name "yield" to make sure it can be called from the application
	// Note: making the page template available to be rendered as "yield" in the layout template can be achieved in
	// a couple of ways.
	// * At the moment, we're 'renaming' the page template which may come at a bit of a memory cost.
	// * We could also add a dynamic template that is just `{{ template 'action.gohtml' . }}`, which may come with a
	//   little bit of extra runtime cost.
	// * We could add a template function 'yield' which calls `page.ExecuteTemplate()`, which may also be a bit less
	//   efficient at runtime wrt memory buffers etc.
	// TODO: Try to properly evaluate the options above. This is good enough for now.
	return page.AddParseTree("yield", page.Tree)
}
