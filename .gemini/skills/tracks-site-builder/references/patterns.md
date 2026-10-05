# Tracks Code Patterns & Recipes

## 1. Minimal `config/config.json`

Tracks expects configuration at `./config/config.json`:

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

## 2. Standard `main.go`

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/tmeire/tracks"
	"myapp/controllers"
)

func main() {
	ctx := context.Background()

	// tracks.New loads config/config.json, initializes SQLite, and sets up OpenTelemetry
	router := tracks.New(ctx)

	// Static assets from ./public mounted at /assets
	router.Static("/assets", "./public")

	// Home page: registers "/" -> controller: "default", action: "home"
	// Views mapped to ./views/default/home.gohtml
	router.GetFunc("/", "default", "home", controllers.Home)

	// Resourceful routes for posts
	router.Resource(&controllers.PostsResource{})

	// Start the server (Pass ctx)
	if err := router.Run(ctx); err != nil {
		log.Fatalf("Server exited: %v", err)
	}
}
```

## 3. Base Layout (`views/layouts/application.gohtml`)

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{v "title"}}</title>
  <link rel="stylesheet" href="/assets/css/application.css">
  {{csrf_token}}
</head>
<body>
  <header>
    <h1><a href="/">My App</a></h1>
    <nav>
      <a href="/posts">Posts</a>
      <a href="/posts/new">New Post</a>
    </nav>
  </header>

  <main>
    {{template "yield" .Content}}
  </main>

  <footer>
    <p>&copy; {{year}} My App. Powered by Tracks.</p>
  </footer>

  <script src="/assets/js/application.js"></script>
</body>
</html>
```

## 4. Standard Controller Action with Form Validation

```go
package controllers

import (
	"net/http"

	"github.com/tmeire/tracks"
	"myapp/models"
)

type NewPostInput struct {
	Title   string `form:"title" validate:"required,min=3"`
	Content string `form:"content" validate:"required"`
}

func CreatePost(r *http.Request) (any, error) {
	var input NewPostInput
	if err := tracks.ParseAndValidate(r, &input); err != nil {
		if vErrs, ok := err.(tracks.ValidationErrors); ok {
			return tracks.UnprocessableEntity("Validation failed", vErrs.FieldErrors()), nil
		}
		return tracks.BadRequest(err), nil
	}

	_ = input

	return &tracks.Response{
		StatusCode: http.StatusSeeOther,
		Location:   "/posts",
	}, nil
}
```

## 5. Model Implementation (`models/post.go`)

Models implement `database.Model[any, *Post]`:

```go
package models

import (
	"context"
	"time"

	"github.com/tmeire/tracks/database"
)

type Post struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (p *Post) TableName() string {
	return "posts"
}

func (p *Post) Fields() []string {
	return []string{"id", "title", "content", "created_at"}
}

func (p *Post) Values() []any {
	return []any{p.ID, p.Title, p.Content, p.CreatedAt}
}

func (p *Post) Scan(ctx context.Context, schema any, row database.Scanner) (*Post, error) {
	var post Post
	err := row.Scan(&post.ID, &post.Title, &post.Content, &post.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (p *Post) HasAutoIncrementID() bool {
	return true
}

func (p *Post) GetID() any {
	return p.ID
}
```

## 6. Goose Migration (`migrations/central/00001_create_posts.sql`)

```sql
-- +goose Up
CREATE TABLE posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE posts;
```

## 7. Resource Controller (`controllers/posts.go`)

```go
package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tmeire/tracks"
	"github.com/tmeire/tracks/database"
	"myapp/models"
)

type PostsResource struct {
	repo *database.Repository[any, *models.Post]
}

func NewPostsResource() *PostsResource {
	return &PostsResource{
		repo: database.NewRepository[any, *models.Post](nil),
	}
}

func (p *PostsResource) Index(r *http.Request) (any, error) {
	ctx := r.Context()
	posts, err := p.repo.Select().OrderBy("created_at", database.DESC).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (p *PostsResource) New(r *http.Request) (any, error) {
	return map[string]any{}, nil
}

func (p *PostsResource) Create(r *http.Request) (any, error) {
	ctx := r.Context()
	var input struct {
		Title   string `form:"title" validate:"required"`
		Content string `form:"content" validate:"required"`
	}
	if err := tracks.ParseAndValidate(r, &input); err != nil {
		return tracks.BadRequest(err), nil
	}

	post := &models.Post{
		Title:     input.Title,
		Content:   input.Content,
		CreatedAt: time.Now(),
	}
	if _, err := p.repo.Create(ctx, post); err != nil {
		return nil, err
	}

	return &tracks.Response{
		StatusCode: http.StatusSeeOther,
		Location:   "/posts",
	}, nil
}

func (p *PostsResource) Show(r *http.Request) (any, error) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(r.PathValue("posts_id"), 10, 64)
	post, err := p.repo.FindByID(ctx, id)
	if err != nil {
		return tracks.NotFound("Post not found"), nil
	}
	return post, nil
}

func (p *PostsResource) Edit(r *http.Request) (any, error) {
	return p.Show(r)
}

func (p *PostsResource) Update(r *http.Request) (any, error) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(r.PathValue("posts_id"), 10, 64)
	post, err := p.repo.FindByID(ctx, id)
	if err != nil {
		return tracks.NotFound("Post not found"), nil
	}

	var input struct {
		Title   string `form:"title" validate:"required"`
		Content string `form:"content" validate:"required"`
	}
	if err := tracks.ParseAndValidate(r, &input); err != nil {
		return tracks.BadRequest(err), nil
	}

	post.Title = input.Title
	post.Content = input.Content
	if err := p.repo.Update(ctx, post); err != nil {
		return nil, err
	}

	return &tracks.Response{
		StatusCode: http.StatusSeeOther,
		Location:   "/posts",
	}, nil
}

func (p *PostsResource) Destroy(r *http.Request) (any, error) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(r.PathValue("posts_id"), 10, 64)
	post, err := p.repo.FindByID(ctx, id)
	if err != nil {
		return tracks.NotFound("Post not found"), nil
	}

	if err := p.repo.Delete(ctx, post); err != nil {
		return nil, err
	}

	return &tracks.Response{
		StatusCode: http.StatusSeeOther,
		Location:   "/posts",
	}, nil
}
```
