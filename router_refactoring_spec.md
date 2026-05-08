# Spec: Router Refactoring (Interface Segregation & Branching)

## Overview
Currently, the `tracks.Router` is a "God Object" that holds both the routing definitions and the runtime service dependencies. This spec details the split into focused interfaces and the adoption of a "Branching" (closure-based) fluent API for routing.

## Proposed Architecture

### 1. The `Registrar` Interface
Focuses strictly on defining the application structure. It uses closures for grouping to ensure that indentation matches the URL hierarchy and prevents middleware "leakage."

```go
type Registrar interface {
    // Basic Verbs
    Get(path string, handler ActionFunc) Registrar
    Post(path string, handler ActionFunc) Registrar
    Put(path string, handler ActionFunc) Registrar
    Patch(path string, handler ActionFunc) Registrar
    Delete(path string, handler ActionFunc) Registrar

    // Scoping (The "Branching" Pattern)
    // The closure 'fn' receives a scoped Registrar. 
    // Group() returns the ORIGINAL Registrar to allow further chaining at the same level.
    Group(prefix string, fn func(r Registrar)) Registrar
    
    // Middleware
    // When called on a scoped registrar, it applies only to that scope.
    Use(mws ...Middleware) Registrar

    // Specialized Mounting
    Resource(path string, r Resource) ResourceRegistrar
    Module(m Module) Registrar
}

type ResourceRegistrar interface {
    Only(actions ...string) ResourceRegistrar
    Except(actions ...string) ResourceRegistrar
    With(mws ...Middleware) ResourceRegistrar
    // Returns back to the parent Registrar
    End() Registrar 
}
```

### 2. The `ServiceContainer`
Holds long-lived, shared dependencies. This is the "Engine" of the application.

```go
type ServiceContainer interface {
    DB() database.Database
    Cache() Cache
    Queue() Queue
    Templates() *Templates
    Config() Config
    
    // Creates a new Registrar bound to these services
    Router() Registrar
    
    // Lifecycle
    Run(ctx context.Context, r Registrar) error
}
```

### 3. The `ActionContext`
A request-scoped object passed to every handler. It encapsulates the request, the response writer, and access to services.

```go
type ActionContext struct {
    Request  *http.Request
    Response http.ResponseWriter
    Services ServiceContainer
    Params   map[string]string // URL parameters

    // Fluent Response Helpers
    JSON(status int, data any) error
    Render(view string, data any) error
    Redirect(url string) error
    Error(status int, message string) error
}

type ActionFunc func(ctx *ActionContext) error
```

## Example: Refactored Registration in Floral CRM

```go
services.Router().
    Get("/", controllers.Index).
    Group("/admin", func(admin) {
        admin.Use(authentication.RequireSystemRole("superadmin"))
        
        admin.Get("/tenants", tenants.Index)
        admin.Post("/tenants/{id}/activate", tenants.Activate)
    }).
    // This Resource call is back at the root level because Group() returned the root.
    Resource("/events", &tenants.Events{}).
        With(authentication.WithAnyRole).
    End().
    Module(multitenancy.Register)
```

## Migration Strategy

1.  **Phase 1: Interface Intro.** Define `Registrar`, `ServiceContainer`, and `ActionContext`. Update the existing `router` struct to implement both old and new interfaces.
2.  **Phase 2: Adapter Handlers.** Provide a `WrapOld(ActionFunc)` helper so old-style handlers can work within the new `Registrar`.
3.  **Phase 3: Controller Refactor.** Update `BaseController` to store `ServiceContainer` instead of `Router`.
4.  **Phase 4: Final Cleanup.** Deprecate the old `tracks.New()` entry point in favor of `tracks.NewServiceContainer()`.

## Benefits
- **Visual Clarity:** Indentation in `main.go` provides an immediate "map" of the application.
- **Middleware Safety:** Closures physically bound the scope of middlewares, preventing accidental security leaks.
- **Testability:** `ActionContext` can be easily mocked for unit testing handlers without a network stack.
