package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/vjranagit/jaeger-toolkit/pkg/model"
)

// Receiver accepts telemetry data and emits it on a channel.
// Uses Go 1.21 generics for type safety.
type Receiver[T any] interface {
	Start(ctx context.Context) (<-chan T, error)
	Stop(ctx context.Context) error
	Name() string
}

// Processor transforms telemetry data.
// Reads from input channel and writes to output channel.
type Processor[T any] interface {
	Process(ctx context.Context, in <-chan T) <-chan T
	Name() string
}

// Exporter sends telemetry data to a backend.
type Exporter[T any] interface {
	Export(ctx context.Context, in <-chan T) error
	Name() string
}

// Pipeline orchestrates data flow from receivers through processors to exporters.
// Channel-based architecture (idiomatic Go) vs callback-based (OTel Collector).
type Pipeline[T any] struct {
	name       string
	receiver   Receiver[T]
	processors []Processor[T]
	exporters  []Exporter[T]
	errChan    chan error
	wg         sync.WaitGroup
}

// NewPipeline creates a new pipeline with given components
func NewPipeline[T any](name string, receiver Receiver[T]) *Pipeline[T] {
	return &Pipeline[T]{
		name:       name,
		receiver:   receiver,
		processors: make([]Processor[T], 0),
		exporters:  make([]Exporter[T], 0),
		errChan:    make(chan error, 10),
	}
}

// AddProcessor adds a processor to the pipeline
func (p *Pipeline[T]) AddProcessor(proc Processor[T]) {
	p.processors = append(p.processors, proc)
}

// AddExporter adds an exporter to the pipeline
func (p *Pipeline[T]) AddExporter(exp Exporter[T]) {
	p.exporters = append(p.exporters, exp)
}

// Run starts the pipeline and blocks until context is cancelled
func (p *Pipeline[T]) Run(ctx context.Context) error {
	// Start receiver
	data, err := p.receiver.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start receiver %s: %w", p.receiver.Name(), err)
	}

	// Chain processors
	for _, proc := range p.processors {
		data = proc.Process(ctx, data)
	}

	// Fan-out to exporters
	for _, exp := range p.exporters {
		p.wg.Add(1)
		go func(exporter Exporter[T]) {
			defer p.wg.Done()
			if err := exporter.Export(ctx, data); err != nil {
				p.errChan <- fmt.Errorf("exporter %s failed: %w", exporter.Name(), err)
			}
		}(exp)
	}

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		if err := p.receiver.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop receiver: %w", err)
		}
		p.wg.Wait()
		return ctx.Err()
	case err := <-p.errChan:
		return err
	}
}

// SpanPipeline is a pipeline for spans (convenience type)
type SpanPipeline = Pipeline[*model.Span]

// NewSpanPipeline creates a new span pipeline
func NewSpanPipeline(name string, receiver Receiver[*model.Span]) *SpanPipeline {
	return NewPipeline[*model.Span](name, receiver)
}
