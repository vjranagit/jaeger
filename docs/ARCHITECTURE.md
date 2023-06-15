# Architecture

## Overview

Jaeger Toolkit is a unified Go CLI tool that combines telemetry pipeline management with Kubernetes deployment automation. It reimplements key features from OpenTelemetry Collector and Jaeger Operator using modern Go patterns.

## Design Principles

### 1. Type Safety via Generics

Unlike the OpenTelemetry Collector's `interface{}`-based approach, we use Go 1.21+ generics for compile-time type safety:

```go
// Generic pipeline interface
type Receiver[T any] interface {
    Start(ctx context.Context) (<-chan T, error)
    Stop(ctx context.Context) error
}

// Type-safe span receiver
type SpanReceiver = Receiver[*model.Span]
```

### 2. Channel-based Concurrency

Instead of callback-based processing, we use channels for idiomatic Go concurrency:

```go
func (p *Pipeline[T]) Run(ctx context.Context) error {
    data, _ := p.receiver.Start(ctx)

    // Chain processors
    for _, proc := range p.processors {
        data = proc.Process(ctx, data)
    }

    // Fan-out to exporters (concurrent)
    for _, exp := range p.exporters {
        go exp.Export(ctx, data)
    }
}
```

### 3. HCL Configuration

HCL (HashiCorp Configuration Language) provides better ergonomics than YAML:

```hcl
receiver "otlp" "main" {
  grpc {
    endpoint = "0.0.0.0:4317"
  }
}

pipeline "traces" {
  receivers = [receiver.otlp.main]
  processors = [processor.batch.default]
  exporters = [exporter.jaeger.backend]
}
```

### 4. CLI-first Deployment

Instead of a Kubernetes operator, we provide direct CLI commands:

```bash
# Plan deployment (like terraform plan)
jaeger-toolkit deploy plan deployment.hcl

# Apply deployment (like terraform apply)
jaeger-toolkit deploy apply deployment.hcl
```

## Component Architecture

### Pipeline Components

```
┌─────────────────────────────────────────────────────┐
│                  Pipeline Framework                  │
├─────────────────────────────────────────────────────┤
│                                                      │
│  Receiver[T]  →  Processor[T]  →  Exporter[T]       │
│                                                      │
│  - OTLP           - Batch             - Jaeger      │
│  - Jaeger         - Attributes        - Kafka       │
│  - Zipkin         - Filter            - File        │
│                   - Sampling                         │
└─────────────────────────────────────────────────────┘
```

**Key Interfaces:**

```go
type Receiver[T any] interface {
    Start(ctx context.Context) (<-chan T, error)
    Stop(ctx context.Context) error
    Name() string
}

type Processor[T any] interface {
    Process(ctx context.Context, in <-chan T) <-chan T
    Name() string
}

type Exporter[T any] interface {
    Export(ctx context.Context, in <-chan T) error
    Name() string
}
```

### Deployment Components

```
┌─────────────────────────────────────────────────────┐
│               Deployment Manager                     │
├─────────────────────────────────────────────────────┤
│                                                      │
│  Strategy  →  Template  →  Kubernetes API           │
│                                                      │
│  - AllInOne     - Render      - client-go           │
│  - Production   - Validate    - Apply manifests     │
│  - Streaming                                         │
└─────────────────────────────────────────────────────┘
```

## Data Flow

### Telemetry Pipeline

```
Application
    ↓
[OTLP gRPC Receiver]
    ↓
<-chan *model.Span
    ↓
[Batch Processor]  ← Buffers 1024 spans or 1s timeout
    ↓
<-chan *model.Span
    ↓
[Attributes Processor]  ← Enriches with metadata
    ↓
<-chan *model.Span
    ↓
[Jaeger Exporter]  ← gRPC to Jaeger backend
    ↓
Jaeger Collector
```

### Deployment Workflow

```
deployment.hcl
    ↓
[HCL Parser]
    ↓
DeploymentSpec
    ↓
[Strategy Selector]
    ↓
[Template Renderer]
    ↓
Kubernetes Manifests
    ↓
[kubectl apply]
    ↓
Running Pods
```

## Comparison with Original Projects

### vs OpenTelemetry Collector

| Aspect | OTel Collector | Jaeger Toolkit |
|--------|---------------|----------------|
| **Type System** | `interface{}` + casts | Go 1.21 generics |
| **Concurrency** | Callbacks | Channels |
| **Configuration** | YAML | HCL |
| **Data Model** | OTLP protobuf | Native Go structs |
| **Extensibility** | Factory pattern | Generic interfaces |

### vs Jaeger Operator

| Aspect | Jaeger Operator | Jaeger Toolkit |
|--------|----------------|----------------|
| **Runtime** | Kubernetes operator | CLI tool |
| **API** | Custom Resources (CRDs) | HCL files |
| **Reconciliation** | Controller loops | Direct API calls |
| **Sidecar Injection** | Webhooks | Template rendering |
| **Deployment Model** | GitOps (ArgoCD, Flux) | Terraform-style |

## Package Structure

```
github.com/vjranagit/jaeger-toolkit/
├── cmd/
│   └── jaeger-toolkit/        # CLI entry point
├── pkg/
│   ├── model/                 # Span, Trace data structures
│   ├── pipeline/              # Pipeline framework
│   │   ├── receiver/          # OTLP, Jaeger receivers
│   │   ├── processor/         # Batch, Attributes processors
│   │   └── exporter/          # Jaeger, Kafka exporters
│   ├── config/                # HCL configuration parser
│   ├── deployment/            # Kubernetes deployment logic
│   │   ├── strategy/          # AllInOne, Production, Streaming
│   │   ├── storage/           # Elasticsearch, Cassandra, Kafka
│   │   └── template/          # Manifest templates
│   ├── client/                # Kubernetes client wrappers
│   └── util/                  # Shared utilities
├── internal/
│   ├── codec/                 # Protocol encoding/decoding
│   └── metrics/               # Internal observability
├── examples/                  # Sample configurations
├── docs/                      # Documentation
└── tests/                     # Integration tests
```

## Extension Points

### Adding a New Receiver

```go
type MyReceiver struct {
    name     string
    spanChan chan *model.Span
}

func (r *MyReceiver) Start(ctx context.Context) (<-chan *model.Span, error) {
    // Start listening for spans
    return r.spanChan, nil
}

func (r *MyReceiver) Stop(ctx context.Context) error {
    close(r.spanChan)
    return nil
}

func (r *MyReceiver) Name() string {
    return r.name
}
```

### Adding a New Processor

```go
type MyProcessor struct {
    name string
}

func (p *MyProcessor) Process(ctx context.Context, in <-chan *model.Span) <-chan *model.Span {
    out := make(chan *model.Span)

    go func() {
        defer close(out)
        for span := range in {
            // Transform span
            out <- span
        }
    }()

    return out
}

func (p *MyProcessor) Name() string {
    return p.name
}
```

### Adding a New Deployment Strategy

```go
type MyStrategy struct {
    spec *DeploymentSpec
}

func (s *MyStrategy) Generate() ([]runtime.Object, error) {
    // Generate Kubernetes manifests
    return manifests, nil
}
```

## Performance Considerations

### Channel Buffering

- Receivers use buffered channels (1000 spans) to handle bursts
- Processors use unbuffered channels for backpressure
- Exporters read from channels in dedicated goroutines

### Batch Processing

- Default batch size: 1024 spans
