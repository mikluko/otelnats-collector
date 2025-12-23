// Package natsreceiver receives OpenTelemetry data from NATS subjects.
package natsreceiver

import (
	"errors"

	"go.opentelemetry.io/collector/component"

	internalnats "github.com/mikluko/otelnats-collector/internal/nats"
)

// Config defines configuration for the NATS receiver.
type Config struct {
	internalnats.ClientConfig `mapstructure:",squash"`

	// Traces configuration.
	Traces SignalConfig `mapstructure:"traces"`

	// Metrics configuration.
	Metrics SignalConfig `mapstructure:"metrics"`

	// Logs configuration.
	Logs SignalConfig `mapstructure:"logs"`
}

// SignalConfig holds signal-specific receiver configuration.
type SignalConfig struct {
	// Subject is the NATS subject to subscribe to.
	// Supports wildcards: * (single token), > (multi-level).
	Subject string `mapstructure:"subject"`

	// QueueGroup for load-balanced consumption across receivers.
	// If empty, messages are broadcast to all subscribers.
	QueueGroup string `mapstructure:"queue_group"`

	// Encoding for message deserialization (default: otlp_proto).
	// Currently only otlp_proto is supported.
	Encoding string `mapstructure:"encoding"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.ClientConfig.URL == "" {
		return errors.New("url is required")
	}
	// At least one signal must be configured with a subject
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
