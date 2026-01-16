package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// CoordinatorManager manages the lifecycle of coordinator processes.
type CoordinatorManager struct {
	config      *CoordinatorConfig
	logger      *zap.Logger
	processes   map[string]*CoordinatorProcess
	mu          sync.RWMutex
	databaseURL string // Passed to coordinators that need it
	mqttURL     string // Passed to coordinators
}

// CoordinatorProcess represents a running coordinator process.
type CoordinatorProcess struct {
	Name      string
	Cmd       *exec.Cmd
	Status    ProcessStatus
	StartedAt time.Time
	PID       int
	Retries   int
	Error     error
}

// ProcessStatus represents the status of a coordinator process.
type ProcessStatus string

const (
	// ProcessStatusPending indicates the process hasn't started yet
	ProcessStatusPending ProcessStatus = "pending"
	// ProcessStatusStarting indicates the process is starting
	ProcessStatusStarting ProcessStatus = "starting"
	// ProcessStatusRunning indicates the process is running
	ProcessStatusRunning ProcessStatus = "running"
	// ProcessStatusStopped indicates the process was stopped
	ProcessStatusStopped ProcessStatus = "stopped"
	// ProcessStatusFailed indicates the process failed to start or crashed
	ProcessStatusFailed ProcessStatus = "failed"
)

// NewCoordinatorManager creates a new coordinator manager.
func NewCoordinatorManager(config *CoordinatorConfig, databaseURL, mqttURL string, logger *zap.Logger) *CoordinatorManager {
	return &CoordinatorManager{
		config:      config,
		logger:      logger,
		processes:   make(map[string]*CoordinatorProcess),
		databaseURL: databaseURL,
		mqttURL:     mqttURL,
	}
}

// StartAll starts all coordinators in the configured order.
func (cm *CoordinatorManager) StartAll(ctx context.Context) error {
	cm.logger.Info("Starting all coordinators",
		zap.Int("count", len(cm.config.Order)),
		zap.Bool("fail_fast", cm.config.FailFast))

	for _, coordinatorName := range cm.config.Order {
		if err := cm.Start(ctx, coordinatorName); err != nil {
			if cm.config.FailFast {
				return fmt.Errorf("failed to start %s (fail-fast enabled): %w", coordinatorName, err)
			}
			cm.logger.Error("Failed to start coordinator, continuing",
				zap.String("coordinator", coordinatorName),
				zap.Error(err))
		}

		// Wait for health check before proceeding to next coordinator
		if err := cm.waitForHealthy(ctx, coordinatorName); err != nil {
			if cm.config.FailFast {
				return fmt.Errorf("coordinator %s failed health check (fail-fast enabled): %w", coordinatorName, err)
			}
			cm.logger.Error("Coordinator health check failed, continuing",
				zap.String("coordinator", coordinatorName),
				zap.Error(err))
		}
	}

	cm.logger.Info("All coordinators started successfully")
	return nil
}

// Start starts a single coordinator process.
func (cm *CoordinatorManager) Start(ctx context.Context, coordinatorName string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if already running
	if proc, exists := cm.processes[coordinatorName]; exists {
		if proc.Status == ProcessStatusRunning {
			cm.logger.Info("Coordinator already running",
				zap.String("coordinator", coordinatorName),
				zap.Int("pid", proc.PID))
			return nil
		}
	}

	cm.logger.Info("Starting coordinator",
		zap.String("coordinator", coordinatorName))

	// Build command
	binPath := filepath.Join(cm.config.BinPath, coordinatorName)
	cmd := exec.CommandContext(ctx, binPath)

	// Set up environment variables
	cmd.Env = os.Environ()

	// Pass database URL to coordinators that need it
	if needsDatabaseURL(coordinatorName) {
		cmd.Args = append(cmd.Args, "--database-url", cm.databaseURL)
	}

	// Set log level from environment or default
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	cmd.Args = append(cmd.Args, "--log-level", logLevel)

	// Set up stdout/stderr logging
	cmd.Stdout = &coordinatorLogWriter{
		coordinator: coordinatorName,
		logger:      cm.logger,
		isError:     false,
	}
	cmd.Stderr = &coordinatorLogWriter{
		coordinator: coordinatorName,
		logger:      cm.logger,
		isError:     true,
	}

	// Set process group to allow clean shutdown
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Create process record
	proc := &CoordinatorProcess{
		Name:      coordinatorName,
		Cmd:       cmd,
		Status:    ProcessStatusStarting,
		StartedAt: time.Now(),
		PID:       cmd.Process.Pid,
		Retries:   0,
	}
	cm.processes[coordinatorName] = proc

	cm.logger.Info("Coordinator process started",
		zap.String("coordinator", coordinatorName),
		zap.Int("pid", proc.PID))

	// Monitor process in background
	go cm.monitorProcess(coordinatorName)

	return nil
}

// Stop stops a single coordinator process.
func (cm *CoordinatorManager) Stop(ctx context.Context, coordinatorName string) error {
	cm.mu.Lock()
	proc, exists := cm.processes[coordinatorName]
	cm.mu.Unlock()

	if !exists {
		return fmt.Errorf("coordinator %s not found", coordinatorName)
	}

	if proc.Status == ProcessStatusStopped {
		return nil
	}

	cm.logger.Info("Stopping coordinator",
		zap.String("coordinator", coordinatorName),
		zap.Int("pid", proc.PID))

	// Send SIGTERM for graceful shutdown
	if err := proc.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		cm.logger.Warn("Failed to send SIGTERM, trying SIGKILL",
			zap.String("coordinator", coordinatorName),
			zap.Error(err))
		// Force kill if SIGTERM fails
		if killErr := proc.Cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill process: %w", killErr)
		}
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- proc.Cmd.Wait()
	}()

	select {
	case <-time.After(10 * time.Second):
		// Force kill after timeout
		cm.logger.Warn("Process did not stop gracefully, force killing",
			zap.String("coordinator", coordinatorName))
		if err := proc.Cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to force kill process: %w", err)
		}
		<-done // Wait for process to actually exit
	case err := <-done:
		if err != nil && err.Error() != "signal: terminated" && err.Error() != "signal: killed" {
			cm.logger.Warn("Process exited with error",
				zap.String("coordinator", coordinatorName),
				zap.Error(err))
		}
	case <-ctx.Done():
		// Context cancelled, force kill
		proc.Cmd.Process.Kill()
		return ctx.Err()
	}

	cm.mu.Lock()
	proc.Status = ProcessStatusStopped
	cm.mu.Unlock()

	cm.logger.Info("Coordinator stopped",
		zap.String("coordinator", coordinatorName))

	return nil
}

// StopAll stops all running coordinators in reverse order.
func (cm *CoordinatorManager) StopAll(ctx context.Context) error {
	cm.logger.Info("Stopping all coordinators")

	// Stop in reverse order (respecting dependencies)
	order := make([]string, len(cm.config.Order))
	copy(order, cm.config.Order)

	// Reverse the order
	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}

	var lastErr error
	for _, coordinatorName := range order {
		if err := cm.Stop(ctx, coordinatorName); err != nil {
			cm.logger.Error("Failed to stop coordinator",
				zap.String("coordinator", coordinatorName),
				zap.Error(err))
			lastErr = err
		}
	}

	if lastErr != nil {
		return fmt.Errorf("failed to stop some coordinators: %w", lastErr)
	}

	cm.logger.Info("All coordinators stopped")
	return nil
}

// Restart restarts a coordinator process.
func (cm *CoordinatorManager) Restart(ctx context.Context, coordinatorName string) error {
	cm.logger.Info("Restarting coordinator",
		zap.String("coordinator", coordinatorName))

	if err := cm.Stop(ctx, coordinatorName); err != nil {
		return fmt.Errorf("failed to stop coordinator: %w", err)
	}

	// Brief pause before restart
	time.Sleep(1 * time.Second)

	if err := cm.Start(ctx, coordinatorName); err != nil {
		return fmt.Errorf("failed to start coordinator: %w", err)
	}

	return nil
}

// monitorProcess monitors a coordinator process and handles crashes.
func (cm *CoordinatorManager) monitorProcess(coordinatorName string) {
	cm.mu.RLock()
	proc, exists := cm.processes[coordinatorName]
	cm.mu.RUnlock()

	if !exists {
		return
	}

	// Wait for process to exit
	err := proc.Cmd.Wait()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if intentionally stopped
	if proc.Status == ProcessStatusStopped {
		return
	}

	// Process crashed
	cm.logger.Error("Coordinator process exited unexpectedly",
		zap.String("coordinator", coordinatorName),
		zap.Int("pid", proc.PID),
		zap.Error(err))

	proc.Status = ProcessStatusFailed
	proc.Error = err

	// Attempt restart if retries available
	if proc.Retries < cm.config.MaxStartupRetries {
		proc.Retries++
		cm.logger.Info("Attempting to restart coordinator",
			zap.String("coordinator", coordinatorName),
			zap.Int("retry", proc.Retries),
			zap.Int("max_retries", cm.config.MaxStartupRetries))

		// Release lock before restarting
		cm.mu.Unlock()
		if err := cm.Restart(context.Background(), coordinatorName); err != nil {
			cm.logger.Error("Failed to restart coordinator",
				zap.String("coordinator", coordinatorName),
				zap.Error(err))
		}
		cm.mu.Lock()
	} else {
		cm.logger.Error("Coordinator exceeded maximum restart attempts",
			zap.String("coordinator", coordinatorName),
			zap.Int("retries", proc.Retries))
	}
}

// waitForHealthy waits for a coordinator to report healthy status.
func (cm *CoordinatorManager) waitForHealthy(ctx context.Context, coordinatorName string) error {
	cm.logger.Debug("Waiting for coordinator health check",
		zap.String("coordinator", coordinatorName),
		zap.Duration("timeout", cm.config.StartupTimeout))

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, cm.config.StartupTimeout)
	defer cancel()

	ticker := time.NewTicker(cm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for coordinator to become healthy")
		case <-ticker.C:
			// Check if process is still running
			cm.mu.RLock()
			proc, exists := cm.processes[coordinatorName]
			cm.mu.RUnlock()

			if !exists {
				return fmt.Errorf("coordinator process not found")
			}

			if proc.Status == ProcessStatusFailed {
				return fmt.Errorf("coordinator process failed")
			}

			// In a real implementation, this would check MQTT health topic
			// For now, we just verify the process is running
			if proc.Cmd.Process != nil {
				// Simple check: process exists
				if err := proc.Cmd.Process.Signal(syscall.Signal(0)); err == nil {
					// Process is running, mark as healthy
					cm.mu.Lock()
					proc.Status = ProcessStatusRunning
					cm.mu.Unlock()

					cm.logger.Info("Coordinator is healthy",
						zap.String("coordinator", coordinatorName),
						zap.Duration("startup_time", time.Since(proc.StartedAt)))
					return nil
				}
			}
		}
	}
}

// GetStatus returns the status of all coordinators.
func (cm *CoordinatorManager) GetStatus() map[string]*CoordinatorProcess {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Create a copy to avoid race conditions
	status := make(map[string]*CoordinatorProcess, len(cm.processes))
	for name, proc := range cm.processes {
		procCopy := *proc
		status[name] = &procCopy
	}

	return status
}

// PrintStatus logs the status of all coordinators.
func (cm *CoordinatorManager) PrintStatus() {
	status := cm.GetStatus()

	cm.logger.Info("Coordinator Status",
		zap.Int("total", len(status)))

	for name, proc := range status {
		fields := []zap.Field{
			zap.String("coordinator", name),
			zap.String("status", string(proc.Status)),
		}

		if proc.PID > 0 {
			fields = append(fields, zap.Int("pid", proc.PID))
		}

		if !proc.StartedAt.IsZero() {
			fields = append(fields,
				zap.Time("started_at", proc.StartedAt),
				zap.Duration("uptime", time.Since(proc.StartedAt)))
		}

		if proc.Error != nil {
			fields = append(fields, zap.Error(proc.Error))
		}

		if proc.Retries > 0 {
			fields = append(fields, zap.Int("retries", proc.Retries))
		}

		cm.logger.Info("Coordinator", fields...)
	}
}

// needsDatabaseURL determines if a coordinator needs the database URL parameter.
func needsDatabaseURL(coordinatorName string) bool {
	// All coordinators now load their config from the database
	// except they get the connection string as a bootstrap parameter
	return coordinatorName == "datastore-coordinator" ||
		coordinatorName == "security-coordinator" ||
		coordinatorName == "message-coordinator" ||
		coordinatorName == "application-coordinator" ||
		coordinatorName == "plugin-coordinator" ||
		coordinatorName == "telescope-coordinator" ||
		coordinatorName == "uielement-coordinator"
}

// coordinatorLogWriter writes coordinator logs with proper tagging.
type coordinatorLogWriter struct {
	coordinator string
	logger      *zap.Logger
	isError     bool
}

// Write implements io.Writer interface.
func (w *coordinatorLogWriter) Write(p []byte) (n int, err error) {
	msg := string(p)

	// Remove trailing newline if present
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}

	if w.isError {
		w.logger.Error("Coordinator stderr",
			zap.String("coordinator", w.coordinator),
			zap.String("message", msg))
	} else {
		w.logger.Info("Coordinator stdout",
			zap.String("coordinator", w.coordinator),
			zap.String("message", msg))
	}

	return len(p), nil
}
