// Package natsexporter exports OpenTelemetry data to NATS subjects.
package natsexporter

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	internalnats "github.com/mikluko/opentelemetry-collector-nats/internal/nats"
)

// Config defines configuration for the NATS exporter.
type Config struct {
	exporterhelper.TimeoutConfig `mapstructure:",squash"`
	configretry.BackOffConfig    `mapstructure:"retry_on_failure"`
	internalnats.ClientConfig    `mapstructure:",squash"`

	// Traces configuration.
	Traces SignalConfig `mapstructure:"traces"`

	// Metrics configuration.
	Metrics SignalConfig `mapstructure:"metrics"`

	// Logs configuration.
	Logs SignalConfig `mapstructure:"logs"`

	// SubjectFromAttribute extracts subject from a resource attribute.
	// If set, the value of this attribute will be used as the subject.
	SubjectFromAttribute string `mapstructure:"subject_from_attribute,omitempty"`
}

// SignalConfig holds signal-specific configuration.
type SignalConfig struct {
	// Subject is the NATS subject to publish to.
	// Supports template variables:
	//   ${signal} - the signal type (traces, metrics, logs)
	//   ${attr:key} - value of resource attribute "key"
	Subject string `mapstructure:"subject"`

	// Encoding for message serialization (default: otlp_proto).
	// Currently only otlp_proto is supported.
	Encoding string `mapstructure:"encoding"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.ClientConfig.URL == "" {
		return errors.New("url is required")
	}
	if c.Traces.Subject == "" && c.Metrics.Subject == "" && c.Logs.Subject == "" {
		return errors.New("at least one signal subject must be configured")
	}
	// Validate encoding if specified
	for _, cfg := range []SignalConfig{c.Traces, c.Metrics, c.Logs} {
		if cfg.Encoding != "" && cfg.Encoding != defaultEncoding {
			return errors.New("only otlp_proto encoding is currently supported")
		}
	}
	return nil
}
