# Advanced Pipeline Configuration
# Demonstrates self-observability, adaptive sampling, and health checks

receiver "otlp" "main" {
  grpc {
    endpoint = "0.0.0.0:4317"
  }
  http {
    endpoint = "0.0.0.0:4318"
  }
}

# Adaptive sampling processor
# Intelligently samples spans based on errors and latency
processor "sampling" "adaptive" {
  # Base sampling rate (10% of normal traffic)
  base_sample_rate = 0.1
  
  # Always keep error spans
  always_sample_errors = true
  
  # Always keep slow requests (>500ms)
  slow_threshold = "500ms"
  
  # Adapt sampling rate over 1000 spans
  adaptive_window = 1000
}

# Batch processor for efficient export
processor "batch" "default" {
  timeout = "1s"
  send_batch_size = 1024
}

exporter "jaeger" "backend" {
  endpoint = "jaeger-collector:14250"
  tls {
    insecure = false
    cert_file = "/etc/certs/client.crt"
    key_file = "/etc/certs/client.key"
    ca_file = "/etc/certs/ca.crt"
  }
}

# Main traces pipeline
pipeline "traces" {
  receivers = [receiver.otlp.main]
  
  # Apply adaptive sampling before batching
  processors = [
    processor.sampling.adaptive,
    processor.batch.default
  ]
  
  exporters = [exporter.jaeger.backend]
  
  # Enable self-observability
  observability {
    enabled = true
    
    # Health check configuration
    health_check {
      addr = ":8888"
      
      # Thresholds for health status
      drop_rate_warning = 1.0   # 1% drops -> degraded
      drop_rate_critical = 5.0  # 5% drops -> unhealthy
      error_rate_warning = 2.0  # 2% errors -> degraded
      error_rate_critical = 10.0 # 10% errors -> unhealthy
    }
    
    # Metrics export (future enhancement)
    metrics {
      enabled = true
      endpoint = "prometheus:9090"
      interval = "15s"
    }
  }
}
