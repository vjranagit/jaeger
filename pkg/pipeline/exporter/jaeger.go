package exporter

import (
	"context"
	"fmt"

	"github.com/vjranagit/jaeger-toolkit/pkg/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// JaegerExporter exports spans to Jaeger backend via gRPC
type JaegerExporter struct {
	name     string
	endpoint string
	conn     *grpc.ClientConn
}

// JaegerConfig configures the Jaeger exporter
type JaegerConfig struct {
	Endpoint string // e.g., "jaeger-collector:14250"
	TLS      bool
}

// NewJaegerExporter creates a new Jaeger exporter
func NewJaegerExporter(name string, config JaegerConfig) *JaegerExporter {
	return &JaegerExporter{
		name:     name,
		endpoint: config.Endpoint,
	}
}

// Export sends spans to Jaeger backend
func (e *JaegerExporter) Export(ctx context.Context, in <-chan *model.Span) error {
	// Establish gRPC connection
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.Dial(e.endpoint, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %w", e.endpoint, err)
	}
	defer conn.Close()

	e.conn = conn

	// Process spans from channel
	for {
		select {
		case span, ok := <-in:
			if !ok {
				// Channel closed
				return nil
			}

			if err := e.sendSpan(ctx, span); err != nil {
				// Log error but continue (would emit metric in production)
				fmt.Printf("Failed to send span: %v\n", err)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// sendSpan sends a single span to Jaeger
func (e *JaegerExporter) sendSpan(ctx context.Context, span *model.Span) error {
	// TODO: Convert span to Jaeger protobuf format and send via gRPC
	// For now, this is a skeleton implementation
	_ = span
	return nil
}

// Name returns the exporter name
func (e *JaegerExporter) Name() string {
	return e.name
}
