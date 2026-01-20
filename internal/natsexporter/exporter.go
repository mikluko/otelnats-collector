package natsexporter

import (
	"context"

	"github.com/mikluko/otelnats"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	internalnats "github.com/mikluko/otelnats-collector/internal/nats"
)

type natsExporter struct {
	config   *Config
	settings exporter.Settings
	logger   *zap.Logger

	conn *nats.Conn

	// Proto marshalers for converting pdata to OTLP protobuf
	tracesMarshaler  ptrace.Marshaler
	metricsMarshaler pmetric.Marshaler
	logsMarshaler    plog.Marshaler
}

func newNatsExporter(cfg *Config, set exporter.Settings) *natsExporter {
	return &natsExporter{
		config:   cfg,
		settings: set,
		logger:   set.Logger,
	}
}

func (e *natsExporter) start(ctx context.Context, _ component.Host) error {
	conn, err := internalnats.Connect(ctx, e.config.ClientConfig, e.logger)
	if err != nil {
		return err
	}
	e.conn = conn

	// Initialize proto marshalers
	e.tracesMarshaler = &ptrace.ProtoMarshaler{}
	e.metricsMarshaler = &pmetric.ProtoMarshaler{}
	e.logsMarshaler = &plog.ProtoMarshaler{}

	e.logger.Info("NATS exporter started",
		zap.String("url", e.config.URL),
	)
	return nil
}

func (e *natsExporter) shutdown(_ context.Context) error {
	// Drain ensures all pending messages are sent before closing
	if e.conn != nil {
		return e.conn.Drain()
	}
	return nil
}

func (e *natsExporter) publishTraces(ctx context.Context, td ptrace.Traces) error {
	// Marshal pdata to OTLP protobuf bytes
	data, err := e.tracesMarshaler.MarshalTraces(td)
	if err != nil {
		return consumererror.NewPermanent(err)
	}

	// Use configured subject and SDK protocol headers
	subject := e.config.Traces.Subject
	headers := otelnats.BuildHeaders(ctx, otelnats.SignalTraces, otelnats.EncodingProtobuf, nil)

	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  headers,
	}

	if err := e.conn.PublishMsg(msg); err != nil {
		e.logger.Error("failed to publish traces",
			zap.String("subject", subject),
			zap.Error(err),
		)
		return err
	}

	e.logger.Debug("published traces",
		zap.String("subject", subject),
		zap.Int("spans", td.SpanCount()),
		zap.Int("bytes", len(data)),
	)
	return nil
}

func (e *natsExporter) publishMetrics(ctx context.Context, md pmetric.Metrics) error {
	data, err := e.metricsMarshaler.MarshalMetrics(md)
	if err != nil {
		return consumererror.NewPermanent(err)
	}

	// Use configured subject and SDK protocol headers
	subject := e.config.Metrics.Subject
	headers := otelnats.BuildHeaders(ctx, otelnats.SignalMetrics, otelnats.EncodingProtobuf, nil)

	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  headers,
	}

	if err := e.conn.PublishMsg(msg); err != nil {
		e.logger.Error("failed to publish metrics",
			zap.String("subject", subject),
			zap.Error(err),
		)
		return err
	}

	e.logger.Debug("published metrics",
		zap.String("subject", subject),
		zap.Int("datapoints", md.DataPointCount()),
		zap.Int("bytes", len(data)),
	)
	return nil
}

func (e *natsExporter) publishLogs(ctx context.Context, ld plog.Logs) error {
	data, err := e.logsMarshaler.MarshalLogs(ld)
	if err != nil {
		return consumererror.NewPermanent(err)
	}

	// Use configured subject and SDK protocol headers
	subject := e.config.Logs.Subject
	headers := otelnats.BuildHeaders(ctx, otelnats.SignalLogs, otelnats.EncodingProtobuf, nil)

	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  headers,
	}

	if err := e.conn.PublishMsg(msg); err != nil {
		e.logger.Error("failed to publish logs",
			zap.String("subject", subject),
			zap.Error(err),
		)
		return err
	}

	e.logger.Debug("published logs",
		zap.String("subject", subject),
		zap.Int("records", ld.LogRecordCount()),
		zap.Int("bytes", len(data)),
	)
	return nil
}
