// Package natsreceiver receives OpenTelemetry data from NATS subjects.
package natsreceiver

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"

	internalnats "github.com/mikluko/otelnats-collector/internal/nats"
)

// Config defines configuration for the NATS receiver.
type Config struct {
	internalnats.ClientConfig `mapstructure:",squash"`

	// QueueGroup for load-balanced consumption across receivers.
	// When multiple receivers use the same queue group, each message is
	// delivered to only one receiver in the group.
	// Only applies to core NATS mode (JetStream uses durable consumers).
	QueueGroup string `mapstructure:"queue_group"`

	// JetStream configuration for at-least-once delivery guarantees.
	// If not set, uses core NATS (at-most-once delivery).
	JetStream *JetStreamConfig `mapstructure:"jetstream,omitempty"`

	// Traces configuration.
	Traces SignalConfig `mapstructure:"traces"`

	// Metrics configuration.
	Metrics SignalConfig `mapstructure:"metrics"`

	// Logs configuration.
	Logs SignalConfig `mapstructure:"logs"`
}

// JetStreamConfig holds JetStream-specific receiver configuration.
type JetStreamConfig struct {
	// Stream is the JetStream stream to consume from.
	Stream string `mapstructure:"stream"`

	// Consumer is the durable consumer name.
	// If empty, a consumer name is auto-generated.
	// Multiple receiver instances can share the same consumer name for load balancing.
	Consumer string `mapstructure:"consumer,omitempty"`

	// AckWait is the acknowledgment timeout for messages.
	// If a message is not acknowledged within this duration, it will be redelivered.
	// Default is 30 seconds.
	AckWait time.Duration `mapstructure:"ack_wait,omitempty"`

	// BacklogSize is the buffer size for message backlog.
	// Default is 100.
	BacklogSize int `mapstructure:"backlog_size,omitempty"`
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
	// Validate JetStream configuration if enabled
	if c.JetStream != nil {
		if c.JetStream.Stream == "" {
			return errors.New("jetstream.stream is required when jetstream is enabled")
		}
		if c.JetStream.AckWait < 0 {
			return errors.New("jetstream.ack_wait must be non-negative")
		}
		if c.JetStream.BacklogSize < 0 {
			return errors.New("jetstream.backlog_size must be non-negative")
		}
	}
	return nil
}
