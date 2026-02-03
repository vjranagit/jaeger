package processor

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/vjranagit/jaeger-toolkit/pkg/model"
)

// SamplingProcessor implements adaptive sampling based on span characteristics
type SamplingProcessor struct {
	name string
	
	// Base sampling rate (0.0 - 1.0)
	baseSampleRate float64
	
	// Always sample errors and slow requests
	alwaysSampleErrors bool
	slowThreshold      time.Duration
	
	// Adaptive sampling state
	mu                sync.RWMutex
	recentErrors      int
	recentTotal       int
	adaptiveRate      float64
	adaptiveWindow    int
	
	rng *rand.Rand
}

// SamplingConfig configures the sampling processor
type SamplingConfig struct {
	BaseSampleRate     float64       // Base probability (0.0 - 1.0)
	AlwaysSampleErrors bool          // Always keep error spans
	SlowThreshold      time.Duration // Always keep spans slower than this
	AdaptiveWindow     int           // Number of spans to track for adaptation
}

// DefaultSamplingConfig returns sensible defaults
func DefaultSamplingConfig() SamplingConfig {
	return SamplingConfig{
		BaseSampleRate:     0.1,               // 10% baseline
		AlwaysSampleErrors: true,              // Keep all errors
		SlowThreshold:      1 * time.Second,   // Keep slow spans
		AdaptiveWindow:     1000,              // Adapt over 1k spans
	}
}

// NewSamplingProcessor creates a new adaptive sampling processor
func NewSamplingProcessor(name string, config SamplingConfig) *SamplingProcessor {
	return &SamplingProcessor{
		name:               name,
		baseSampleRate:     config.BaseSampleRate,
		alwaysSampleErrors: config.AlwaysSampleErrors,
		slowThreshold:      config.SlowThreshold,
		adaptiveRate:       config.BaseSampleRate,
		adaptiveWindow:     config.AdaptiveWindow,
		rng:                rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Process applies adaptive sampling to spans
func (p *SamplingProcessor) Process(ctx context.Context, in <-chan *model.Span) <-chan *model.Span {
	out := make(chan *model.Span, 100)

	go func() {
		defer close(out)

		for {
			select {
			case span, ok := <-in:
				if !ok {
					return
				}

				if p.shouldSample(span) {
					select {
					case out <- span:
					case <-ctx.Done():
						return
					}
				}
				// Else: drop span (sampled out)

			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

// shouldSample determines if a span should be kept
func (p *SamplingProcessor) shouldSample(span *model.Span) bool {
	// Priority 1: Always sample errors if configured
	if p.alwaysSampleErrors && p.isError(span) {
		p.recordSample(true)
		return true
	}

	// Priority 2: Always sample slow requests
	if span.Duration >= p.slowThreshold {
		p.recordSample(false)
		return true
	}

	// Priority 3: Adaptive sampling based on recent error rate
	rate := p.getAdaptiveRate()
	
	// Use deterministic sampling based on trace ID for consistency
	// This ensures all spans in a trace are sampled together
	decision := p.deterministicSample(span.TraceID, rate)
	
	p.recordSample(p.isError(span))
	
	return decision
}

// deterministicSample uses trace ID for consistent sampling decisions
func (p *SamplingProcessor) deterministicSample(traceID model.TraceID, rate float64) bool {
	// Use trace ID low bits for deterministic decision
	// This ensures all spans in same trace get same decision
	threshold := uint64(float64(^uint64(0)) * rate)
	return traceID.Low <= threshold
}

// isError checks if span represents an error
func (p *SamplingProcessor) isError(span *model.Span) bool {
	for _, tag := range span.Tags {
		if tag.Key == "error" && tag.VType == model.BoolType && tag.VBool {
			return true
		}
		if tag.Key == "http.status_code" && tag.VType == model.Int64Type && tag.VInt64 >= 500 {
			return true
		}
	}
	return false
}

// recordSample updates adaptive sampling state
func (p *SamplingProcessor) recordSample(isError bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.recentTotal++
	if isError {
		p.recentErrors++
	}

	// Recalculate adaptive rate periodically
	if p.recentTotal >= p.adaptiveWindow {
		errorRate := float64(p.recentErrors) / float64(p.recentTotal)

		// Increase sampling if error rate is high
		if errorRate > 0.05 { // More than 5% errors
			p.adaptiveRate = min(1.0, p.baseSampleRate*2.0)
		} else if errorRate > 0.01 { // More than 1% errors
			p.adaptiveRate = min(1.0, p.baseSampleRate*1.5)
		} else {
			p.adaptiveRate = p.baseSampleRate
		}

		// Reset counters
		p.recentErrors = 0
		p.recentTotal = 0
	}
}

// getAdaptiveRate returns current adaptive sampling rate
func (p *SamplingProcessor) getAdaptiveRate() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.adaptiveRate
}

// Name returns the processor name
func (p *SamplingProcessor) Name() string {
	return p.name
}

// GetStats returns current sampling statistics
func (p *SamplingProcessor) GetStats() SamplingStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return SamplingStats{
		BaseSampleRate:   p.baseSampleRate,
		AdaptiveRate:     p.adaptiveRate,
		RecentErrorCount: p.recentErrors,
		RecentTotalCount: p.recentTotal,
	}
}

// SamplingStats represents sampling statistics
type SamplingStats struct {
	BaseSampleRate   float64
	AdaptiveRate     float64
	RecentErrorCount int
	RecentTotalCount int
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
