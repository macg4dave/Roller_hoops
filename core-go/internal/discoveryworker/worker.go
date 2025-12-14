package discoveryworker

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"roller_hoops/core-go/internal/sqlcgen"
)

// Queries is the minimal DB interface the discovery worker needs.
//
// NOTE: core-go uses sqlc for DB access. *sqlcgen.Queries satisfies this.
type Queries interface {
	ClaimNextDiscoveryRun(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error)
	UpdateDiscoveryRun(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error)
	InsertDiscoveryRunLog(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error
}

type Worker struct {
	log          zerolog.Logger
	q            Queries
	pollInterval time.Duration
	runDelay     time.Duration
}

type Options struct {
	PollInterval time.Duration
	RunDelay     time.Duration
}

func New(log zerolog.Logger, q Queries, opts Options) *Worker {
	pi := opts.PollInterval
	if pi <= 0 {
		pi = 400 * time.Millisecond
	}
	rd := opts.RunDelay
	if rd < 0 {
		rd = 0
	}

	return &Worker{
		log:          log,
		q:            q,
		pollInterval: pi,
		runDelay:     rd,
	}
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil || w.q == nil {
		return
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = w.runOnce(ctx)
		}
	}
}

func (w *Worker) runOnce(ctx context.Context) (bool, error) {
	// Claim a run.
	run, err := w.q.ClaimNextDiscoveryRun(ctx, map[string]any{"stage": "running"})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		w.log.Error().Err(err).Msg("discovery worker failed to claim next run")
		return false, err
	}

	w.log.Info().Str("run_id", run.ID).Msg("discovery run claimed")

	// Execute (currently a minimal stub that just transitions state and writes logs).
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := w.q.InsertDiscoveryRunLog(execCtx, sqlcgen.InsertDiscoveryRunLogParams{
		RunID:   run.ID,
		Level:   "info",
		Message: "discovery run started",
	}); err != nil {
		w.log.Warn().Err(err).Str("run_id", run.ID).Msg("failed to write discovery start log")
	}

	if w.runDelay > 0 {
		t := time.NewTimer(w.runDelay)
		select {
		case <-execCtx.Done():
			t.Stop()
			return true, execCtx.Err()
		case <-t.C:
		}
	}

	completedAt := time.Now()
	if _, err := w.q.UpdateDiscoveryRun(execCtx, sqlcgen.UpdateDiscoveryRunParams{
		ID:          run.ID,
		Status:      "succeeded",
		Stats:       map[string]any{"stage": "completed", "devices_seen": 0},
		CompletedAt: &completedAt,
		LastError:   nil,
	}); err != nil {
		w.log.Error().Err(err).Str("run_id", run.ID).Msg("failed to mark discovery run succeeded")

		msg := err.Error()
		_ = w.failRun(execCtx, run.ID, msg)
		return true, err
	}

	if err := w.q.InsertDiscoveryRunLog(execCtx, sqlcgen.InsertDiscoveryRunLogParams{
		RunID:   run.ID,
		Level:   "info",
		Message: "discovery run completed",
	}); err != nil {
		w.log.Warn().Err(err).Str("run_id", run.ID).Msg("failed to write discovery completion log")
	}

	return true, nil
}

func (w *Worker) failRun(ctx context.Context, runID string, errMsg string) error {
	completedAt := time.Now()
	lastErr := errMsg
	_, err := w.q.UpdateDiscoveryRun(ctx, sqlcgen.UpdateDiscoveryRunParams{
		ID:          runID,
		Status:      "failed",
		Stats:       map[string]any{"stage": "failed"},
		CompletedAt: &completedAt,
		LastError:   &lastErr,
	})
	if err != nil {
		w.log.Error().Err(err).Str("run_id", runID).Msg("failed to mark discovery run failed")
		return err
	}

	_ = w.q.InsertDiscoveryRunLog(ctx, sqlcgen.InsertDiscoveryRunLogParams{
		RunID:   runID,
		Level:   "error",
		Message: "discovery run failed: " + errMsg,
	})

	return nil
}
