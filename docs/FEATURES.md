# Advanced Features

This document describes the advanced features implemented in Jaeger Toolkit beyond the base reimplementation.

## 1. Self-Observability Engine

### Overview
The pipeline monitors its own health and performance, providing real-time metrics about span processing.

### Features
- **Metrics Tracking**: Automatically tracks spans received, processed, dropped, and exported
- **Latency Monitoring**: Records p50, p95, and p99 processing latencies
- **Error Tracking**: Monitors export failures and error rates
- **Drop Rate Calculation**: Measures data loss due to backpressure

### Usage

```go
import "github.com/vjranagit/jaeger-toolkit/pkg/observability"

// Create metrics tracker
metrics := observability.NewMetrics()

// Record events
metrics.RecordSpanReceived()
metrics.RecordSpanProcessed()
metrics.RecordProcessingTime(5 * time.Millisecond)

// Get snapshot
snapshot := metrics.Snapshot()
fmt.Printf("Drop rate: %.2f%%\n", snapshot.DropRate())
fmt.Printf("P95 latency: %v\n", snapshot.LatencyP95)
```

### Metrics Exposed
- `spans_received`: Total spans received
- `spans_processed`: Total spans successfully processed
- `spans_dropped`: Total spans dropped (backpressure)
- `spans_exported`: Total spans exported to backends
- `export_errors`: Total export failures
- `latency_p50/p95/p99`: Processing latency percentiles

## 2. Health Check API

### Overview
HTTP endpoints for monitoring pipeline health, suitable for Kubernetes probes and load balancers.

### Endpoints

#### `/health` - Overall Health Status
Returns health status based on drop rates and error rates:
- `healthy`: Normal operation
- `degraded`: Warning thresholds exceeded
- `unhealthy`: Critical thresholds exceeded

Example response:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-18T12:00:00Z",
  "metrics": {
    "spansReceived": 1000000,
    "spansProcessed": 999500,
    "spansDropped": 500,
    "spansExported": 999500,
    "exportErrors": 10,
    "latencyP50": "2ms",
    "latencyP95": "15ms",
    "latencyP99": "50ms"
  }
}
```

#### `/metrics` - Detailed Metrics
Returns raw metrics in JSON format for external monitoring systems.

#### `/ready` - Readiness Probe
Returns 200 OK when pipeline is ready to accept traffic (for Kubernetes).

### Configuration

```go
config := observability.DefaultHealthCheckConfig()
config.Addr = ":8888"
config.DropRateWarning = 1.0   // 1% drop rate triggers "degraded"
config.DropRateCritical = 5.0  // 5% drop rate triggers "unhealthy"

healthCheck := observability.NewHealthCheck(metrics, config)
healthCheck.Start(ctx)
```

### Kubernetes Integration

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8888
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /ready
    port: 8888
  initialDelaySeconds: 5
  periodSeconds: 10
```

## 3. Adaptive Sampling Processor

### Overview
Intelligent span sampling that adapts to system conditions, keeping important spans while reducing volume.

### Features
- **Error-Aware Sampling**: Always keeps error spans
- **Latency-Aware Sampling**: Always keeps slow requests
- **Adaptive Rate Adjustment**: Increases sampling rate when error rate rises
- **Trace-Consistent Sampling**: All spans in a trace get the same decision

### Algorithm

1. **Priority 1**: Always sample if span has error tag or HTTP 5xx status
2. **Priority 2**: Always sample if duration exceeds slow threshold
3. **Priority 3**: Deterministic sampling based on trace ID
4. **Adaptation**: Monitors recent error rate (per 1000 spans) and adjusts:
   - Error rate > 5%: Double sampling rate
   - Error rate > 1%: Increase sampling rate by 50%
   - Error rate < 1%: Use base sampling rate

### Usage

```go
import "github.com/vjranagit/jaeger-toolkit/pkg/pipeline/processor"

config := processor.DefaultSamplingConfig()
config.BaseSampleRate = 0.1            // 10% baseline
config.AlwaysSampleErrors = true       // Keep all errors
config.SlowThreshold = 500 * time.Millisecond

sampler := processor.NewSamplingProcessor("adaptive-sampler", config)

// Add to pipeline
pipeline.AddProcessor(sampler)

// Get statistics
stats := sampler.GetStats()
fmt.Printf("Current adaptive rate: %.2f%%\n", stats.AdaptiveRate*100)
```

### HCL Configuration

```hcl
processor "sampling" "adaptive" {
  base_sample_rate = 0.1  # 10% baseline
  always_sample_errors = true
  slow_threshold = "500ms"
  adaptive_window = 1000
}

pipeline "traces" {
  receivers = [receiver.otlp.main]
  processors = [processor.sampling.adaptive, processor.batch.default]
  exporters = [exporter.jaeger.backend]
}
```

### Benefits

1. **Cost Reduction**: Reduces storage and bandwidth requirements
2. **Signal Preservation**: Keeps important traces (errors, slow requests)
3. **Automatic Adaptation**: Increases sampling during incidents
4. **Trace Consistency**: Entire traces are sampled together

### Comparison with Static Sampling

| Aspect | Static Sampling | Adaptive Sampling |
|--------|----------------|-------------------|
| Error Detection | May miss rare errors | Always captures errors |
| Performance Issues | Random sampling | Captures slow spans |
| Storage Cost | Fixed | Optimizes automatically |
| Incident Response | Fixed rate | Increases during issues |

## Implementation Details

### Thread Safety
All metrics operations use atomic counters and mutex protection for concurrent access.

### Performance
- Metrics tracking adds < 1μs overhead per span
- Sampling decision adds < 500ns per span
- Health check runs in separate goroutine

### Memory Usage
- Metrics buffer: ~8KB for 1000 latency samples
- Sampling state: ~200 bytes
- Health check: ~1KB

## Future Enhancements

1. **Prometheus Integration**: Export metrics in Prometheus format
2. **Advanced Sampling**: ML-based sampling decisions
3. **Distributed Rate Limiting**: Coordinate sampling across collectors
4. **Custom Health Rules**: User-defined health thresholds
5. **Metrics Aggregation**: Time-series metrics with bucketing

## References

- [OpenTelemetry Sampling](https://opentelemetry.io/docs/specs/otel/trace/sdk/#sampling)
- [Jaeger Adaptive Sampling](https://www.jaegertracing.io/docs/latest/sampling/#adaptive-sampling)
- [Kubernetes Health Checks](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
