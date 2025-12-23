package natsreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/mikluko/otelnats-collector/internal/metadata"
	internalnats "github.com/mikluko/otelnats-collector/internal/nats"
)

const (
	defaultTracesSubject  = "otel.traces"
	defaultMetricsSubject = "otel.metrics"
	defaultLogsSubject    = "otel.logs"
	defaultQueueGroup     = "otel-collector"
	defaultEncoding       = "otlp_proto"
)

// NewFactory creates a factory for the NATS receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		ClientConfig: internalnats.NewDefaultClientConfig(),
		Traces: SignalConfig{
			Subject:    defaultTracesSubject,
			QueueGroup: defaultQueueGroup,
			Encoding:   defaultEncoding,
		},
		Metrics: SignalConfig{
			Subject:    defaultMetricsSubject,
			QueueGroup: defaultQueueGroup,
			Encoding:   defaultEncoding,
		},
		Logs: SignalConfig{
			Subject:    defaultLogsSubject,
			QueueGroup: defaultQueueGroup,
			Encoding:   defaultEncoding,
		},
	}
}

func createTracesReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	config := cfg.(*Config)
	return newNatsReceiver(config, set, config.Traces, nextConsumer, nil, nil)
}

func createMetricsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	config := cfg.(*Config)
	return newNatsReceiver(config, set, config.Metrics, nil, nextConsumer, nil)
}

func createLogsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	config := cfg.(*Config)
	return newNatsReceiver(config, set, config.Logs, nil, nil, nextConsumer)
}
