package proxy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConnectionPool manages a pool of device proxies with health monitoring and failover.
// This provides high availability by maintaining multiple backend connections and
// automatically routing requests to healthy backends.
//
// The pool supports multiple strategies:
// - Primary/backup: Use primary, failover to backup on failure
// - Round-robin: Distribute load across all healthy backends
// - Least-latency: Route to the backend with lowest latency
type ConnectionPool struct {
	// config contains pool configuration
	config *PoolConfig

	// logger provides structured logging
	logger *zap.Logger

	// proxies is the list of managed proxy instances
	proxies []DeviceProxy

	// proxyStates tracks health state for each proxy
	proxyStates []*ProxyState

	// mu protects access to pool state
	mu sync.RWMutex

	// currentIndex tracks the next proxy to use for round-robin
	currentIndex int

	// stopChan signals goroutines to stop
	stopChan chan struct{}

	// wg tracks active goroutines
	wg sync.WaitGroup

	// metrics tracks pool-level metrics
	metrics PoolMetrics
}

// PoolConfig configures the connection pool behavior.
type PoolConfig struct {
	// DeviceType is the ASCOM device type
	DeviceType string

	// DeviceNumber is the device instance number
	DeviceNumber int

	// Strategy determines how requests are routed
	// Options: "primary", "round-robin", "least-latency"
	Strategy string

	// HealthCheckInterval is how often to check backend health
	HealthCheckInterval time.Duration

	// HealthCheckTimeout is the timeout for health checks
	HealthCheckTimeout time.Duration

	// FailureThreshold is how many consecutive failures before marking unhealthy
	FailureThreshold int

	// RecoveryThreshold is how many consecutive successes before marking healthy
	RecoveryThreshold int

	// MinHealthyBackends is the minimum number of healthy backends required
	MinHealthyBackends int
}

// ProxyState tracks the health and performance state of a single proxy.
type ProxyState struct {
	// Proxy is the managed proxy instance
	Proxy DeviceProxy

	// Healthy indicates if the proxy is currently healthy
	Healthy bool

	// ConsecutiveFailures counts recent failures
	ConsecutiveFailures int

	// ConsecutiveSuccesses counts recent successes
	ConsecutiveSuccesses int

	// LastHealthCheck is when the health check was last performed
	LastHealthCheck time.Time

	// LastHealthCheckError is the most recent health check error
	LastHealthCheckError error

	// mu protects state updates
	mu sync.RWMutex
}

// PoolMetrics tracks overall pool performance.
type PoolMetrics struct {
	// TotalBackends is the total number of configured backends
	TotalBackends int

	// HealthyBackends is the current number of healthy backends
	HealthyBackends int

	// TotalRequests is the total number of requests routed
	TotalRequests int64

	// FailedRequests is the number of requests that failed
	FailedRequests int64

	// LastHealthCheckTime is when health checks were last performed
	LastHealthCheckTime time.Time
}

// NewConnectionPool creates a new connection pool with the given proxies.
//
// Parameters:
//   - config: Pool configuration
//   - proxies: List of proxy instances to manage
//   - logger: Structured logger (if nil, a no-op logger is used)
//
// Returns a configured ConnectionPool ready to start.
func NewConnectionPool(config *PoolConfig, proxies []DeviceProxy, logger *zap.Logger) (*ConnectionPool, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Validate configuration
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}
	if len(proxies) == 0 {
		return nil, errors.New("at least one proxy is required")
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	if config.HealthCheckTimeout == 0 {
		config.HealthCheckTimeout = 10 * time.Second
	}
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 3
	}
	if config.RecoveryThreshold == 0 {
		config.RecoveryThreshold = 2
	}
	if config.MinHealthyBackends == 0 {
		config.MinHealthyBackends = 1
	}
	if config.Strategy == "" {
		config.Strategy = "primary"
	}

	// Initialize proxy states
	proxyStates := make([]*ProxyState, len(proxies))
	for i, proxy := range proxies {
		proxyStates[i] = &ProxyState{
			Proxy:   proxy,
			Healthy: false, // Will be determined by initial health check
		}
	}

	pool := &ConnectionPool{
		config:       config,
		logger:       logger.With(zap.String("component", "connection_pool")),
		proxies:      proxies,
		proxyStates:  proxyStates,
		currentIndex: 0,
		stopChan:     make(chan struct{}),
		metrics: PoolMetrics{
			TotalBackends: len(proxies),
		},
	}

	return pool, nil
}

// Start begins pool operations including health monitoring.
// This connects all backends and starts the health check goroutine.
func (p *ConnectionPool) Start(ctx context.Context) error {
	p.logger.Info("Starting connection pool",
		zap.Int("backends", len(p.proxies)),
		zap.String("strategy", p.config.Strategy))

	// Connect all proxies
	for i, proxy := range p.proxies {
		if err := proxy.Connect(ctx); err != nil {
			p.logger.Warn("Failed to connect proxy",
				zap.Int("index", i),
				zap.Error(err))
			// Don't fail the entire pool if one proxy fails
			continue
		}
		p.proxyStates[i].Healthy = true
	}

	// Perform initial health check
	p.performHealthChecks(ctx)

	// Verify we have minimum healthy backends
	if p.metrics.HealthyBackends < p.config.MinHealthyBackends {
		return fmt.Errorf("insufficient healthy backends: have %d, need %d",
			p.metrics.HealthyBackends, p.config.MinHealthyBackends)
	}

	// Start health monitoring goroutine
	p.wg.Add(1)
	go p.healthMonitor()

	p.logger.Info("Connection pool started",
		zap.Int("healthy_backends", p.metrics.HealthyBackends))

	return nil
}

// Stop gracefully shuts down the connection pool.
// This disconnects all proxies and stops health monitoring.
func (p *ConnectionPool) Stop(ctx context.Context) error {
	p.logger.Info("Stopping connection pool")

	// Signal health monitor to stop
	close(p.stopChan)

	// Wait for goroutines to finish
	p.wg.Wait()

	// Disconnect all proxies
	for i, proxy := range p.proxies {
		if err := proxy.Disconnect(ctx); err != nil {
			p.logger.Warn("Failed to disconnect proxy",
				zap.Int("index", i),
				zap.Error(err))
		}
	}

	p.logger.Info("Connection pool stopped")
	return nil
}

// Get executes a GET request through the pool.
// This selects a healthy backend based on the routing strategy.
func (p *ConnectionPool) Get(ctx context.Context, method string, params map[string]string) (interface{}, error) {
	p.metrics.TotalRequests++

	// Select a proxy using the configured strategy
	proxy, stateIndex, err := p.selectProxy()
	if err != nil {
		p.metrics.FailedRequests++
		return nil, err
	}

	// Execute the request
	result, err := proxy.Get(ctx, method, params)

	// Update proxy state based on result
	p.updateProxyState(stateIndex, err == nil)

	if err != nil {
		p.metrics.FailedRequests++

		// Try failover if primary failed and we're using primary strategy
		if p.config.Strategy == "primary" && stateIndex == 0 {
			p.logger.Warn("Primary proxy failed, attempting failover")
			proxy, stateIndex, err := p.selectBackupProxy()
			if err != nil {
				return nil, fmt.Errorf("failover failed: %w", err)
			}

			result, err = proxy.Get(ctx, method, params)
			p.updateProxyState(stateIndex, err == nil)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return result, nil
}

// Put executes a PUT request through the pool.
// This selects a healthy backend based on the routing strategy.
func (p *ConnectionPool) Put(ctx context.Context, method string, params map[string]string) (interface{}, error) {
	p.metrics.TotalRequests++

	// Select a proxy using the configured strategy
	proxy, stateIndex, err := p.selectProxy()
	if err != nil {
		p.metrics.FailedRequests++
		return nil, err
	}

	// Execute the request
	result, err := proxy.Put(ctx, method, params)

	// Update proxy state based on result
	p.updateProxyState(stateIndex, err == nil)

	if err != nil {
		p.metrics.FailedRequests++

		// Try failover if primary failed and we're using primary strategy
		if p.config.Strategy == "primary" && stateIndex == 0 {
			p.logger.Warn("Primary proxy failed, attempting failover")
			proxy, stateIndex, err := p.selectBackupProxy()
			if err != nil {
				return nil, fmt.Errorf("failover failed: %w", err)
			}

			result, err = proxy.Put(ctx, method, params)
			p.updateProxyState(stateIndex, err == nil)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return result, nil
}

// GetMetrics returns current pool metrics.
func (p *ConnectionPool) GetMetrics() *PoolMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	metricsCopy := p.metrics
	return &metricsCopy
}

// selectProxy selects a healthy proxy based on the configured strategy.
func (p *ConnectionPool) selectProxy() (DeviceProxy, int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	switch p.config.Strategy {
	case "primary":
		// Use the first healthy proxy
		for i, state := range p.proxyStates {
			if state.isHealthy() {
				return state.Proxy, i, nil
			}
		}

	case "round-robin":
		// Try each proxy in sequence until we find a healthy one
		for i := 0; i < len(p.proxyStates); i++ {
			p.currentIndex = (p.currentIndex + 1) % len(p.proxyStates)
			state := p.proxyStates[p.currentIndex]
			if state.isHealthy() {
				return state.Proxy, p.currentIndex, nil
			}
		}

	case "least-latency":
		// Find the healthy proxy with the lowest latency
		var bestProxy DeviceProxy
		var bestIndex int = -1
		var bestLatency float64 = 0

		for i, state := range p.proxyStates {
			if !state.isHealthy() {
				continue
			}

			metrics := state.Proxy.GetMetrics()
			if bestIndex == -1 || metrics.AverageLatency < bestLatency {
				bestProxy = state.Proxy
				bestIndex = i
				bestLatency = metrics.AverageLatency
			}
		}

		if bestIndex != -1 {
			return bestProxy, bestIndex, nil
		}
	}

	return nil, -1, ErrBackendUnavailable
}

// selectBackupProxy selects a backup proxy (not the primary).
// This is used for failover in primary strategy.
func (p *ConnectionPool) selectBackupProxy() (DeviceProxy, int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Start from index 1 (skip primary)
	for i := 1; i < len(p.proxyStates); i++ {
		if p.proxyStates[i].isHealthy() {
			return p.proxyStates[i].Proxy, i, nil
		}
	}

	return nil, -1, ErrBackendUnavailable
}

// updateProxyState updates the health state of a proxy based on operation result.
func (p *ConnectionPool) updateProxyState(index int, success bool) {
	if index < 0 || index >= len(p.proxyStates) {
		return
	}

	state := p.proxyStates[index]
	state.mu.Lock()
	defer state.mu.Unlock()

	if success {
		state.ConsecutiveSuccesses++
		state.ConsecutiveFailures = 0

		// Mark healthy if we've reached recovery threshold
		if !state.Healthy && state.ConsecutiveSuccesses >= p.config.RecoveryThreshold {
			state.Healthy = true
			p.logger.Info("Proxy recovered",
				zap.Int("index", index),
				zap.Int("consecutive_successes", state.ConsecutiveSuccesses))
		}
	} else {
		state.ConsecutiveFailures++
		state.ConsecutiveSuccesses = 0

		// Mark unhealthy if we've reached failure threshold
		if state.Healthy && state.ConsecutiveFailures >= p.config.FailureThreshold {
			state.Healthy = false
			p.logger.Warn("Proxy marked unhealthy",
				zap.Int("index", index),
				zap.Int("consecutive_failures", state.ConsecutiveFailures))
		}
	}
}

// healthMonitor periodically checks the health of all proxies.
func (p *ConnectionPool) healthMonitor() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), p.config.HealthCheckTimeout)
			p.performHealthChecks(ctx)
			cancel()

		case <-p.stopChan:
			p.logger.Debug("Stopping health monitor")
			return
		}
	}
}

// performHealthChecks executes health checks on all proxies.
func (p *ConnectionPool) performHealthChecks(ctx context.Context) {
	p.logger.Debug("Performing health checks")

	healthyCount := 0

	for i, state := range p.proxyStates {
		state.mu.Lock()

		// Perform health check
		err := state.Proxy.HealthCheck(ctx)
		state.LastHealthCheck = time.Now()
		state.LastHealthCheckError = err

		if err == nil {
			state.ConsecutiveSuccesses++
			state.ConsecutiveFailures = 0

			// Mark healthy if we've reached recovery threshold
			if !state.Healthy && state.ConsecutiveSuccesses >= p.config.RecoveryThreshold {
				state.Healthy = true
				p.logger.Info("Proxy recovered via health check",
					zap.Int("index", i))
			}

			if state.Healthy {
				healthyCount++
			}
		} else {
			state.ConsecutiveFailures++
			state.ConsecutiveSuccesses = 0

			// Mark unhealthy if we've reached failure threshold
			if state.Healthy && state.ConsecutiveFailures >= p.config.FailureThreshold {
				state.Healthy = false
				p.logger.Warn("Proxy marked unhealthy via health check",
					zap.Int("index", i),
					zap.Error(err))
			}
		}

		state.mu.Unlock()
	}

	// Update pool metrics
	p.mu.Lock()
	p.metrics.HealthyBackends = healthyCount
	p.metrics.LastHealthCheckTime = time.Now()
	p.mu.Unlock()

	p.logger.Debug("Health check completed",
		zap.Int("healthy", healthyCount),
		zap.Int("total", len(p.proxyStates)))

	// Log warning if we're below minimum healthy backends
	if healthyCount < p.config.MinHealthyBackends {
		p.logger.Error("Insufficient healthy backends",
			zap.Int("healthy", healthyCount),
			zap.Int("minimum", p.config.MinHealthyBackends))
	}
}

// isHealthy returns the current health status of the proxy state.
func (ps *ProxyState) isHealthy() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.Healthy
}
