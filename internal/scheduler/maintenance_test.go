package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type mockMaintenanceRepo struct {
	mu           sync.Mutex
	purgeCalls   []int
	deletedCount int64
	purgeErr     error
}

func (m *mockMaintenanceRepo) PurgeInactiveGuestUsers(ctx context.Context, olderThanDays int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.purgeCalls = append(m.purgeCalls, olderThanDays)
	if m.purgeErr != nil {
		return 0, m.purgeErr
	}
	return m.deletedCount, nil
}

func TestMaintenanceEngine_Defaults(t *testing.T) {
	engine := NewMaintenanceEngine(nil, 0, 0)
	if engine.olderThanDays != 7 {
		t.Errorf("expected default olderThanDays to be 7, got %d", engine.olderThanDays)
	}
	if engine.interval != 24*time.Hour {
		t.Errorf("expected default interval to be 24h, got %v", engine.interval)
	}
}

func TestMaintenanceEngine_RunPurge_Success(t *testing.T) {
	mockRepo := &mockMaintenanceRepo{deletedCount: 15}
	engine := NewMaintenanceEngine(mockRepo, 7, time.Hour)

	deleted, err := engine.RunPurge(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 15 {
		t.Errorf("expected 15 deleted, got %d", deleted)
	}

	mockRepo.mu.Lock()
	defer mockRepo.mu.Unlock()
	if len(mockRepo.purgeCalls) != 1 || mockRepo.purgeCalls[0] != 7 {
		t.Errorf("expected 1 call with 7 days, got %v", mockRepo.purgeCalls)
	}
}

func TestMaintenanceEngine_RunPurge_NilRepo(t *testing.T) {
	engine := NewMaintenanceEngine(nil, 7, time.Hour)
	deleted, err := engine.RunPurge(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted on nil repo, got %d", deleted)
	}
}

func TestMaintenanceEngine_RunPurge_ErrorPropagated(t *testing.T) {
	expectedErr := errors.New("db connection failure")
	mockRepo := &mockMaintenanceRepo{purgeErr: expectedErr}
	engine := NewMaintenanceEngine(mockRepo, 7, time.Hour)

	deleted, err := engine.RunPurge(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted on error, got %d", deleted)
	}
}

func TestMaintenanceEngine_StartAndContextCancel(t *testing.T) {
	mockRepo := &mockMaintenanceRepo{deletedCount: 3}
	// Fast interval for test
	engine := NewMaintenanceEngine(mockRepo, 7, 20*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		engine.Start(ctx)
		close(done)
	}()

	// Wait for at least initial run + 1 ticker tick
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// success: cleanly exited
	case <-time.After(500 * time.Millisecond):
		t.Fatal("engine.Start did not exit promptly after context cancel")
	}

	mockRepo.mu.Lock()
	defer mockRepo.mu.Unlock()
	if len(mockRepo.purgeCalls) < 2 {
		t.Errorf("expected at least 2 purge calls (initial + 1 tick), got %d", len(mockRepo.purgeCalls))
	}
}

