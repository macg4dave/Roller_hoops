package discoveryworker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"roller_hoops/core-go/internal/sqlcgen"
)

type fakeQueries struct {
	claimFn  func(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error)
	updateFn func(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error)
	insertFn func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error
}

func (f *fakeQueries) ClaimNextDiscoveryRun(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error) {
	return f.claimFn(ctx, stats)
}

func (f *fakeQueries) UpdateDiscoveryRun(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
	return f.updateFn(ctx, arg)
}

func (f *fakeQueries) InsertDiscoveryRunLog(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error {
	return f.insertFn(ctx, arg)
}

func TestWorker_RunOnce_NoQueuedRuns(t *testing.T) {
	q := &fakeQueries{
		claimFn: func(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error) {
			return sqlcgen.DiscoveryRun{}, pgx.ErrNoRows
		},
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			t.Fatalf("UpdateDiscoveryRun should not be called")
			return sqlcgen.DiscoveryRun{}, nil
		},
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error {
			t.Fatalf("InsertDiscoveryRunLog should not be called")
			return nil
		},
	}

	w := New(zerolog.Nop(), q, Options{PollInterval: 0, RunDelay: 0})
	processed, err := w.runOnce(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if processed {
		t.Fatalf("expected processed=false")
	}
}

func TestWorker_RunOnce_ClaimsAndCompletes(t *testing.T) {
	var (
		seenStarted   bool
		seenCompleted bool
		updatedStatus string
	)

	now := time.Now()
	q := &fakeQueries{
		claimFn: func(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error) {
			if stats == nil || stats["stage"] != "running" {
				t.Fatalf("expected running stats, got %#v", stats)
			}
			return sqlcgen.DiscoveryRun{ID: "run-1", Status: "running", StartedAt: now}, nil
		},
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			updatedStatus = arg.Status
			if arg.Status != "succeeded" {
				t.Fatalf("expected succeeded, got %q", arg.Status)
			}
			if arg.CompletedAt == nil {
				t.Fatalf("expected completed_at set")
			}
			return sqlcgen.DiscoveryRun{ID: arg.ID, Status: arg.Status, StartedAt: now, CompletedAt: arg.CompletedAt}, nil
		},
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error {
			switch arg.Message {
			case "discovery run started":
				seenStarted = true
			case "discovery run completed":
				seenCompleted = true
			}
			return nil
		},
	}

	w := New(zerolog.Nop(), q, Options{PollInterval: 0, RunDelay: 0})
	processed, err := w.runOnce(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !processed {
		t.Fatalf("expected processed=true")
	}
	if updatedStatus != "succeeded" {
		t.Fatalf("expected run to succeed, got %q", updatedStatus)
	}
	if !seenStarted || !seenCompleted {
		t.Fatalf("expected both logs, got started=%v completed=%v", seenStarted, seenCompleted)
	}
}

func TestWorker_RunOnce_FailsRunWhenUpdateFails(t *testing.T) {
	q := &fakeQueries{}

	q.claimFn = func(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error) {
		return sqlcgen.DiscoveryRun{ID: "run-2", Status: "running"}, nil
	}

	updateCalls := 0
	q.updateFn = func(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
		updateCalls++
		if updateCalls == 1 {
			return sqlcgen.DiscoveryRun{}, errors.New("boom")
		}
		if arg.Status != "failed" {
			t.Fatalf("expected failed status on retry, got %q", arg.Status)
		}
		if arg.LastError == nil || *arg.LastError == "" {
			t.Fatalf("expected last_error to be set")
		}
		return sqlcgen.DiscoveryRun{ID: arg.ID, Status: arg.Status}, nil
	}

	q.insertFn = func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error {
		return nil
	}

	w := New(zerolog.Nop(), q, Options{PollInterval: 0, RunDelay: 0})
	processed, err := w.runOnce(context.Background())
	if !processed {
		t.Fatalf("expected processed=true")
	}
	if err == nil {
		t.Fatalf("expected error")
	}
	if updateCalls < 2 {
		t.Fatalf("expected at least two update calls, got %d", updateCalls)
	}
}
