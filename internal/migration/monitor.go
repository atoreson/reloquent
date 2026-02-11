package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/reloquent/reloquent/internal/aws"
)

const (
	pollInterval = 10 * time.Second
)

// Monitor polls provisioner status and fires callbacks with progress.
type Monitor struct {
	provisioner aws.Provisioner
	resourceID  string
}

// NewMonitor creates a new migration monitor.
func NewMonitor(prov aws.Provisioner, resourceID string) *Monitor {
	return &Monitor{
		provisioner: prov,
		resourceID:  resourceID,
	}
}

// Poll repeatedly checks the provisioner status until migration completes or fails.
func (m *Monitor) Poll(ctx context.Context, callback StatusCallback) (*Status, error) {
	status := &Status{
		Phase: "running",
	}

	for {
		select {
		case <-ctx.Done():
			status.Phase = "failed"
			status.Errors = append(status.Errors, "migration cancelled")
			return status, ctx.Err()
		default:
		}

		provStatus, err := m.provisioner.Status(ctx, m.resourceID)
		if err != nil {
			return nil, fmt.Errorf("polling status: %w", err)
		}

		switch provStatus.State {
		case "COMPLETED":
			status.Phase = "completed"
			status.Overall.PercentComplete = 100
			if callback != nil {
				callback(status)
			}
			return status, nil

		case "FAILED":
			status.Phase = "failed"
			if provStatus.Message != "" {
				status.Errors = append(status.Errors, provStatus.Message)
			}
			// Check for partial failure
			failedCount := 0
			completedCount := 0
			for _, col := range status.Collections {
				switch col.State {
				case "failed":
					failedCount++
				case "completed":
					completedCount++
				}
			}
			if completedCount > 0 && failedCount > 0 {
				status.Phase = "partial_failure"
			}
			if callback != nil {
				callback(status)
			}
			return status, fmt.Errorf("migration failed: %s", provStatus.Message)

		case "TERMINATED":
			status.Phase = "failed"
			status.Errors = append(status.Errors, "infrastructure terminated unexpectedly")
			return status, fmt.Errorf("infrastructure terminated")

		case "RUNNING":
			// Update progress and continue polling
			if callback != nil {
				callback(status)
			}

		case "STARTING":
			status.Phase = "provisioning"
			if callback != nil {
				callback(status)
			}
		}

		// Wait before next poll
		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return status, ctx.Err()
		case <-timer.C:
		}
	}
}
