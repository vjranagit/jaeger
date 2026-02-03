package observability

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetricsCounters(t *testing.T) {
	m := NewMetrics()

	// Record some metrics
	m.RecordSpanReceived()
	m.RecordSpanReceived()
	m.RecordSpanProcessed()
	m.RecordSpanDropped()
	m.RecordSpanExported()
	m.RecordExportError()

	snapshot := m.Snapshot()

	assert.Equal(t, uint64(2), snapshot.SpansReceived)
	assert.Equal(t, uint64(1), snapshot.SpansProcessed)
	assert.Equal(t, uint64(1), snapshot.SpansDropped)
	assert.Equal(t, uint64(1), snapshot.SpansExported)
	assert.Equal(t, uint64(1), snapshot.ExportErrors)
}

func TestMetricsDropRate(t *testing.T) {
	m := NewMetrics()

	for i := 0; i < 100; i++ {
		m.RecordSpanReceived()
	}
	for i := 0; i < 5; i++ {
		m.RecordSpanDropped()
	}

	snapshot := m.Snapshot()
	assert.InDelta(t, 5.0, snapshot.DropRate(), 0.01)
}

func TestMetricsErrorRate(t *testing.T) {
	m := NewMetrics()

	for i := 0; i < 50; i++ {
		m.RecordSpanExported()
	}
	for i := 0; i < 2; i++ {
		m.RecordExportError()
	}

	snapshot := m.Snapshot()
	assert.InDelta(t, 4.0, snapshot.ErrorRate(), 0.01)
}

func TestMetricsLatency(t *testing.T) {
	m := NewMetrics()

	// Record some processing times
	m.RecordProcessingTime(10 * time.Millisecond)
	m.RecordProcessingTime(20 * time.Millisecond)
	m.RecordProcessingTime(30 * time.Millisecond)
	m.RecordProcessingTime(100 * time.Millisecond)

	snapshot := m.Snapshot()

	assert.Greater(t, snapshot.LatencyP50, time.Duration(0))
	assert.Greater(t, snapshot.LatencyP99, snapshot.LatencyP50)
}

func TestMetricsBufferRotation(t *testing.T) {
	m := NewMetrics()
	m.maxSamples = 10 // Small buffer for testing

	// Fill buffer beyond capacity
	for i := 0; i < 20; i++ {
		m.RecordProcessingTime(time.Duration(i) * time.Millisecond)
	}

	m.mu.RLock()
	bufferSize := len(m.processingTimes)
	m.mu.RUnlock()

	assert.LessOrEqual(t, bufferSize, 10)
}
