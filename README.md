# Tracks

**Tracks** is a full-featured, batteries-included web framework for Go (Go 1.26+) built on Go idioms, explicit typing, and zero-magic design. It combines the developer ergonomics of modern web frameworks (convention-over-configuration, layouts, partials, scaffolding) with Go's high performance, explicit error handling, and cloud-native observability.

---

## Highlights

- **Standard Library Foundation**: Built directly on Go's `net/http.ServeMux` with pattern matching (`/posts/{id}`).
- **Rich Templating Engine**: Layouts, auto-discovered nested partials (`folder#partial`), localized translations (`t`), view variables (`v`), Phosphor icons (`icon`), CSRF injection, and domain-specific template overrides.
- **Explicit Database & Generic Repositories**: Reflection-free model mapping (`database.Repository[S, T]`), query builder, lifecycle hooks (`BeforeCreate`, `AfterUpdate`, etc.), transactions (`database.WithTransaction`), atomic field updates (`AtomicUpdate`), domain-scoped models, and Goose migrations.
- **Deep OpenTelemetry Instrumentation**: Distributed tracing, metrics (including SQLite connection pool stats), and structured slog integration pre-wired out of the box via OTLP.
- **Built-in Modular Ecosystem**:
  - `authentication`: Session management, bcrypt hashing, system roles (`RequireSystemRole`), user hooks.
  - `multitenancy`: Subdomain isolation (`splitter`), central vs. tenant SQLite databases, and tenant migrations.
  - `featureflags`: Database-backed, in-memory cached, rule-targeted feature flags with CLI management.
  - `mail`: Pluggable delivery drivers (`smtp`, `log`, `testing`) with HTML template rendering.
  - `storage`: Multi-provider blob storage (`disk`, `s3`, `gcs`).
- **Core Platform Capabilities**:
  - CSRF protection (cookie, header, form field).
  - Sliding-window rate limiting.
  - In-memory tag-invalidated cache.
  - Asynchronous background job queue with retry policies.
  - Native WebSockets with rooms/hub support.
  - File-based Blog/CMS with frontmatter parsing.
  - API versioning (path, header, query parameter).
  - Request parsing & struct validation (`validate` tags).
  - Automated content negotiation (HTML, JSON, XML, text).
  - Static file asset pipeline with content-based MD5 hashing.
  - Scaffolding CLI (`tracks init`, `tracks generate`, `tracks db`, `tracks assets`).

---

## Directory Structure of a Tracks App

```text
my-app/
├── config/
│   └── config.json              # App configuration (ports, database, sessions, modules)
├── controllers/
│   ├── default.go               # Standard action controllers
│   └── posts.go                 # Resource controllers
├── models/
│   └── post.go                  # Database models implementing database.Model[S, T]
├── views/
│   ├── layouts/
│   │   └── application.gohtml   # Master layout template (contains {{template "yield" .Content}})
│   ├── default/
│   │   └── home.gohtml          # Action-specific template
│   └── posts/
│       ├── _card.gohtml         # Reusable partial
│       └── index.gohtml
├── public/
│   ├── css/
│   ├── js/
│   └── images/
├── migrations/
│   └── central/                 # Goose SQL migrations
├── main.go                      # Application bootstrapper
└── go.mod
```

---

## Quickstart

### 1. Requirements

- Go 1.26 or higher
- SQLite3 (CGO enabled for `github.com/mattn/go-sqlite3`)

### 2. Bootstrapping a Project

Create a new Tracks application:

```bash
# Using the Tracks CLI (built from ./cli)
tracks init github.com/example/myapp
cd myapp
```

Or configure manually in a few lines of Go:

```go
// main.go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/tmeire/tracks"
)

func main() {
	ctx := context.Background()

	router := tracks.New(ctx)

	// Register a simple route
	router.GetFunc("/", "default", "home", func(r *http.Request) (any, error) {
		return map[string]string{
			"message": "Welcome to Tracks!",
		}, nil
	})

	if err := router.Run(ctx); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
```

### 3. Configuration (`config/config.json`)

Tracks automatically reads `./config/config.json` (or the path defined in `TRACKS_CONFIG_FILE`):

```json
{
  "name": "myapp",
  "version": "1.0.0",
  "port": 8080,
  "development": true,
  "secure": false,
  "base_domain": "localhost:8080",
  "sessions": {
    "store": {
      "type": "db",
      "config": {
        "type": "sqlite",
        "config": {
          "path": "./data/sessions.sqlite"
        }
      }
    }
  },
  "database": {
    "type": "sqlite",
    "config": {
      "path": "./data/app.sqlite"
    }
  }
}
```

---

## Core Framework Guides

### 1. Routing & Controllers

Tracks provides both functional handlers (`ActionFunc`) and resourceful controllers (`Resource`).

#### Functional Routing

```go
router.GetFunc("/articles", "articles", "index", articlesHandler)
router.PostFunc("/articles", "articles", "create", createArticleHandler)
router.PutFunc("/articles/{id}", "articles", "update", updateArticleHandler)
router.DeleteFunc("/articles/{id}", "articles", "destroy", deleteArticleHandler)
```

#### Action Handlers

An `ActionFunc` signature receives `*http.Request` and returns `(any, error)`:

```go
func articlesHandler(r *http.Request) (any, error) {
    // Returning an object renders HTML (if views/articles/index.gohtml exists)
    // or JSON if the client requests application/json.
    return []Article{...}, nil
}
```

Redirects and custom responses use `tracks.Response`:

```go
func createArticleHandler(r *http.Request) (any, error) {
    // Process form ...
    return &tracks.Response{
        StatusCode: http.StatusSeeOther,
        Location:   "/articles",
    }, nil
}
```

Standard error helpers:
- `tracks.BadRequest(err)`
- `tracks.NotFound("item not found")`
- `tracks.Forbidden("access denied")`
- `tracks.Unauthorized("login required")`
- `tracks.Conflict("already exists")`
- `tracks.UnprocessableEntity("validation failed", validationErrors)`

#### RESTful Resources

Implement `tracks.Resource` to define standard REST endpoints:

```go
type Resource interface {
    Index(r *http.Request) (any, error)
    New(r *http.Request) (any, error)
    Create(r *http.Request) (any, error)
    Show(r *http.Request) (any, error)
    Edit(r *http.Request) (any, error)
    Update(r *http.Request) (any, error)
    Destroy(r *http.Request) (any, error)
}
```

Register on the router:

```go
router.Resource(&controllers.ArticlesResource{})
// Or mount at a custom sub-path:
router.ResourceAtPath("/admin", &controllers.ArticlesResource{})
```

This registers:
- `GET /articles/` -> `articles#index`
- `GET /articles/new` -> `articles#new`
- `POST /articles/` -> `articles#create`
- `GET /articles/{articles_id}` -> `articles#show`
- `GET /articles/{articles_id}/edit` -> `articles#edit`
- `PUT /articles/{articles_id}` -> `articles#update`
- `DELETE /articles/{articles_id}` -> `articles#destroy`

---

### 2. Views & Templating

Tracks uses Go's standard `html/template` with automatic layout inheritance and partial discovery.

#### Layout (`views/layouts/application.gohtml`)

Every page template renders inside a layout:

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{v "title"}}</title>
    <link rel="stylesheet" href="/assets/css/application.css">
    {{csrf_token}}
</head>
<body>
    <header>
        <!-- Include partial from views/layouts/_nav.gohtml -->
        {{template "nav" .}}
    </header>

    <main>
        {{template "yield" .Content}}
    </main>

    <script src="/assets/js/application.js"></script>
</body>
</html>
```

#### Page Template (`views/articles/index.gohtml`)

```html
<h1>Articles</h1>
<form method="POST" action="/articles">
    {{csrf_field}}
    <input type="text" name="title" />
    <button type="submit">Create</button>
</form>

{{range .}}
    <!-- Call nested partial from views/articles/_card.gohtml -->
    {{template "articles#card" .}}
{{end}}
```

#### Built-in Template Helpers

| Helper | Description | Example |
|---|---|---|
| `{{t "key"}}` | Localized translation from `translations/{lang}.json` | `{{t "welcome_message"}}` |
| `{{localize "/path"}}` | Path in the current language (`/nl/path` on a Dutch page) | `<a href="{{localize "/cart"}}">` |
| `{{alternate_links}}` | `hreflang` alternate `<link>` tags incl. `x-default` | `{{alternate_links}}` in `<head>` |
| `{{alternates}}` | Language versions of the page (`.Locale`, `.Path`, `.URL`, `.Current`) | `{{range alternates}}…{{end}}` |
| `{{canonical_url}}` | Absolute URL of the page in the current language | `<link rel="canonical" href="{{canonical_url}}">` |
| `{{v "key"}}` | View variable set in handler via `tracks.AddViewVar` | `{{v "page_title"}}` |
| `{{csrf_field}}` | Generates `<input type="hidden" name="csrf_token" ...>` | `{{csrf_field}}` |
| `{{csrf_token}}` | Raw CSRF token string | `{{csrf_token}}` |
| `{{icon "name"}}` | Inlines SVG Phosphor icon from `assets/icons/` | `{{icon "magnifying-glass-bold" "w-5 h-5"}}` |
| `{{date "format" .Time}}` | Formats a `time.Time` value | `{{date "Jan 02, 2006" .CreatedAt}}` |
| `{{now}}`, `{{today}}`, `{{year}}` | Date helpers | `{{year}}` |
| `{{dict "k1" v1 "k2" v2}}` | Constructs a dictionary for partial passing | `{{template "card" (dict "Item" . "Class" "highlight")}}` |
| `{{cents 1050}}` | Formats cents to dollars/euros (`10.50`) | `{{cents .PriceInCents}}` |
| `{{safe .HTML}}` | Marks string as trusted HTML (`template.HTML`) | `{{safe .RichContent}}` |

#### Internationalization

Translations live in `translations/{locale}.json` (nested keys, accessed with dots: `{{t "nav.cart"}}`). Missing keys fall back to the default language and are logged once per locale and key.

Add an `i18n` block to `config.json` to serve every language on its own URL, which is what search engines expect:

```json
"i18n": { "default": "en", "locales": ["en", "nl", "fr"], "strategy": "path" }
```

* `/nl/about` is routed as `/about` with language `nl`, so you register every route once. Unprefixed URLs use the default language and `/en/...` redirects (301) to the unprefixed URL.
* Cookies and `Accept-Language` never change the language of a URL. When the visitor prefers another configured language, the `suggested_locale` view var holds it, so you can offer a switch.
* Root-relative redirects stay in the current language (`tracks.Redirect("/cart")` sends Dutch visitors to `/nl/cart`). To switch language, redirect to `i18n.URLFor(ctx, "en", "/cart")`.
* Pages whose paths differ per language (translated slugs), or that don't exist in every language, declare their versions; locales left out get no alternate link:

```go
tracks.SetAlternates(r, map[string]string{
    "en": "/coloring-pages/cow",
    "nl": "/kleurplaten/koe", // served as /nl/kleurplaten/koe
})
```

A minimal localized layout:

```html
<html lang="{{v "locale"}}">
<head>
  <link rel="canonical" href="{{canonical_url}}">
  {{alternate_links}}
</head>
<body>
  <a href="{{localize "/"}}">{{t "nav.home"}}</a>
  {{range alternates}}{{if not .Current}}<a href="{{.Path}}" hreflang="{{.Locale}}">{{.Locale}}</a>{{end}}{{end}}
</body>
```

In Go code, use `tracks.T(r, "key")`, `tracks.LocalizePath(r, "/path")` and `i18n.CanonicalURL(ctx)`. Absolute URLs use `base_url` from the `i18n` block, or `base_domain` and `secure` when it's omitted.

Without an `i18n` block, the language is detected per request (`?locale=`, `locale` cookie, session, `Accept-Language`) and URLs are shared by all languages, as before. Set `"strategy": "detect"` to keep that behaviour while limiting it to the configured locales.

#### Multi-Domain Template Overrides

Placing a template in `views/domains/{domain}/{controller}/{action}.gohtml` automatically overrides the base template when accessed from that domain.

---

### 3. Database & Generic Repository

Tracks avoids reflection during database execution by asking models to declare their fields and scanning functions explicitly.

#### Defining a Model

```go
package models

import (
    "context"
    "time"

    "github.com/tmeire/tracks/database"
)

type Article struct {
    ID        int64     `json:"id"`
    Title     string    `json:"title"`
    Content   string    `json:"content"`
    CreatedAt time.Time `json:"created_at"`
}

func (a *Article) TableName() string {
    return "articles"
}

func (a *Article) Fields() []string {
    return []string{"id", "title", "content", "created_at"}
}

func (a *Article) Values() []any {
    return []any{a.ID, a.Title, a.Content, a.CreatedAt}
}

func (a *Article) Scan(ctx context.Context, schema any, row database.Scanner) (*Article, error) {
    var article Article
    err := row.Scan(&article.ID, &article.Title, &article.Content, &article.CreatedAt)
    if err != nil {
        return nil, err
    }
    return &article, nil
}

func (a *Article) HasAutoIncrementID() bool {
    return true
}

func (a *Article) GetID() any {
    return a.ID
}
```

#### Lifecycle Hooks

Models can optionally implement:
- `BeforeCreate(ctx context.Context) error`
- `AfterCreate(ctx context.Context) error`
- `BeforeUpdate(ctx context.Context) error`
- `AfterUpdate(ctx context.Context) error`
- `BeforeDelete(ctx context.Context) error`
- `AfterDelete(ctx context.Context) error`

#### Using the Repository

```go
repo := database.NewRepository[any, *models.Article](nil)

// Insert
article := &models.Article{Title: "Hello Tracks", Content: "Body text", CreatedAt: time.Now()}
saved, err := repo.Create(ctx, article)

// Query by ID
found, err := repo.FindByID(ctx, saved.ID)

// Query Builder
list, err := repo.Select().
    Where("title LIKE ?", "%Tracks%").
    OrderBy("created_at", database.DESC).
    Limit(10).
    Exec(ctx)

// Atomic Updates
err = repo.AtomicUpdate(ctx, saved.ID, database.AtomicOp{
    Field: "views",
    Delta: 1,
})

// Transactions
err = database.WithTransaction(ctx, func(txCtx context.Context) error {
    // queries in txCtx participate in the transaction
    return repo.Update(txCtx, saved)
})
```

---

### 4. Request Parsing & Validation

```go
type CreatePostRequest struct {
    Title   string `form:"title" json:"title" validate:"required,min=5"`
    Content string `form:"content" json:"content" validate:"required"`
    Email   string `form:"email" json:"email" validate:"email"`
}

func CreatePost(r *http.Request) (any, error) {
    var req CreatePostRequest
    if err := tracks.ParseAndValidate(r, &req); err != nil {
        if vErrors, ok := err.(tracks.ValidationErrors); ok {
            return tracks.UnprocessableEntity("Validation failed", vErrors.FieldErrors()), nil
        }
        return tracks.BadRequest(err), nil
    }

    // Validated payload ready to use
    return &tracks.Response{StatusCode: http.StatusCreated, Data: req}, nil
}
```

---

### 5. Modules

Tracks includes plug-and-play modules registered with `router.Module(...)`:

#### Authentication (`modules/authentication`)
```go
import "github.com/tmeire/tracks/modules/authentication"

// Protect routes by role
router.GetFunc("/admin", "admin", "dashboard", adminHandler, 
    authentication.RequireSystemRole("admin"))
```

#### Multitenancy (`modules/multitenancy`)
Routes requests dynamically by subdomain to dedicated SQLite tenant databases while maintaining a central database for users and tenants:
```go
import "github.com/tmeire/tracks/modules/multitenancy"

router.Module(multitenancy.Register)
```

#### Feature Flags (`modules/featureflags`)
Evaluate feature flags tied to database models with in-memory caching:
```go
import "github.com/tmeire/tracks/modules/featureflags"

router.Module(featureflags.Register)
```

#### Mail (`modules/mail`)
```go
import "github.com/tmeire/tracks/modules/mail"

mail.Send(ctx, mail.Message{
    To:      []string{"user@example.com"},
    Subject: "Welcome",
    Body:    "Hello from Tracks!",
})
```

#### Storage (`modules/storage`)
```go
import "github.com/tmeire/tracks/modules/storage"

service := storage.FromContext(r.Context())
err := service.Save(ctx, "avatars/user1.png", fileReader)
```

---

### 6. Background Jobs & Caching

#### In-Memory Cache with Tag Invalidation

```go
cache := router.Cache()
cache.SetWithTags("user:1", userData, []string{"users"}, 10*time.Minute)

// Invalidate all items tagged "users"
cache.InvalidateTag("users")
```

#### Background Job Queue

```go
type SendWelcomeEmailJob struct {
    UserID int64
}

func (j SendWelcomeEmailJob) Handle(ctx context.Context) error {
    // Send email asynchronously
    return nil
}

// Enqueue
router.Queue().Enqueue(ctx, SendWelcomeEmailJob{UserID: 42})
```

---

### 7. CLI Utilities

Build the CLI binary:

```bash
go build -o tracks ./cli/main.go
```

Available commands:

```bash
# Initialize a new application
./tracks init [module_name]

# Generate a controller action and view
./tracks generate controller [method] [path]
# Example: ./tracks generate controller GET /about

# Generate a resourceful scaffold (model, controller, views)
./tracks generate resource [name]
# Example: ./tracks generate resource posts

# Run database migrations (Note the subcommand nesting)
./tracks db db up
./tracks db db down
./tracks db db status

# Asset compilation and hashing
./tracks assets compile
```

---

### 8. Testing

Run the test suite across all packages:

```bash
go test ./...
```

Tracks provides test helpers in `testing.go`:
- `tracks.NewTestRouter(t)`
- `tracks.ExecuteRequest(r, req)`
- `tracks.AssertStatus(t, resp, http.StatusOK)`
- `tracks.AssertJSON(t, resp, expected)`

---

## License

Tracks is open source under the MIT License.
