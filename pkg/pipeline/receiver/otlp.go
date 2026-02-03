package receiver

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/vjranagit/jaeger-toolkit/pkg/model"
	"google.golang.org/grpc"
)

// OTLPReceiver receives spans via OTLP gRPC protocol
type OTLPReceiver struct {
	name     string
	endpoint string
	server   *grpc.Server
	spanChan chan *model.Span
	mu       sync.Mutex
	started  bool
}

// OTLPConfig configures the OTLP receiver
type OTLPConfig struct {
	Endpoint string // e.g., "0.0.0.0:4317"
}

// NewOTLPReceiver creates a new OTLP receiver
func NewOTLPReceiver(name string, config OTLPConfig) *OTLPReceiver {
	return &OTLPReceiver{
		name:     name,
		endpoint: config.Endpoint,
		spanChan: make(chan *model.Span, 1000), // Buffered channel
	}
}

// Start starts the gRPC server and returns the span channel
func (r *OTLPReceiver) Start(ctx context.Context) (<-chan *model.Span, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return nil, fmt.Errorf("receiver already started")
	}

	listener, err := net.Listen("tcp", r.endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", r.endpoint, err)
	}

	r.server = grpc.NewServer()
	// TODO: Register OTLP trace service handler
	// For now, this is a skeleton implementation

	go func() {
		if err := r.server.Serve(listener); err != nil {
			// Log error (would use structured logging in production)
			fmt.Printf("gRPC server error: %v\n", err)
		}
	}()

	r.started = true
	return r.spanChan, nil
}

// Stop gracefully stops the receiver
func (r *OTLPReceiver) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return nil
	}

	r.server.GracefulStop()
	close(r.spanChan)
	r.started = false
	return nil
}

// Name returns the receiver name
func (r *OTLPReceiver) Name() string {
	return r.name
}

// SubmitSpan is called by the gRPC handler to submit spans
func (r *OTLPReceiver) SubmitSpan(span *model.Span) {
	select {
	case r.spanChan <- span:
	default:
		// Channel full, drop span (would emit metric in production)
		fmt.Printf("Warning: span channel full, dropping span\n")
	}
}
