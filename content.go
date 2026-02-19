package tracks

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// Content represents the common metadata for file-based content.
type Content struct {
	Slug string        `yaml:"slug"`
	Body template.HTML `yaml:"-"`
}

// ContentConfig defines the configuration for a content resource.
type ContentConfig struct {
	Layout     string
	ListLayout string
	SortBy     string
	SortDesc   bool
}

type ContentController[T any] struct {
	dir    string
	config ContentConfig
}

func NewContentController[T any](dir string, config ContentConfig) *ContentController[T] {
	return &ContentController[T]{
		dir:    dir,
		config: config,
	}
}

func (c *ContentController[T]) Register(r Router, path string) Router {
	if c.config.Layout == "" {
		c.config.Layout = "content/show"
	}
	if c.config.ListLayout == "" {
		c.config.ListLayout = "content/index"
	}

	return r.GetFunc(path, "content", "index", c.Index).
		GetFunc(path+"/{slug}", "content", "show", c.Show)
}

func (c *ContentController[T]) Index(r *http.Request) (any, error) {
	items, err := c.loadAll()
	if err != nil {
		return nil, err
	}

	// TODO: Sorting based on config.SortBy
	
	return &Response{
		StatusCode: http.StatusOK,
		Data:       items,
	}, nil
}

func (c *ContentController[T]) Show(r *http.Request) (any, error) {
	slug := r.PathValue("slug")
	if slug == "" {
		return nil, fmt.Errorf("missing slug")
	}

	item, err := c.loadBySlug(slug)
	if err != nil {
		return nil, err
	}

	return &Response{
		StatusCode: http.StatusOK,
		Data:       item,
	}, nil
}

func (c *ContentController[T]) loadAll() ([]T, error) {
	var items []T
	err := filepath.WalkDir(c.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") && !strings.HasSuffix(d.Name(), ".html") {
			return nil
		}

		item, err := c.loadFile(path)
		if err != nil {
			return err
		}
		items = append(items, item)
		return nil
	})

	return items, err
}

func (c *ContentController[T]) loadBySlug(slug string) (T, error) {
	var zero T
	items, err := c.loadAll()
	if err != nil {
		return zero, err
	}

	for _, item := range items {
		// Use reflection to check slug
		v := reflect.ValueOf(item)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		
		s := v.FieldByName("Slug").String()
		if s == slug {
			return item, nil
		}
	}

	return zero, fmt.Errorf("content not found: %s", slug)
}

func (c *ContentController[T]) loadFile(path string) (T, error) {
	var item T
	data, err := os.ReadFile(path)
	if err != nil {
		return item, err
	}

	// Simple frontmatter parsing (separated by ---)
	parts := bytes.SplitN(data, []byte("---\n"), 3)
	var content []byte
	if len(parts) == 3 {
		if err := yaml.Unmarshal(parts[1], &item); err != nil {
			return item, fmt.Errorf("failed to parse frontmatter in %s: %w", path, err)
		}
		content = parts[2]
	} else {
		content = data
	}

	// Set slug if empty
	v := reflect.ValueOf(&item).Elem()
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	slugField := v.FieldByName("Slug")
	if slugField.IsValid() && slugField.String() == "" {
		slug := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		slugField.SetString(slug)
	}

	// Set content body
	bodyField := v.FieldByName("Body")
	if bodyField.IsValid() {
		bodyField.Set(reflect.ValueOf(template.HTML(content)))
	}

	return item, nil
}
