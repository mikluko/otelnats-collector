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
	// Only applies to core NATS mode (JetStream uses durable consumers).
	QueueGroup string `mapstructure:"queue_group"`

	// Encoding for message deserialization (default: otlp_proto).
	// Currently only otlp_proto is supported.
	Encoding string `mapstructure:"encoding"`

	// JetStream configuration for at-least-once delivery guarantees.
	// If not set, uses core NATS (at-most-once delivery).
	JetStream *JetStreamConfig `mapstructure:"jetstream,omitempty"`
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

	// BacklogSize is the buffer size for message backlog (core NATS mode only).
	// Default is 100.
	// NOTE: This option does NOT apply when rate limiting is enabled,
	// which uses the Fetch API and processes messages directly.
	BacklogSize int `mapstructure:"backlog_size,omitempty"`

	// RateLimit enables token bucket rate limiting for message consumption.
	// Specifies the target rate in messages per second.
	// When enabled, switches from push-based Consume() to pull-based Fetch() API.
	// Tokens are acquired BEFORE fetching to avoid wasting ACK timeout.
	// A value of 0 disables rate limiting (default).
	RateLimit float64 `mapstructure:"rate_limit,omitempty"`

	// RateBurst is the token bucket capacity (maximum burst size).
	// Required when RateLimit is set. Also used as the default fetch batch size.
	RateBurst int `mapstructure:"rate_burst,omitempty"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if err := internalnats.ValidateURL(c.ClientConfig.URL); err != nil {
		return err
	}

	if err := c.ClientConfig.Auth.Validate(); err != nil {
		return err
	}

	// At least one signal must be configured with a subject
	if c.Traces.Subject == "" && c.Metrics.Subject == "" && c.Logs.Subject == "" {
		return errors.New("at least one signal subject must be configured")
	}

	// Validate each signal configuration
	signals := map[string]SignalConfig{
		"traces":  c.Traces,
		"metrics": c.Metrics,
		"logs":    c.Logs,
	}

	for name, cfg := range signals {
		// Validate subject format if configured
		if cfg.Subject != "" {
			if err := internalnats.ValidateSubject(cfg.Subject); err != nil {
				return errors.New(name + ".subject: " + err.Error())
			}
		}

		// Validate encoding if specified
		if cfg.Encoding != "" && cfg.Encoding != defaultEncoding {
			return errors.New("only otlp_proto encoding is currently supported")
		}

		// Validate JetStream configuration if enabled for this signal
		if cfg.JetStream != nil {
			if cfg.JetStream.Stream == "" {
				return errors.New(name + ".jetstream.stream is required when jetstream is enabled")
			}
			if cfg.JetStream.AckWait < 0 {
				return errors.New(name + ".jetstream.ack_wait must be non-negative")
			}
			if cfg.JetStream.BacklogSize < 0 {
				return errors.New(name + ".jetstream.backlog_size must be non-negative")
			}
			if cfg.JetStream.RateLimit < 0 {
				return errors.New(name + ".jetstream.rate_limit must be non-negative")
			}
			if cfg.JetStream.RateLimit > 0 && cfg.JetStream.RateBurst <= 0 {
				return errors.New(name + ".jetstream.rate_burst is required when rate_limit is set")
			}
			if cfg.JetStream.RateBurst < 0 {
				return errors.New(name + ".jetstream.rate_burst must be non-negative")
			}
		}
	}

	return nil
}
