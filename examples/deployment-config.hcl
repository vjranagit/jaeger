# Example Jaeger deployment configuration for Kubernetes
# This demonstrates the HCL-based deployment specification

deployment "production-jaeger" {
  strategy = "production"

  storage {
    type = "elasticsearch"
    elasticsearch {
      urls = ["https://elasticsearch:9200"]
      index_prefix = "jaeger"
      username = "elastic"
      password = "${env.ES_PASSWORD}"
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

    resources {
      requests {
        cpu = "500m"
        memory = "512Mi"
      }
      limits {
        cpu = "2000m"
        memory = "2Gi"
      }
    }
  }

  query {
    replicas = 3

    resources {
      requests {
        cpu = "250m"
        memory = "256Mi"
      }
      limits {
        cpu = "1000m"
        memory = "1Gi"
      }
    }
  }

  ingress {
    enabled = true
    host = "jaeger.example.com"
    tls = true
    annotations = {
      "kubernetes.io/ingress.class" = "nginx"
      "cert-manager.io/cluster-issuer" = "letsencrypt-prod"
    }
  }

  monitoring {
    prometheus = true
    service_monitor = true
  }
}
