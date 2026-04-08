# Feature Request: Background Job Queue

**Priority:** Medium  
**Status:** Open

## Description

The framework lacks a background job processing system. Currently, async work is done with goroutines (e.g., `DeliverLater` in mail module), which is not reliable for production.

## Current Implementation

```go
// modules/mail/mail.go
func (m *Message) DeliverLater(ctx context.Context) {
    go func() {
        _ = m.DeliverNow(context.Background())
    }()
}
```

Problems with this approach:
- Jobs lost on application restart
- No retry logic
- No visibility into job status
- No rate limiting
- Hard to scale

## Required Functionality

1. **Queue Interface**: Generic job queue with multiple backends
2. **In-Memory Queue**: For development/testing
3. **Database Queue**: PostgreSQL/MySQL-backed queue
4. **Redis Queue**: Redis-backed queue (Sidekiq-compatible)
5. **Job Definition**: Type-safe job definitions with parameters
6. **Worker Pool**: Configurable worker concurrency
7. **Retry Logic**: Exponential backoff with max retries
8. **Scheduling**: Schedule jobs for future execution
9. **Monitoring**: Job status, failures, and metrics
10. **Dead Letter Queue**: Failed jobs that exceeded retries

## Proposed API

```go
// Define a job
type SendWaitlistWelcomeEmail struct {
    Email string
    Name  string
    Domain string
}

func (j SendWaitlistWelcomeEmail) Handle(ctx context.Context) error {
    // Send email logic
    return mailer.Send(ctx, j.Email, "welcome", map[string]any{
        "Name": j.Name,
        "Domain": j.Domain,
    })
}

func (j SendWaitlistWelcomeEmail) MaxRetries() int { return 3 }
func (j SendWaitlistWelcomeEmail) RetryDelay(attempt int) time.Duration {
    return time.Duration(attempt*attempt) * time.Minute
}

// Enqueue job
queue := jobs.FromContext(ctx)
queue.Enqueue(SendWaitlistWelcomeEmail{
    Email: user.Email,
    Name:  user.Name,
    Domain: domain,
})

// Schedule for later
queue.EnqueueAt(time.Now().Add(1*time.Hour), SendWaitlistWelcomeEmail{...})

// Or schedule with cron
queue.Schedule("0 9 * * *", DailyDigestJob{})

// Configuration
config := tracks.Config{
    Jobs: jobs.Config{
        Driver: "redis",
        Redis: jobs.RedisConfig{
            Addr: "localhost:6379",
        },
        Workers: 5,
        Queues: map[string]int{
            "default": 3,
            "mail":    2,
            "billing": 1,
        },
    },
}
```

## Use Cases

- Sending welcome emails
- Processing image uploads
- Generating reports
- Data imports/exports
- Webhook delivery
- Scheduled maintenance tasks
- Billing invoice generation

## Acceptance Criteria

- [ ] Job interface with type-safe parameters
- [ ] Multiple queue backends (memory, database, Redis)
- [ ] Worker pool with configurable concurrency
- [ ] Retry logic with exponential backoff
- [ ] Scheduled/delayed job execution
- [ ] Cron-like recurring jobs
- [ ] Dead letter queue for failed jobs
- [ ] Job monitoring and metrics
- [ ] Web UI for job monitoring (optional)
- [ ] Documentation and examples

## Job Lifecycle

```
Enqueue -> Queue -> Worker Pickup -> Execute
                 |                    |
                 |                    v
                 |               Success -> Done
                 |                    |
                 |               Failure -> Retry? -> Requeue
                 |                          |
                 |                          v
                 |                    Max Retries -> Dead Letter Queue
```
