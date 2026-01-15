// Package healthcheck provides health check reporting functionality.
package healthcheck

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Reporter publishes health check results.
type Reporter struct {
	engine    *Engine
	publisher PublishFunc
	logger    *zap.Logger
}

// PublishFunc is called when health check results are ready to be published.
type PublishFunc func(ctx context.Context, result *AggregatedResult) error

// NewReporter creates a new health check reporter.
func NewReporter(engine *Engine, publisher PublishFunc, logger *zap.Logger) *Reporter {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Reporter{
		engine:    engine,
		publisher: publisher,
		logger:    logger,
	}
}

// Report runs health checks and publishes the results.
func (r *Reporter) Report(ctx context.Context) error {
	result := r.engine.CheckAll(ctx)

	if r.publisher != nil {
		if err := r.publisher(ctx, result); err != nil {
			r.logger.Error("Failed to publish health check results", zap.Error(err))
			return err
		}
	}

	r.logger.Debug("Health check report published",
		zap.String("status", string(result.OverallStatus)),
		zap.Int("components", len(result.Components)))

	return nil
}

// StartReporting begins periodic health check reporting.
func (r *Reporter) StartReporting(ctx context.Context, interval time.Duration) {
	r.logger.Info("Starting health check reporter", zap.Duration("interval", interval))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Health check reporter stopped")
			return
		case <-ticker.C:
			if err := r.Report(ctx); err != nil {
				r.logger.Error("Health check report failed", zap.Error(err))
			}
		}
	}
}
