package tracks

import (
	"context"
	"net/http"
	"time"
)

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
)

type HealthCheck struct {
	Name     string
	Check    func(context.Context) error
	Critical bool
	Timeout  time.Duration
}

type HealthConfig struct {
	Checks []HealthCheck
}

type ComponentHealth struct {
	Name         string       `json:"name"`
	Status       HealthStatus `json:"status"`
	Error        string       `json:"error,omitempty"`
	ResponseTime string       `json:"response_time,omitempty"`
}

type HealthReport struct {
	Status     HealthStatus      `json:"status"`
	Timestamp  time.Time         `json:"timestamp"`
	Components []ComponentHealth `json:"components,omitempty"`
}

func (r *router) HealthCheck(path string, config ...HealthConfig) Router {
	if path == "" {
		path = "/health"
	}

	var conf HealthConfig
	if len(config) > 0 {
		conf = config[0]
	}

	// Add default checks if not provided
	if len(conf.Checks) == 0 {
		conf.Checks = append(conf.Checks, HealthCheck{
			Name: "database",
			Check: func(ctx context.Context) error {
				// We don't have direct access to sql.DB here easily without a ping method on Database interface
				// But we can try a simple query.
				_, err := r.Database().ExecContext(ctx, "SELECT 1")
				return err
			},
			Critical: true,
		})
	}

	r.GetFunc(path, "health", "check", func(req *http.Request) (any, error) {
		detailed := req.URL.Query().Get("detailed") == "true"
		
		report := HealthReport{
			Status:    StatusHealthy,
			Timestamp: time.Now(),
		}

		for _, check := range conf.Checks {
			ctx := req.Context()
			if check.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, check.Timeout)
				defer cancel()
			}

			start := time.Now()
			err := check.Check(ctx)
			duration := time.Since(start)

			comp := ComponentHealth{
				Name:         check.Name,
				Status:       StatusHealthy,
				ResponseTime: duration.String(),
			}

			if err != nil {
				comp.Error = err.Error()
				if check.Critical {
					comp.Status = StatusUnhealthy
					report.Status = StatusUnhealthy
				} else {
					comp.Status = StatusDegraded
					if report.Status == StatusHealthy {
						report.Status = StatusDegraded
					}
				}
			}

			if detailed {
				report.Components = append(report.Components, comp)
			}
		}

		if !detailed {
			status := string(report.Status)
			if report.Status == StatusHealthy {
				status = "healthy"
			}
			
			// Return a Response that will be rendered as plain text if possible
			return &Response{
				StatusCode: http.StatusOK,
				Data:       status,
			}, nil
		}

		return report, nil
	})

	return r
}
