package processor

import (
	"context"
	"time"

	"github.com/vjranagit/jaeger-toolkit/pkg/model"
)

// BatchProcessor batches spans for efficient processing
type BatchProcessor struct {
	name          string
	timeout       time.Duration
	batchSize     int
	sendBatchSize int
}

// BatchConfig configures the batch processor
type BatchConfig struct {
	Timeout       time.Duration
	BatchSize     int
	SendBatchSize int
}

// DefaultBatchConfig returns default batch configuration
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		Timeout:       1 * time.Second,
		BatchSize:     8192,
		SendBatchSize: 1024,
	}
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(name string, config BatchConfig) *BatchProcessor {
	return &BatchProcessor{
		name:          name,
		timeout:       config.Timeout,
		batchSize:     config.BatchSize,
		sendBatchSize: config.SendBatchSize,
	}
}

// Process batches incoming spans
func (p *BatchProcessor) Process(ctx context.Context, in <-chan *model.Span) <-chan *model.Span {
	out := make(chan *model.Span, p.batchSize)

	go func() {
		defer close(out)

		batch := make([]*model.Span, 0, p.sendBatchSize)
		ticker := time.NewTicker(p.timeout)
		defer ticker.Stop()

		flush := func() {
			for _, span := range batch {
				select {
				case out <- span:
				case <-ctx.Done():
					return
				}
			}
			batch = batch[:0] // Clear batch
		}

		for {
			select {
			case span, ok := <-in:
				if !ok {
					// Input closed, flush remaining batch
					if len(batch) > 0 {
						flush()
					}
					return
				}

				batch = append(batch, span)
				if len(batch) >= p.sendBatchSize {
					flush()
				}

			case <-ticker.C:
				// Timeout, flush batch
				if len(batch) > 0 {
					flush()
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

// Name returns the processor name
func (p *BatchProcessor) Name() string {
	return p.name
}
