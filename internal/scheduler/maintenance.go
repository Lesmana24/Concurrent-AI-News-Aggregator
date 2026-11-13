package scheduler

import (
	"context"
	"log/slog"
	"time"
)

// MaintenanceRepository abstracts database maintenance tasks.
type MaintenanceRepository interface {
	PurgeInactiveGuestUsers(ctx context.Context, olderThanDays int) (int64, error)
}

// MaintenanceEngine periodically executes background database hygiene and cleanup tasks.
type MaintenanceEngine struct {
	repo          MaintenanceRepository
	olderThanDays int
	interval      time.Duration
}

// NewMaintenanceEngine creates a new MaintenanceEngine.
// Default olderThanDays is 7, and default interval is 24 hours.
func NewMaintenanceEngine(repo MaintenanceRepository, olderThanDays int, interval time.Duration) *MaintenanceEngine {
	if olderThanDays <= 0 {
		olderThanDays = 7
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &MaintenanceEngine{
		repo:          repo,
		olderThanDays: olderThanDays,
		interval:      interval,
	}
}

// RunPurge executes a single purge cycle of inactive guest accounts.
func (m *MaintenanceEngine) RunPurge(ctx context.Context) (int64, error) {
	if m.repo == nil {
		return 0, nil
	}
	deleted, err := m.repo.PurgeInactiveGuestUsers(ctx, m.olderThanDays)
	if err != nil {
		slog.Error("Database maintenance: failed to purge inactive guest users", slog.String("error", err.Error()))
		return 0, err
	}
	if deleted > 0 {
		slog.Info("Database maintenance: purged inactive guest users", slog.Int64("deleted_count", deleted), slog.Int("older_than_days", m.olderThanDays))
	}
	return deleted, nil
}

// Start begins the maintenance loop, executing once at start and then every interval until ctx is cancelled.
func (m *MaintenanceEngine) Start(ctx context.Context) {
	// Immediate cleanup on worker startup
	_, _ = m.RunPurge(ctx)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Database maintenance worker stopped")
			return
		case <-ticker.C:
			_, _ = m.RunPurge(ctx)
		}
	}
}

