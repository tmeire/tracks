# Feature Request: Multi-Prefix Static File Serving

**Priority:** Medium  
**Status:** Open

## Description

The framework currently only supports a single static file prefix. Applications often need to serve static files from multiple URL prefixes mapped to different directories.

## Current Limitation

```go
// Current tracks only supports one static path
r.static("/assets/", "public")  // Only one allowed

// But applications need multiple:
mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./static/css"))))
mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("./static/js"))))
mux.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("./images"))))
```

## Required Functionality

1. **Multiple Static Routes**: Register multiple URL prefixes
2. **Path Stripping**: Correctly strip URL prefixes when serving files
3. **Security**: Prevent directory traversal attacks
4. **Caching Headers**: Optional cache control headers

## Proposed API

```go
// Multiple static routes
router.Static("/css/", "./static/css")
router.Static("/js/", "./static/js")
router.Static("/images/", "./images")
router.Static("/assets/", "./public")

// With options
router.StaticWithConfig("/css/", "./static/css", tracks.StaticConfig{
    CacheControl: "public, max-age=31536000",
    StripPrefix: true,
    DisableListing: true,
})
```

## Use Cases

- Organizing assets by type (CSS, JS, images)
- Separating uploaded files from build assets
- Multiple asset sources
- Legacy URL compatibility

## Acceptance Criteria

- [ ] Ability to register multiple static file handlers
- [ ] Each with independent URL prefix and directory
- [ ] Path prefix stripping works correctly
- [ ] Directory traversal protection
- [ ] Optional cache control headers
- [ ] Method chaining for fluent API
- [ ] Documentation and examples

## Security Considerations

- Prevent `..` path traversal
- Validate that directories exist at registration time
- Option to disable directory listing
