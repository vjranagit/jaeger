# Example telemetry pipeline configuration
# This demonstrates the HCL-based configuration approach

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

processor "attributes" "enrich" {
  action {
    key = "environment"
    value = "production"
    action = "insert"
  }
  action {
    key = "service.version"
    value = "1.0.0"
    action = "upsert"
  }
}

exporter "jaeger" "backend" {
  endpoint = "jaeger-collector:14250"
  tls {
    insecure = false
  }
}

pipeline "traces" {
  receivers = ["receiver.otlp.main"]
  processors = [
    "processor.batch.default",
    "processor.attributes.enrich"
  ]
  exporters = ["exporter.jaeger.backend"]
}
