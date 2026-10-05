---
name: tracks-site-builder
description: Guides setup, scaffolding, and building full-stack web applications, APIs, or sites on top of the Tracks Go web framework. Use when the user wants to create a new Tracks project, add controllers/views/models, configure database/migrations, or build features with Tracks.
---

# Tracks Site Builder

This skill provides step-by-step procedures, architectural best practices, and recipes for setting up and building production-ready web applications on top of the **Tracks** Go web framework.

## Overview of Tracks

Tracks is a full-stack Go web framework built around:
1. `net/http.ServeMux` with Go 1.22+ pattern matching (`/posts/{id}`).
2. Go `html/template` with layouts (`views/layouts/application.gohtml`) and partials (`views/dir/_partial.gohtml`).
3. Reflection-free, generic SQLite database repositories (`database.Repository[S, T]`).
4. Automatic OpenTelemetry tracing, metrics, and structured logging.
5. Modular extensions: `authentication`, `multitenancy`, `featureflags`, `mail`, `storage`.

---

## Workflow: Scaffolding a New Site

### Step 1: Initialize Project Structure
Run the following structure setup:

```bash
mkdir -p config controllers models views/layouts views/default public/css public/js migrations/central data
```

### Step 2: Initialize `go.mod`
```bash
go mod init <module-path>
go get github.com/tmeire/tracks
```

### Step 3: Configure `config/config.json`
Tracks reads configuration from `./config/config.json` (or `TRACKS_CONFIG_FILE`):

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

### Step 4: Create Master Layout (`views/layouts/application.gohtml`)
Tracks requires a layout file named `application.gohtml` under `views/layouts/`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{v "title"}}</title>
  <link rel="stylesheet" href="/assets/css/application.css">
</head>
<body>
  <header>
    <h1>{{v "appName"}}</h1>
  </header>

  <main>
    {{template "yield" .Content}}
  </main>

  <footer>
    <p>Powered by Tracks</p>
  </footer>
  <script src="/assets/js/application.js"></script>
</body>
</html>
```

### Step 5: Create Default View (`views/default/home.gohtml`)
Note: In Go templates, struct field names or map keys must match exact casing. Because Tracks uses `missingkey=error`, use matching case:

```html
<div class="welcome">
  <h2>{{.message}}</h2>
  <p>Your Tracks application is running.</p>
</div>
```

### Step 6: Create `main.go`
IMPORTANT: `tracks.New(ctx)` already sets up OpenTelemetry and database connections. Pass `ctx` to `router.Run(ctx)`:

```go
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

	// Serve static assets from ./public at /assets
	router.Static("/assets", "./public")

	// Home action
	router.GetFunc("/", "default", "home", func(r *http.Request) (any, error) {
		r = tracks.AddViewVar(r, "appName", "My Site")
		r = tracks.AddViewVar(r, "title", "Home")
		return map[string]string{
			"message": "Welcome to your new Tracks site!",
		}, nil
	})

	if err := router.Run(ctx); err != nil {
		log.Fatalf("Server exited: %v", err)
	}
}
```

---

## Workflow: Building Features

### 1. Adding Controller Actions
Use `router.GetFunc`, `PostFunc`, `PutFunc`, `DeleteFunc`:

```go
router.GetFunc("/items", "items", "index", controllers.ListItems)
router.PostFunc("/items", "items", "create", controllers.CreateItem)
```

Handler pattern:
```go
func CreateItem(r *http.Request) (any, error) {
    var req CreateItemInput
    if err := tracks.ParseAndValidate(r, &req); err != nil {
        if vErrors, ok := err.(tracks.ValidationErrors); ok {
            return tracks.UnprocessableEntity("Validation failed", vErrors.FieldErrors()), nil
        }
        return tracks.BadRequest(err), nil
    }

    // Return redirect on success
    return &tracks.Response{
        StatusCode: http.StatusSeeOther,
        Location:   "/items",
    }, nil
}
```

### 2. Creating Database Models & Repositories
Tracks models implement `database.Model[S, T]` explicitly without reflection:

```go
package models

import (
	"context"
	"time"

	"github.com/tmeire/tracks/database"
)

type Item struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

func (i *Item) TableName() string { return "items" }
func (i *Item) Fields() []string { return []string{"id", "title", "created_at"} }
func (i *Item) Values() []any    { return []any{i.ID, i.Title, i.CreatedAt} }
func (i *Item) HasAutoIncrementID() bool { return true }
func (i *Item) GetID() any { return i.ID }

func (i *Item) Scan(ctx context.Context, schema any, row database.Scanner) (*Item, error) {
	var item Item
	err := row.Scan(&item.ID, &item.Title, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}
```

Repository usage:
```go
repo := database.NewRepository[any, *models.Item](nil)

// Query
items, err := repo.Select().
    Where("title LIKE ?", "%test%").
    OrderBy("created_at", database.DESC).
    Limit(20).
    Exec(ctx)

// Create
saved, err := repo.Create(ctx, &models.Item{Title: "New", CreatedAt: time.Now()})
```

### 3. Migrations
Place SQL migrations in `migrations/central/` (e.g. `00001_create_items.sql`):

```sql
-- +goose Up
CREATE TABLE items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE items;
```

To run migrations using the Tracks CLI:
```bash
tracks db db up
```
*(Notice the `db db` subcommand nesting required by the CLI).*

### 4. Partials & View Helpers
- **Partials**: Place files with leading underscores: `views/items/_card.gohtml`. Call them with `{{template "items#card" .}}`.
- **View Variables**: Set via `r = tracks.AddViewVar(r, "key", val)` and read in template via `{{v "key"}}`.
- **Translations**: Put keys in `translations/en.json`, read in template via `{{t "key"}}`.
- **CSRF**: Include `{{csrf_field}}` inside `<form>` tags or `{{csrf_token}}` in headers/meta.

---

## Known Framework Quirks & Gotchas

When building with Tracks, be aware of these design traits:

1. **CLI `main.go.tmpl` Drift**:
   Do NOT use `otel.SetupTracerProvider` (it does not exist in `otel`). `tracks.New(ctx)` already handles OTel.
2. **`Run` Context Parameter**:
   `router.Run(ctx)` requires a `context.Context`.
3. **CLI Subcommand Nesting**:
   Database tasks require `tracks db db up` rather than `tracks db up` or `tracks db migrate`.
4. **Template Missing Key Errors**:
   Tracks parses layouts with `.Option("missingkey=error")`. Make sure map keys or struct fields passed in `resp.Data` match the template's casing exactly.
5. **Icon Hardcoding**:
   Phosphor icons fall back to `floral-crm/assets/icons/phosphor` if not found in `assets/icons/phosphor`. Store icons under `./assets/icons/phosphor/{weight}/{name}.svg`.
6. **URL Paths and Slashes**:
   `Resource` registers routes with a trailing slash (`basePath + "/"`). Ensure incoming requests include or tolerate the trailing slash.
