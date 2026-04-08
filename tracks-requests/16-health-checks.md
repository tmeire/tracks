# Feature Request: Health Check Endpoint

**Priority:** Medium  
**Status:** Open

## Description

The framework lacks a built-in health check endpoint for monitoring application status, database connectivity, and dependency health.

## Current Gap

No `/health` or `/healthz` endpoint exists for:
- Kubernetes/Docker health probes
- Load balancer health checks
- Monitoring systems
- Database connectivity verification

## Required Functionality

1. **Health Check Endpoint**: Configurable endpoint (default: `/health`)
2. **Component Checks**: Database, cache, external services
3. **Status Aggregation**: Overall health based on components
4. **Custom Checks**: Register application-specific health checks
5. **Response Formats**: JSON and plain text
6. **Detail Levels**: Basic (up/down) vs detailed (component status)

## Proposed API

```go
// Automatic health endpoint
router := tracks.New(ctx)
router.HealthCheck("/health") // Adds GET /health endpoint

// With custom checks
router.HealthCheck("/health", tracks.HealthConfig{
    Checks: []tracks.HealthCheck{
        {
            Name: "database",
            Check: func(ctx context.Context) error {
                db := database.FromContext(ctx)
                return db.Ping()
            },
            Critical: true, // Fail overall health if this fails
        },
        {
            Name: "redis",
            Check: func(ctx context.Context) error {
                cache := cache.FromContext(ctx)
                return cache.Ping()
            },
            Critical: false, // Warning only
        },
        {
            Name: "external_api",
            Check: func(ctx context.Context) error {
                resp, err := http.Get("https://api.example.com/status")
                if err != nil {
                    return err
                }
                if resp.StatusCode != 200 {
                    return fmt.Errorf("API returned %d", resp.StatusCode)
                }
                return nil
            },
            Timeout: 5 * time.Second,
        },
    },
})

// Manual health check registration
health := tracks.HealthChecker()
health.Register("payment_gateway", paymentHealthCheck)
health.Register("search_index", searchHealthCheck)

// In handlers - custom health logic
func (h *Handler) Check(r *http.Request) (any, error) {
    // Returns detailed health report
    return tracks.HealthReport{
        Status: "healthy", // or "degraded", "unhealthy"
        Version: "1.2.3",
        Components: []tracks.ComponentHealth{
            {Name: "database", Status: "healthy", Latency: "2ms"},
            {Name: "cache", Status: "healthy", Latency: "1ms"},
            {Name: "queue", Status: "degraded", Error: "high backlog"},
        },
    }, nil
}
```

## Response Examples

**Basic (plain text):**
```
GET /health

HTTP 200 OK
healthy
```

**Detailed (JSON):**
```json
GET /health?detailed=true

{
  "status": "healthy",
  "timestamp": "2025-01-15T10:30:00Z",
  "version": "1.2.3",
  "uptime": "72h15m",
  "components": [
    {
      "name": "database",
      "status": "healthy",
      "response_time": "2ms",
      "last_check": "2025-01-15T10:30:00Z"
    },
    {
      "name": "redis",
      "status": "healthy",
      "response_time": "1ms"
    },
    {
      "name": "queue",
      "status": "degraded",
      "error": "Queue depth: 1000 jobs",
      "response_time": "5ms"
    }
  ]
}
```

## Use Cases

- Kubernetes liveness/readiness probes
- Docker HEALTHCHECK
- Load balancer health checks
- Monitoring dashboards
- Deployment verification
- SLA monitoring

## Acceptance Criteria

- [ ] Built-in `/health` endpoint
- [ ] Database connectivity check
- [ ] Cache connectivity check (if configured)
- [ ] Custom health check registration
- [ ] Critical vs non-critical component distinction
- [ ] JSON and plain text response formats
- [ ] Response time/latency reporting
- [ ] Configurable endpoint path
- [ ] Timeout handling for checks
- [ ] Documentation and examples

## Kubernetes Integration

```yaml
# deployment.yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health?detailed=true
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```
