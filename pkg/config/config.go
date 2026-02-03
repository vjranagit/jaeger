package config

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
)

// Config represents the root configuration
type Config struct {
	Receivers  []ReceiverBlock  `hcl:"receiver,block"`
	Processors []ProcessorBlock `hcl:"processor,block"`
	Exporters  []ExporterBlock  `hcl:"exporter,block"`
	Pipelines  []PipelineBlock  `hcl:"pipeline,block"`
}

// ReceiverBlock represents a receiver configuration block
type ReceiverBlock struct {
	Type   string   `hcl:"type,label"`
	Name   string   `hcl:"name,label"`
	Body   hcl.Body `hcl:",remain"`
	Config ReceiverConfig
}

// ReceiverConfig holds receiver-specific configuration
type ReceiverConfig struct {
	OTLP *OTLPReceiverConfig `hcl:"otlp,block"`
}

// OTLPReceiverConfig configures OTLP receiver
type OTLPReceiverConfig struct {
	GRPC *GRPCConfig `hcl:"grpc,block"`
	HTTP *HTTPConfig `hcl:"http,block"`
}

// GRPCConfig configures gRPC endpoint
type GRPCConfig struct {
	Endpoint string `hcl:"endpoint"`
}

// HTTPConfig configures HTTP endpoint
type HTTPConfig struct {
	Endpoint string `hcl:"endpoint"`
}

// ProcessorBlock represents a processor configuration block
type ProcessorBlock struct {
	Type   string   `hcl:"type,label"`
	Name   string   `hcl:"name,label"`
	Body   hcl.Body `hcl:",remain"`
	Config ProcessorConfig
}

// ProcessorConfig holds processor-specific configuration
type ProcessorConfig struct {
	Batch      *BatchProcessorConfig      `hcl:"batch,block"`
	Attributes *AttributesProcessorConfig `hcl:"attributes,block"`
}

// BatchProcessorConfig configures batch processor
type BatchProcessorConfig struct {
	Timeout       string `hcl:"timeout"`
	SendBatchSize int    `hcl:"send_batch_size"`
}

// AttributesProcessorConfig configures attributes processor
type AttributesProcessorConfig struct {
	Actions []AttributeAction `hcl:"action,block"`
}

// AttributeAction represents an attribute modification action
type AttributeAction struct {
	Key    string `hcl:"key"`
	Value  string `hcl:"value"`
	Action string `hcl:"action"`
}

// ExporterBlock represents an exporter configuration block
type ExporterBlock struct {
	Type   string   `hcl:"type,label"`
	Name   string   `hcl:"name,label"`
	Body   hcl.Body `hcl:",remain"`
	Config ExporterConfig
}

// ExporterConfig holds exporter-specific configuration
type ExporterConfig struct {
	Jaeger *JaegerExporterConfig `hcl:"jaeger,block"`
}

// JaegerExporterConfig configures Jaeger exporter
type JaegerExporterConfig struct {
	Endpoint string     `hcl:"endpoint"`
	TLS      *TLSConfig `hcl:"tls,block"`
}

// TLSConfig configures TLS settings
type TLSConfig struct {
	Insecure bool `hcl:"insecure"`
}

// PipelineBlock represents a pipeline configuration block
type PipelineBlock struct {
	Name       string   `hcl:"name,label"`
	Receivers  []string `hcl:"receivers"`
	Processors []string `hcl:"processors"`
	Exporters  []string `hcl:"exporters"`
}

// LoadConfig loads configuration from HCL file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := hclsimple.Decode(filename, data, nil, &config); err != nil {
		return nil, fmt.Errorf("failed to parse HCL config: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Pipelines) == 0 {
		return fmt.Errorf("at least one pipeline must be defined")
	}

	for _, pipeline := range c.Pipelines {
		if len(pipeline.Receivers) == 0 {
			return fmt.Errorf("pipeline %s has no receivers", pipeline.Name)
		}
		if len(pipeline.Exporters) == 0 {
			return fmt.Errorf("pipeline %s has no exporters", pipeline.Name)
		}
	}

	return nil
}
