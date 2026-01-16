// Package healthcheck implements the health check engine.
package healthcheck

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Engine manages health checks for multiple components.
type Engine struct {
	checkers map[string]Checker
	logger   *zap.Logger
	mu       sync.RWMutex
	interval time.Duration
	stopCh   chan struct{}
	running  bool
}

// NewEngine creates a new health check engine.
func NewEngine(logger *zap.Logger, interval time.Duration) *Engine {
	if logger == nil {
		logger = zap.NewNop()
	}
	if interval == 0 {
		interval = 3 * time.Second
	}

	return &Engine{
		checkers: make(map[string]Checker),
		logger:   logger,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Register adds a health checker to the engine.
func (e *Engine) Register(checker Checker) {
	e.mu.Lock()
	defer e.mu.Unlock()

	name := checker.Name()
	e.checkers[name] = checker
	e.logger.Info("Registered health checker", zap.String("component", name))
}

// Unregister removes a health checker from the engine.
func (e *Engine) Unregister(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.checkers, name)
	e.logger.Info("Unregistered health checker", zap.String("component", name))
}

// CheckAll runs all registered health checks and returns aggregated results.
func (e *Engine) CheckAll(ctx context.Context) *AggregatedResult {
	e.mu.RLock()
	checkers := make(map[string]Checker, len(e.checkers))
	for k, v := range e.checkers {
		checkers[k] = v
	}
	e.mu.RUnlock()

	results := make(map[string]*Result, len(checkers))
	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	for name, checker := range checkers {
		wg.Add(1)
		go func(n string, c Checker) {
			defer wg.Done()

			start := time.Now()
			result := c.Check(ctx)
			result.Duration = time.Since(start)

			resultsMu.Lock()
			results[n] = result
			resultsMu.Unlock()
		}(name, checker)
	}

	wg.Wait()

	overallStatus := DetermineOverallStatus(results)

	return &AggregatedResult{
		OverallStatus: overallStatus,
		Components:    results,
		Timestamp:     time.Now(),
	}
}

// Start begins periodic health checks.
func (e *Engine) Start(ctx context.Context) {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.mu.Unlock()

	e.logger.Info("Starting health check engine", zap.Duration("interval", e.interval))

	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("Health check engine stopped (context)")
			e.mu.Lock()
			e.running = false
			e.mu.Unlock()
			return
		case <-e.stopCh:
			e.logger.Info("Health check engine stopped")
			e.mu.Lock()
			e.running = false
			e.mu.Unlock()
			return
		case <-ticker.C:
			result := e.CheckAll(ctx)
			e.logger.Debug("Health check completed",
				zap.String("status", string(result.OverallStatus)),
				zap.Int("components", len(result.Components)))
		}
	}
}

// Stop stops the periodic health checks.
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	close(e.stopCh)
	e.stopCh = make(chan struct{})
}

// IsRunning returns true if the engine is running.
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}
