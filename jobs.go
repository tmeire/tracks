package tracks

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Job interface {
	Handle(ctx context.Context) error
}

type RetryableJob interface {
	Job
	MaxRetries() int
	RetryDelay(attempt int) time.Duration
}

type Queue interface {
	Enqueue(ctx context.Context, job Job) error
	EnqueueAt(ctx context.Context, at time.Time, job Job) error
	Start(ctx context.Context) error
	Stop() error
}

type memoryQueue struct {
	jobs   chan queuedJob
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

type queuedJob struct {
	job     Job
	at      time.Time
	attempt int
}

func NewMemoryQueue(concurrency int) Queue {
	return &memoryQueue{
		jobs: make(chan queuedJob, 1000),
	}
}

func (q *memoryQueue) Enqueue(ctx context.Context, job Job) error {
	return q.EnqueueAt(ctx, time.Now(), job)
}

func (q *memoryQueue) EnqueueAt(ctx context.Context, at time.Time, job Job) error {
	q.jobs <- queuedJob{job: job, at: at, attempt: 0}
	return nil
}

func (q *memoryQueue) Start(ctx context.Context) error {
	q.ctx, q.cancel = context.WithCancel(ctx)
	
	// Start workers
	for i := 0; i < 5; i++ {
		q.wg.Add(1)
		go q.worker()
	}
	
	return nil
}

func (q *memoryQueue) worker() {
	defer q.wg.Done()
	for {
		select {
		case <-q.ctx.Done():
			return
		case qj := <-q.jobs:
			if time.Now().Before(qj.at) {
				// Re-enqueue if it's too early
				// This is inefficient but simple for in-memory
				go func() {
					time.Sleep(time.Until(qj.at))
					q.jobs <- qj
				}()
				continue
			}

			err := qj.job.Handle(q.ctx)
			if err != nil {
				slog.Error("job failed", "error", err, "attempt", qj.attempt)
				if rj, ok := qj.job.(RetryableJob); ok {
					if qj.attempt < rj.MaxRetries() {
						qj.attempt++
						qj.at = time.Now().Add(rj.RetryDelay(qj.attempt))
						q.jobs <- qj
					}
				}
			}
		}
	}
}

func (q *memoryQueue) Stop() error {
	if q.cancel != nil {
		q.cancel()
	}
	q.wg.Wait()
	return nil
}

type jobContextKey struct{}

func WithQueue(ctx context.Context, q Queue) context.Context {
	return context.WithValue(ctx, jobContextKey{}, q)
}

func QueueFromContext(ctx context.Context) Queue {
	if q, ok := ctx.Value(jobContextKey{}).(Queue); ok {
		return q
	}
	return nil
}
