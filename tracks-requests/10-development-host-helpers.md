# Feature Request: Development Host Helpers

**Priority:** Low  
**Status:** Open

## Description

Local development of multi-domain applications requires entries in `/etc/hosts`. Providing helpful logging of required host entries improves developer experience.

## Current Implementation

```go
func main() {
    // ...
    port := ":8080"
    log.Printf("Server starting on port %s", port)
    log.Printf("Test domains locally by adding to /etc/hosts:")
    log.Printf("  127.0.0.1 superapp.localhost")
    log.Printf("  127.0.0.1 devtools.localhost")
    log.Printf("  127.0.0.1 cloudsync.localhost")
    // ...
}
```

## Required Functionality

1. **Domain Logging**: Log all configured domains on startup
2. **Hosts File Help**: Print `/etc/hosts` entries for easy copy-paste
3. **Port Detection**: Show correct port in host entries
4. **Development Mode**: Only show in development, not production

## Proposed API

```go
// Configuration
config := tracks.Config{
    BaseDomain: "localhost",
    Development: true,
    Domains: []string{
        "superapp",
        "devtools", 
        "cloudsync",
    },
}

// On startup, automatically logs:
// Server starting on port 8080
// Add the following to /etc/hosts for local testing:
//   127.0.0.1 superapp.localhost
//   127.0.0.1 devtools.localhost
//   127.0.0.1 cloudsync.localhost

// Or explicit helper
router.LogHostEntries()  // Logs all registered domains

// With custom message
router.LogHostEntriesWithMessage("Configure local domains:")
```

## Use Cases

- Multi-domain application development
- Onboarding new developers
- Testing subdomain routing
- Local SSL/TLS testing

## Acceptance Criteria

- [ ] Automatically detect configured domains
- [ ] Log formatted /etc/hosts entries on startup (dev mode only)
- [ ] Include correct port numbers
- [ ] Support for custom base domains
- [ ] Option to disable logging
- [ ] Works with wildcard domains (optional)
- [ ] Documentation

## Example Output

```
=== Local Development Setup ===
Add these entries to /etc/hosts:

127.0.0.1  superapp.localhost
127.0.0.1  devtools.localhost
127.0.0.1  cloudsync.localhost

Or run this command:
sudo sh -c 'echo "127.0.0.1 superapp.localhost devtools.localhost cloudsync.localhost" >> /etc/hosts'

================================
```

## Security Considerations

- Only display in development mode
- Never log in production
- Consider environment variable override (e.g., `TRACKS_DEV_DOMAINS`)
