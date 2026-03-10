package api

import (
	"context"
	"log"
	"time"
)

type AutoAdvanceWorkerConfig struct {
	Enabled   bool
	Interval  time.Duration
	BatchSize int
}

type AutoAdvanceWorker struct {
	server *Server
	config AutoAdvanceWorkerConfig
}

func NewAutoAdvanceWorker(server *Server, cfg AutoAdvanceWorkerConfig) *AutoAdvanceWorker {
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 8
	}
	return &AutoAdvanceWorker{server: server, config: cfg}
}

func (w *AutoAdvanceWorker) Run(ctx context.Context) error {
	if w == nil || w.server == nil || !w.config.Enabled {
		return nil
	}
	w.runOnce(ctx)
	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *AutoAdvanceWorker) runOnce(ctx context.Context) {
	if err := w.reconcileOnce(ctx); err != nil {
		log.Printf("auto-advance worker reconcile failed: %v", err)
	}
}

func (w *AutoAdvanceWorker) reconcileOnce(ctx context.Context) error {
	if w == nil || w.server == nil {
		return nil
	}
	candidates, err := w.server.store.ListAutoAdvanceCandidateRuns(ctx, w.config.BatchSize)
	if err != nil {
		return err
	}
	for _, run := range candidates {
		if _, err := w.server.autoAdvancePipeline(ctx, run); err != nil {
			log.Printf("auto-advance worker run %s failed: %v", run.ID, err)
		}
	}
	return nil
}
