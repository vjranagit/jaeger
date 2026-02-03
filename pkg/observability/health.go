package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents the health state
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck provides HTTP health check endpoint
type HealthCheck struct {
	addr    string
	metrics *Metrics
	server  *http.Server
	mu      sync.RWMutex
	started bool

	// Thresholds for degraded/unhealthy status
	dropRateWarning    float64
	dropRateCritical   float64
	errorRateWarning   float64
	errorRateCritical  float64
}

// HealthCheckConfig configures the health check server
type HealthCheckConfig struct {
	Addr                 string
	DropRateWarning      float64 // % at which status becomes degraded
	DropRateCritical     float64 // % at which status becomes unhealthy
	ErrorRateWarning     float64
	ErrorRateCritical    float64
}

// DefaultHealthCheckConfig returns default configuration
func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		Addr:                 ":8888",
		DropRateWarning:      1.0,  // 1%
		DropRateCritical:     5.0,  // 5%
		ErrorRateWarning:     2.0,  // 2%
		ErrorRateCritical:    10.0, // 10%
	}
}

// NewHealthCheck creates a new health check server
func NewHealthCheck(metrics *Metrics, config HealthCheckConfig) *HealthCheck {
	return &HealthCheck{
		addr:               config.Addr,
		metrics:            metrics,
		dropRateWarning:    config.DropRateWarning,
		dropRateCritical:   config.DropRateCritical,
		errorRateWarning:   config.ErrorRateWarning,
		errorRateCritical:  config.ErrorRateCritical,
	}
}

// Start starts the health check HTTP server
func (h *HealthCheck) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.started {
		return fmt.Errorf("health check already started")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/metrics", h.handleMetrics)
	mux.HandleFunc("/ready", h.handleReady)

	h.server = &http.Server{
		Addr:         h.addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Health check server error: %v\n", err)
		}
	}()

	h.started = true
	return nil
}

// Stop gracefully stops the health check server
func (h *HealthCheck) Stop(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.started {
		return nil
	}

	if err := h.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown health server: %w", err)
	}

	h.started = false
	return nil
}

// handleHealth returns overall health status
func (h *HealthCheck) handleHealth(w http.ResponseWriter, r *http.Request) {
	snapshot := h.metrics.Snapshot()
	status := h.determineStatus(snapshot)

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Metrics:   snapshot,
	}

	w.Header().Set("Content-Type", "application/json")
	
	// Set appropriate HTTP status code
	switch status {
	case HealthStatusHealthy:
		w.WriteHeader(http.StatusOK)
	case HealthStatusDegraded:
		w.WriteHeader(http.StatusOK) // Still operational
	case HealthStatusUnhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// handleMetrics returns detailed metrics in JSON format
func (h *HealthCheck) handleMetrics(w http.ResponseWriter, r *http.Request) {
	snapshot := h.metrics.Snapshot()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(snapshot)
}

// handleReady returns readiness status (for Kubernetes)
func (h *HealthCheck) handleReady(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	ready := h.started
	h.mu.RUnlock()

	if ready {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("not ready"))
	}
}

// determineStatus calculates health status based on metrics
func (h *HealthCheck) determineStatus(snapshot MetricsSnapshot) HealthStatus {
	dropRate := snapshot.DropRate()
	errorRate := snapshot.ErrorRate()

	// Check critical thresholds
	if dropRate >= h.dropRateCritical || errorRate >= h.errorRateCritical {
		return HealthStatusUnhealthy
	}

	// Check warning thresholds
	if dropRate >= h.dropRateWarning || errorRate >= h.errorRateWarning {
		return HealthStatusDegraded
	}

	return HealthStatusHealthy
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    HealthStatus    `json:"status"`
	Timestamp time.Time       `json:"timestamp"`
	Metrics   MetricsSnapshot `json:"metrics"`
}
