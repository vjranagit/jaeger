package observability

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks pipeline observability metrics
type Metrics struct {
	// Counter metrics
	spansReceived  atomic.Uint64
	spansProcessed atomic.Uint64
	spansDropped   atomic.Uint64
	spansExported  atomic.Uint64
	exportErrors   atomic.Uint64

	// Latency tracking
	mu              sync.RWMutex
	processingTimes []time.Duration
	maxSamples      int
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		maxSamples:      1000,
		processingTimes: make([]time.Duration, 0, 1000),
	}
}

// RecordSpanReceived increments received span counter
func (m *Metrics) RecordSpanReceived() {
	m.spansReceived.Add(1)
}

// RecordSpanProcessed increments processed span counter
func (m *Metrics) RecordSpanProcessed() {
	m.spansProcessed.Add(1)
}

// RecordSpanDropped increments dropped span counter
func (m *Metrics) RecordSpanDropped() {
	m.spansDropped.Add(1)
}

// RecordSpanExported increments exported span counter
func (m *Metrics) RecordSpanExported() {
	m.spansExported.Add(1)
}

// RecordExportError increments export error counter
func (m *Metrics) RecordExportError() {
	m.exportErrors.Add(1)
}

// RecordProcessingTime records span processing latency
func (m *Metrics) RecordProcessingTime(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.processingTimes) >= m.maxSamples {
		// Rotate buffer (keep most recent)
		copy(m.processingTimes, m.processingTimes[1:])
		m.processingTimes = m.processingTimes[:m.maxSamples-1]
	}
	m.processingTimes = append(m.processingTimes, d)
}

// Snapshot returns current metrics snapshot
func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := MetricsSnapshot{
		SpansReceived:  m.spansReceived.Load(),
		SpansProcessed: m.spansProcessed.Load(),
		SpansDropped:   m.spansDropped.Load(),
		SpansExported:  m.spansExported.Load(),
		ExportErrors:   m.exportErrors.Load(),
	}

	// Calculate latency percentiles
	if len(m.processingTimes) > 0 {
		sorted := make([]time.Duration, len(m.processingTimes))
		copy(sorted, m.processingTimes)
		
		// Simple sort for percentile calculation
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[i] > sorted[j] {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}

		snapshot.LatencyP50 = sorted[len(sorted)/2]
		snapshot.LatencyP95 = sorted[int(float64(len(sorted))*0.95)]
		snapshot.LatencyP99 = sorted[int(float64(len(sorted))*0.99)]
	}

	return snapshot
}

// MetricsSnapshot represents a point-in-time metrics view
type MetricsSnapshot struct {
	SpansReceived  uint64
	SpansProcessed uint64
	SpansDropped   uint64
	SpansExported  uint64
	ExportErrors   uint64
	LatencyP50     time.Duration
	LatencyP95     time.Duration
	LatencyP99     time.Duration
}

// DropRate calculates the percentage of dropped spans
func (s MetricsSnapshot) DropRate() float64 {
	if s.SpansReceived == 0 {
		return 0
	}
	return float64(s.SpansDropped) / float64(s.SpansReceived) * 100
}

// ErrorRate calculates the export error rate
func (s MetricsSnapshot) ErrorRate() float64 {
	if s.SpansExported == 0 {
		return 0
	}
	return float64(s.ExportErrors) / float64(s.SpansExported) * 100
}
