package processor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vjranagit/jaeger-toolkit/pkg/model"
)

func TestSamplingProcessorErrorsAlwaysSampled(t *testing.T) {
	config := DefaultSamplingConfig()
	config.BaseSampleRate = 0.0 // Set to 0 to test error-only sampling
	processor := NewSamplingProcessor("test-sampler", config)

	// Create error span
	span := &model.Span{
		TraceID:   model.TraceID{High: 1, Low: 2},
		SpanID:    model.SpanID(123),
		Duration:  50 * time.Millisecond,
		Tags: []model.KeyValue{
			{Key: "error", VType: model.BoolType, VBool: true},
		},
	}

	assert.True(t, processor.shouldSample(span))
}

func TestSamplingProcessorSlowSpansAlwaysSampled(t *testing.T) {
	config := DefaultSamplingConfig()
	config.BaseSampleRate = 0.0
	config.SlowThreshold = 500 * time.Millisecond
	processor := NewSamplingProcessor("test-sampler", config)

	// Create slow span
	span := &model.Span{
		TraceID:  model.TraceID{High: 1, Low: 2},
		SpanID:   model.SpanID(123),
		Duration: 1 * time.Second, // Slow!
	}

	assert.True(t, processor.shouldSample(span))
}

func TestSamplingProcessorBaseSamplingRate(t *testing.T) {
	config := DefaultSamplingConfig()
	config.BaseSampleRate = 1.0 // 100% sampling
	config.AlwaysSampleErrors = false
	processor := NewSamplingProcessor("test-sampler", config)

	sampled := 0
	total := 100

	for i := 0; i < total; i++ {
		span := &model.Span{
			TraceID:  model.TraceID{High: 1, Low: uint64(i)},
			SpanID:   model.SpanID(i),
			Duration: 10 * time.Millisecond,
		}

		if processor.shouldSample(span) {
			sampled++
		}
	}

	// With 100% rate, all should be sampled
	assert.Equal(t, total, sampled)
}

func TestSamplingProcessorAdaptiveRate(t *testing.T) {
	config := DefaultSamplingConfig()
	config.BaseSampleRate = 0.1
	config.AdaptiveWindow = 100 // Small window for testing
	processor := NewSamplingProcessor("test-sampler", config)

	// Simulate high error rate
	for i := 0; i < 50; i++ {
		processor.recordSample(true) // Error
	}
	for i := 0; i < 50; i++ {
		processor.recordSample(false) // No error
	}

	// Error rate is 50%, should increase adaptive rate
	stats := processor.GetStats()
	assert.Greater(t, stats.AdaptiveRate, config.BaseSampleRate)
}

func TestSamplingProcessorProcessChannel(t *testing.T) {
	config := DefaultSamplingConfig()
	config.BaseSampleRate = 0.5
	processor := NewSamplingProcessor("test-sampler", config)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	in := make(chan *model.Span, 10)
	out := processor.Process(ctx, in)

	// Send test spans
	for i := 0; i < 5; i++ {
		in <- &model.Span{
			TraceID: model.TraceID{High: 1, Low: uint64(i)},
			SpanID:  model.SpanID(i),
		}
	}
	close(in)

	// Collect output
	received := 0
	for range out {
		received++
	}

	// Should have sampled some (not all, not none)
	assert.Greater(t, received, 0)
	assert.LessOrEqual(t, received, 5)
}

func TestSamplingProcessorHTTPErrorCodes(t *testing.T) {
	config := DefaultSamplingConfig()
	config.BaseSampleRate = 0.0
	processor := NewSamplingProcessor("test-sampler", config)

	// Create span with HTTP 500 error
	span := &model.Span{
		TraceID: model.TraceID{High: 1, Low: 2},
		SpanID:  model.SpanID(123),
		Tags: []model.KeyValue{
			{Key: "http.status_code", VType: model.Int64Type, VInt64: 500},
		},
	}

	assert.True(t, processor.shouldSample(span))
}
