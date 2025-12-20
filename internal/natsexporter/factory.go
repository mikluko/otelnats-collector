package natsexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"github.com/mikluko/nats-otel-collector/internal/metadata"
	internalnats "github.com/mikluko/nats-otel-collector/internal/nats"
)

const (
	defaultTracesSubject  = "otel.traces"
	defaultMetricsSubject = "otel.metrics"
	defaultLogsSubject    = "otel.logs"
	defaultEncoding       = "otlp_proto"
)

// NewFactory creates a factory for the NATS exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithTraces(createTracesExporter, metadata.TracesStability),
		exporter.WithMetrics(createMetricsExporter, metadata.MetricsStability),
		exporter.WithLogs(createLogsExporter, metadata.LogsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		TimeoutConfig: exporterhelper.NewDefaultTimeoutConfig(),
		BackOffConfig: configretry.NewDefaultBackOffConfig(),
		ClientConfig:  internalnats.NewDefaultClientConfig(),
		Traces: SignalConfig{
			Subject:  defaultTracesSubject,
			Encoding: defaultEncoding,
		},
		Metrics: SignalConfig{
			Subject:  defaultMetricsSubject,
			Encoding: defaultEncoding,
		},
		Logs: SignalConfig{
			Subject:  defaultLogsSubject,
			Encoding: defaultEncoding,
		},
	}
}

func createTracesExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	config := cfg.(*Config)
	exp := newNatsExporter(config, set, signalTraces)

	return exporterhelper.NewTraces(
		ctx,
		set,
		cfg,
		exp.publishTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithShutdown(exp.shutdown),
		exporterhelper.WithTimeout(config.TimeoutConfig),
		exporterhelper.WithRetry(config.BackOffConfig),
	)
}

func createMetricsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Metrics, error) {
	config := cfg.(*Config)
	exp := newNatsExporter(config, set, signalMetrics)

	return exporterhelper.NewMetrics(
		ctx,
		set,
		cfg,
		exp.publishMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithShutdown(exp.shutdown),
		exporterhelper.WithTimeout(config.TimeoutConfig),
		exporterhelper.WithRetry(config.BackOffConfig),
	)
}

func createLogsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	config := cfg.(*Config)
	exp := newNatsExporter(config, set, signalLogs)

	return exporterhelper.NewLogs(
		ctx,
		set,
		cfg,
		exp.publishLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithShutdown(exp.shutdown),
		exporterhelper.WithTimeout(config.TimeoutConfig),
		exporterhelper.WithRetry(config.BackOffConfig),
	)
}
