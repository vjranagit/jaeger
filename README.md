# Jaeger Toolkit

> Re-implemented features from [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) and [Jaeger Operator](https://github.com/jaegertracing/jaeger-operator) with a unified CLI and modern Go architecture

## What's Different?

This project reimplements key features from two complementary Jaeger ecosystem projects:

### Original Projects

**OpenTelemetry Collector** (1,590 Go files):
- Vendor-neutral telemetry collection framework
- Supports traces, metrics, and logs
- Receiver/Processor/Exporter plugin architecture
- YAML configuration
- Callback-based component interfaces

**Jaeger Operator** (231 Go files):
- Kubernetes operator for Jaeger deployment
- CRD-based declarative configuration
- Reconciliation loop architecture
- Webhook-based sidecar injection

### Our Implementation

**Jaeger Toolkit** - A unified Go CLI tool combining both capabilities:

**Different Architecture:**
- **Go 1.21+ generics** for type-safe pipelines (vs `interface{}`)
- **Channel-based streaming** (vs callback interfaces)
- **HCL configuration** (vs YAML/CRDs)
- **Direct CLI commands** (vs Kubernetes operator)
- **Event-driven design** (vs reconciliation loops)

**Better Developer Experience:**
- Single binary for all operations
- Cleaner configuration syntax
- Type safety at compile time
- Idiomatic Go patterns

## Features
n### Advanced Features (New!)

This fork includes production-ready enhancements beyond the base implementation:

#### 1. Self-Observability Engine
- **Real-time Metrics**: Track spans received, processed, dropped, exported
- **Latency Monitoring**: P50, P95, P99 processing latencies
- **Error Tracking**: Export failures and error rates
- **Drop Rate Analysis**: Measure data loss from backpressure

#### 2. Health Check API
- **HTTP Endpoints**: , ,  for monitoring
- **Kubernetes Integration**: Liveness and readiness probes
- **Smart Status**: Automatic degraded/unhealthy detection based on thresholds
- **Real-time Diagnostics**: JSON metrics export for external systems

#### 3. Adaptive Sampling Processor
- **Error-Aware**: Always keeps error spans (HTTP 5xx, error tags)
- **Latency-Aware**: Always keeps slow requests
- **Adaptive Rate**: Automatically increases sampling during incidents
- **Trace-Consistent**: All spans in a trace sampled together
- **Cost Optimization**: Reduces storage while preserving signal

See [docs/FEATURES.md](docs/FEATURES.md) for detailed documentation.


### Telemetry Pipeline (OpenTelemetry Collector features)

- **Generic Pipeline Framework**: Type-safe receivers, processors, and exporters using Go 1.21 generics
- **Built-in Components**:
  - Receivers: OTLP (gRPC/HTTP)
  - Processors: Batch, Attributes
  - Exporters: Jaeger (gRPC)
- **HCL Configuration**: Ergonomic, type-safe configuration language
- **Channel-based Architecture**: Idiomatic Go concurrency patterns

### Deployment Manager (Jaeger Operator features)

- **Kubernetes Deployment**: Automated Jaeger instance deployment
- **Deployment Strategies**:
  - All-in-one (development)
  - Production (separate collector, query, agent)
  - Streaming (with Kafka buffering)
- **Storage Backends**: Elasticsearch, Cassandra, Kafka, Memory
- **Autoscaling**: HorizontalPodAutoscaler support
- **Ingress Management**: Automatic ingress/route creation

## Installation

```bash
go install github.com/vjranagit/jaeger-toolkit/cmd/jaeger-toolkit@latest
```

Or build from source:

```bash
git clone https://github.com/vjranagit/jaeger-toolkit
cd jaeger-toolkit
go build -o jaeger-toolkit ./cmd/jaeger-toolkit
```

## Usage

### Pipeline Commands

Run a telemetry pipeline:

```bash
jaeger-toolkit pipeline run config.hcl
```

Validate pipeline configuration:

```bash
jaeger-toolkit pipeline validate config.hcl
```

### Deployment Commands

Deploy Jaeger to Kubernetes:

```bash
jaeger-toolkit deploy apply deployment.hcl
```

Plan deployment (dry-run):

```bash
jaeger-toolkit deploy plan deployment.hcl
```

## Configuration Examples

### Pipeline Configuration (HCL)

```hcl
receiver "otlp" "main" {
  grpc {
    endpoint = "0.0.0.0:4317"
  }
  http {
    endpoint = "0.0.0.0:4318"
  }
}

processor "batch" "default" {
  timeout = "1s"
  send_batch_size = 1024
}

exporter "jaeger" "backend" {
  endpoint = "jaeger-collector:14250"
  tls {
    insecure = false
  }
}

pipeline "traces" {
  receivers = [receiver.otlp.main]
  processors = [processor.batch.default]
  exporters = [exporter.jaeger.backend]
}
```

### Deployment Configuration (HCL)

```hcl
deployment "production-jaeger" {
  strategy = "production"

  storage {
    type = "elasticsearch"
    elasticsearch {
      urls = ["https://elasticsearch:9200"]
      index_prefix = "jaeger"
    }
  }

  collector {
    replicas = 5
    autoscale {
      enabled = true
      min_replicas = 3
      max_replicas = 20
      cpu_target = 70
    }
  }

  query {
    replicas = 3
  }

  ingress {
    enabled = true
    host = "jaeger.example.com"
    tls = true
  }
}
```

## Architecture

### Pipeline Architecture

```
┌─────────────┐     ┌────────────┐     ┌──────────┐
│  Receiver   │────►│ Processor  │────►│ Exporter │
│   (OTLP)    │     │  (Batch)   │     │ (Jaeger) │
└─────────────┘     └────────────┘     └──────────┘
       │                   │                  │
       └───────────────────┴──────────────────┘
              Go channels (type-safe)
```

**Key Differences from OTel Collector:**
- Go generics instead of `interface{}`
- Channels instead of callbacks
- HCL instead of YAML
- Simplified component model

### Deployment Architecture

```
┌──────────────────┐
│ jaeger-toolkit   │
│  deploy apply    │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  HCL Parser      │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Template Engine │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Kubernetes API  │
│  (kubectl apply) │
└──────────────────┘
```

**Key Differences from Jaeger Operator:**
- CLI tool instead of Kubernetes operator
- HCL specs instead of CRDs
- Template rendering instead of code generation
- Direct API calls instead of reconciliation loops

## Development History

This project was developed incrementally from 2021-2024 following modern Go best practices:

- **2021**: Foundation and core pipeline framework
- **2022**: Deployment manager and advanced features
- **2023**: Production hardening and optimization
- **2024**: Maintenance and refinements

## Technology Stack

- **Language**: Go 1.21+ (generics, improved error handling)
- **Configuration**: HCL (HashiCorp Configuration Language)
- **CLI Framework**: Cobra
- **Kubernetes Client**: client-go
- **Protocols**: gRPC, HTTP/2

## Acknowledgments

- Original project: [Jaeger](https://github.com/jaegertracing/jaeger)
- Inspiration: [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector)
- Inspiration: [Jaeger Operator](https://github.com/jaegertracing/jaeger-operator)
- Re-implemented by: vjranagit

## License

Apache License 2.0 (same as original Jaeger project)
