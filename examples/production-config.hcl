# Production Pipeline Configuration
# High-throughput setup with observability and intelligent sampling

receiver "otlp" "main" {
  grpc {
    endpoint = "0.0.0.0:4317"
    max_recv_msg_size = "16MB"
    max_concurrent_streams = 1000
  }
  http {
    endpoint = "0.0.0.0:4318"
    max_request_body_size = "16MB"
  }
}

# Multi-stage sampling strategy
processor "sampling" "head" {
  # Aggressive initial sampling (1%)
  base_sample_rate = 0.01
  always_sample_errors = true
  slow_threshold = "1s"
  adaptive_window = 10000
}

processor "sampling" "tail" {
  # Additional tail-based sampling for quality
  base_sample_rate = 1.0 # Keep all after head sampling
  always_sample_errors = true
  slow_threshold = "2s"
}

processor "batch" "large" {
  timeout = "2s"
  send_batch_size = 8192
  batch_size = 16384
}

# Multiple exporters for redundancy
exporter "jaeger" "primary" {
  endpoint = "jaeger-collector-1:14250"
  tls {
    insecure = false
  }
  retry {
    enabled = true
    initial_interval = "1s"
    max_interval = "30s"
    max_elapsed_time = "5m"
  }
}

exporter "jaeger" "secondary" {
  endpoint = "jaeger-collector-2:14250"
  tls {
    insecure = false
  }
}

pipeline "production-traces" {
  receivers = [receiver.otlp.main]
  
  processors = [
    processor.sampling.head,   # Initial aggressive sampling
    processor.sampling.tail,   # Quality-based tail sampling
    processor.batch.large      # Large batches for efficiency
  ]
  
  # Export to multiple backends
  exporters = [
    exporter.jaeger.primary,
    exporter.jaeger.secondary
  ]
  
  observability {
    enabled = true
    
    health_check {
      addr = ":8888"
      drop_rate_warning = 0.5
      drop_rate_critical = 2.0
      error_rate_warning = 1.0
      error_rate_critical = 5.0
    }
  }
}
