# Tracks Framework Improvements

This document outlines architectural and runtime improvements for the Tracks framework to enhance simplicity, efficiency, and modularity.

## 1. Architectural: Breaking the "God Object" Router
The `Router` interface currently violates the **Interface Segregation Principle**, managing routing, database, configuration, templating, caching, and background jobs.

*   **Improvement:** Split `Router` into focused interfaces:
    *   **`Registrar`**: Purely for defining routes and middleware.
    *   **`ServiceContainer`**: A container for shared services (DB, Cache, Queue).
    *   **`Context`**: A lighter request-scoped object passed to handlers.
*   **Impact:** Better modularity, easier testing, and a cleaner API.

## 2. Runtime: Context-Aware Template Rendering
The current template system clones the entire template and rebuilds the `FuncMap` on every request.

*   **Improvement:** Define `t` (translate) and `v` (view var) functions once at boot. These functions should pull data directly from the `http.Request` context.
*   **Impact:** Eliminates per-request cloning and map allocations, reducing GC pressure and latency.

## 3. Concurrency: Thread-Safe Icon Cache
The `iconCache` in `templates.go` is a global map accessed concurrently during template execution without synchronization.

*   **Improvement:** Replace the global map with a `sync.Map` or protect it with a `sync.RWMutex`.
*   **Impact:** Prevents data races and production panics.

## 4. DX: Decoupling HTMX Logic
HTMX-specific logic (like `HX-Redirect`) is currently hardcoded into the core `action.go`.

*   **Improvement:** Introduce **Response Modifiers** or specialized middleware hooks to handle library-specific headers.
*   **Impact:** Keeps the core framework library-agnostic and extensible for other tools like Turbo JS.

## 5. Efficiency: Pre-computed Middleware Chains
Middlewares are currently wrapped and allocated dynamically on every request.

*   **Improvement:** Resolve the full middleware chain for every route at boot time.
*   **Impact:** Moves the "linking" cost from the request path to startup, improving runtime performance.

## 6. Memory: Buffer Pooling for Rendering
Rendering directly to `ResponseWriter` can lead to partial/broken pages if an error occurs mid-render.

*   **Improvement:** Use a `sync.Pool` of `bytes.Buffer` to render templates before writing to the response.
*   **Impact:** Enables clean 500 error responses on failure and reduces memory allocations.

## 7. Simplification: Unified Request Binding
Request parsing and validation are currently fragmented.

*   **Improvement:** Standardize on a single `Bind(r, &dst)` pattern that integrates parsing and validation.
*   **Impact:** Reduces boilerplate in controllers and ensures data integrity.
