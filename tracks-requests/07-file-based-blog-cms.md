# Feature Request: File-Based Blog/CMS Support

**Priority:** Low  
**Status:** Open

## Description

Many applications need simple content management without a database. A file-based blog/CMS system allows content to be stored as static files with metadata.

## Current Implementation

```go
// Posts are defined in code with metadata
type BlogPost struct {
    Title   string
    Slug    string
    Date    time.Time
    Summary string
    Image   string
    Content template.HTML
}

func (h *BlogHandler) getPosts() []BlogPost {
    return []BlogPost{
        {
            Title:   "Measuring Go Applications",
            Slug:    "measuring-go-applications",
            Date:    time.Date(2025, 8, 27, 0, 0, 0, 0, time.UTC),
            Summary: "If traces tell you why something is wrong...",
            Image:   "/images/measuring-go-applications.png",
        },
        // ...
    }
}

// Content is stored in HTML files
tmpl, err := template.ParseFiles(
    filepath.Join(h.templateDir, "layout.html"),
    filepath.Join(h.templateDir, "blog", "post.html"),
    postFile,  // e.g., "templates/blog/posts/measuring-go-applications.html"
)
```

## Required Functionality

1. **File Discovery**: Automatically discover content files from directory
2. **Frontmatter Support**: Parse metadata (YAML/TOML/JSON) from file headers
3. **Slug Generation**: URL-friendly identifiers from filenames or frontmatter
4. **Content Rendering**: Integrate with template system
5. **Listing/Index**: Generate list pages from discovered content

## Proposed API

```go
// File structure:
// content/
//   blog/
//     hello-world.md
//     another-post.md

// Define content type
type BlogPost struct {
    tracks.Content `yaml:",inline"`
    Title          string   `yaml:"title"`
    Date           time.Time `yaml:"date"`
    Tags           []string `yaml:"tags"`
    Summary        string   `yaml:"summary"`
}

// Register content resource
router.Content("/blog", "./content/blog", tracks.ContentConfig{
    Type: BlogPost{},
    Layout: "blog/post",
    ListLayout: "blog/list",
    SortBy: "date",
    SortDesc: true,
})

// Or as a controller
posts := tracks.NewContentController[BlogPost]("./content/blog")
router.ControllerAtPath("/blog", posts)
```

## Use Cases

- Simple blogs without database
- Documentation sites
- Marketing pages
- Changelogs
- Help/knowledge base articles

## Acceptance Criteria

- [ ] Directory-based content discovery
- [ ] Frontmatter parsing (YAML, TOML, or JSON)
- [ ] Automatic slug generation
- [ ] Sorting and filtering options
- [ ] List/index pages
- [ ] Individual content pages
- [ ] Integration with template/layout system
- [ ] Hot reloading in development mode
- [ ] Documentation and examples

## File Format Example

```markdown
---
title: "Hello World"
date: 2025-01-15T10:00:00Z
tags: ["go", "tutorial"]
summary: "Getting started with Tracks"
---

# Hello World

This is the content of the blog post...
```
