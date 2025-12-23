package natsexporter

import (
	"context"
	"strings"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	internalnats "github.com/mikluko/otelnats-collector/internal/nats"
)

type signalType string

const (
	signalTraces  signalType = "traces"
	signalMetrics signalType = "metrics"
	signalLogs    signalType = "logs"
)

type natsExporter struct {
	config     *Config
	settings   exporter.Settings
	logger     *zap.Logger
	signalType signalType

	conn             *nats.Conn
	tracesMarshaler  ptrace.Marshaler
	metricsMarshaler pmetric.Marshaler
	logsMarshaler    plog.Marshaler
}

func newNatsExporter(cfg *Config, set exporter.Settings, signal signalType) *natsExporter {
	return &natsExporter{
		config:     cfg,
		settings:   set,
		logger:     set.Logger,
		signalType: signal,
	}
}

func (e *natsExporter) start(ctx context.Context, _ component.Host) error {
	conn, err := internalnats.Connect(ctx, e.config.ClientConfig, e.logger)
	if err != nil {
		return err
	}
	e.conn = conn

	// Initialize marshalers
	e.tracesMarshaler = &ptrace.ProtoMarshaler{}
	e.metricsMarshaler = &pmetric.ProtoMarshaler{}
	e.logsMarshaler = &plog.ProtoMarshaler{}

	e.logger.Info("NATS exporter started",
		zap.String("url", e.config.URL),
		zap.String("signal", string(e.signalType)),
	)
	return nil
}

func (e *natsExporter) shutdown(_ context.Context) error {
	if e.conn != nil {
		// Drain ensures all pending messages are sent before closing
		return e.conn.Drain()
	}
	return nil
}

func (e *natsExporter) signalSubject() string {
	switch e.signalType {
	case signalTraces:
		return e.config.Traces.Subject
	case signalMetrics:
		return e.config.Metrics.Subject
	case signalLogs:
		return e.config.Logs.Subject
	default:
		return ""
	}
}

// expandSubject expands template variables in the subject string.
// Supported variables:
//   - ${signal} - the signal type (traces, metrics, logs)
//   - ${attr:key} - value of resource attribute "key"
func (e *natsExporter) expandSubject(template string, attrs map[string]string) string {
	result := strings.ReplaceAll(template, "${signal}", string(e.signalType))
	for key, value := range attrs {
		placeholder := "${attr:" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

func (e *natsExporter) publishTraces(ctx context.Context, td ptrace.Traces) error {
	data, err := e.tracesMarshaler.MarshalTraces(td)
	if err != nil {
		return consumererror.NewPermanent(err)
	}

	attrs := extractResourceAttrsFromTraces(td)
	subject := e.expandSubject(e.signalSubject(), attrs)

	if err := e.conn.Publish(subject, data); err != nil {
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

	attrs := extractResourceAttrsFromMetrics(md)
	subject := e.expandSubject(e.signalSubject(), attrs)

	if err := e.conn.Publish(subject, data); err != nil {
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

	attrs := extractResourceAttrsFromLogs(ld)
	subject := e.expandSubject(e.signalSubject(), attrs)

	if err := e.conn.Publish(subject, data); err != nil {
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

func extractResourceAttrsFromTraces(td ptrace.Traces) map[string]string {
	attrs := make(map[string]string)
	if td.ResourceSpans().Len() > 0 {
		td.ResourceSpans().At(0).Resource().Attributes().Range(func(k string, v pcommon.Value) bool {
			attrs[k] = v.AsString()
			return true
		})
	}
	return attrs
}

func extractResourceAttrsFromMetrics(md pmetric.Metrics) map[string]string {
	attrs := make(map[string]string)
	if md.ResourceMetrics().Len() > 0 {
		md.ResourceMetrics().At(0).Resource().Attributes().Range(func(k string, v pcommon.Value) bool {
			attrs[k] = v.AsString()
			return true
		})
	}
	return attrs
}

func extractResourceAttrsFromLogs(ld plog.Logs) map[string]string {
	attrs := make(map[string]string)
	if ld.ResourceLogs().Len() > 0 {
		ld.ResourceLogs().At(0).Resource().Attributes().Range(func(k string, v pcommon.Value) bool {
			attrs[k] = v.AsString()
			return true
		})
	}
	return attrs
}
