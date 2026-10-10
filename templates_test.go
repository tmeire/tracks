package tracks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTemplates_PartialsInSymlinkedViews loads a layout whose partial lives in a views directory
// that is reached through a symlink, as when a package's tests link to the app's views.
func TestTemplates_PartialsInSymlinkedViews(t *testing.T) {
	dir := t.TempDir()
	views := filepath.Join(dir, "views")
	files := map[string]string{
		"layouts/application.gohtml": `<main>{{ template "yield" . }}</main>{{ template "footer" . }}`,
		"layouts/_footer.gohtml":     `<footer>shared footer</footer>`,
		"home/index.gohtml":          `home page`,
	}
	for name, content := range files {
		path := filepath.Join(views, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(dir, "linked-views")
	if err := os.Symlink(views, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	for _, basedir := range []string{views, link} {
		ts := newTemplates("localhost")
		ts.basedir = basedir
		tpl, err := ts.Load("application", "home", "index")
		if err != nil || tpl == nil {
			t.Fatalf("%s: load: %v (template %v)", basedir, err, tpl)
		}
		var out strings.Builder
		if err := tpl.ExecuteTemplate(&out, "page", nil); err != nil {
			t.Fatalf("%s: execute: %v", basedir, err)
		}
		if want := "<main>home page</main><footer>shared footer</footer>"; out.String() != want {
			t.Errorf("%s: rendered %q, want %q", basedir, out.String(), want)
		}
	}
}
