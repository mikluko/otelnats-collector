package natsreceiver

import (
	"context"
	"errors"
	"fmt"

	"github.com/mikluko/otelnats"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracespb "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	internalnats "github.com/mikluko/otelnats-collector/internal/nats"
)

type natsReceiver struct {
	config   *Config
	settings receiver.Settings
	logger   *zap.Logger
	obsrecv  *receiverhelper.ObsReport

	downstreamErrLevel zapcore.Level

	conn        *nats.Conn
	sdkReceiver otelnats.Receiver

	// Standard pdata unmarshalers (Kafka pattern)
	tracesUnmarshaler  ptrace.Unmarshaler
	metricsUnmarshaler pmetric.Unmarshaler
	logsUnmarshaler    plog.Unmarshaler

	tracesConsumer  consumer.Traces
	metricsConsumer consumer.Metrics
	logsConsumer    consumer.Logs
}

func newNatsReceiver(
	cfg *Config,
	set receiver.Settings,
	tracesConsumer consumer.Traces,
	metricsConsumer consumer.Metrics,
	logsConsumer consumer.Logs,
) (*natsReceiver, error) {
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		Transport:              "nats",
		ReceiverCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}

	return &natsReceiver{
		config:          cfg,
		settings:        set,
		logger:          set.Logger,
		obsrecv:         obsrecv,
		tracesConsumer:  tracesConsumer,
		metricsConsumer: metricsConsumer,
		logsConsumer:    logsConsumer,
	}, nil
}

func (r *natsReceiver) Start(ctx context.Context, _ component.Host) error {
	// Initialize unmarshalers for bytes→pdata conversion
	r.tracesUnmarshaler = &ptrace.ProtoUnmarshaler{}
	r.metricsUnmarshaler = &pmetric.ProtoUnmarshaler{}
	r.logsUnmarshaler = &plog.ProtoUnmarshaler{}

	// Connect to NATS
	conn, err := internalnats.Connect(ctx, r.config.ClientConfig, r.logger)
	if err != nil {
		return err
	}
	r.conn = conn

	// Build SDK receiver options (same for both core NATS and JetStream)
	opts := []otelnats.ReceiverOption{
		otelnats.WithReceiverBaseContext(context.Background()),
		otelnats.WithReceiverErrorHandler(r.handleError),
	}

	// Determine which signal is enabled and get its configuration
	var signalConfig *SignalConfig
	var jsConfig *JetStreamConfig

	// Register handlers and subjects only for enabled signals (where consumer is not nil)
	if r.tracesConsumer != nil {
		signalConfig = &r.config.Traces
		opts = append(opts, otelnats.WithReceiverTracesHandler(r.handleTracesMessage))
		if r.config.Traces.Subject != "" {
			opts = append(opts, otelnats.WithReceiverSignalSubject(otelnats.SignalTraces, r.config.Traces.Subject))
		}
		if r.config.Traces.JetStream != nil {
			jsConfig = r.config.Traces.JetStream
		}
	}
	if r.metricsConsumer != nil {
		signalConfig = &r.config.Metrics
		opts = append(opts, otelnats.WithReceiverMetricsHandler(r.handleMetricsMessage))
		if r.config.Metrics.Subject != "" {
			opts = append(opts, otelnats.WithReceiverSignalSubject(otelnats.SignalMetrics, r.config.Metrics.Subject))
		}
		if r.config.Metrics.JetStream != nil {
			jsConfig = r.config.Metrics.JetStream
		}
	}
	if r.logsConsumer != nil {
		signalConfig = &r.config.Logs
		opts = append(opts, otelnats.WithReceiverLogsHandler(r.handleLogsMessage))
		if r.config.Logs.Subject != "" {
			opts = append(opts, otelnats.WithReceiverSignalSubject(otelnats.SignalLogs, r.config.Logs.Subject))
		}
		if r.config.Logs.JetStream != nil {
			jsConfig = r.config.Logs.JetStream
		}
	}

	// Add mode-specific options based on signal configuration
	if jsConfig != nil {
		// JetStream mode

		r.downstreamErrLevel = zap.WarnLevel

		js, err := jetstream.New(conn)
		if err != nil {
			return fmt.Errorf("failed to create JetStream context: %w", err)
		}

		opts = append(opts, otelnats.WithReceiverJetStream(js, jsConfig.Stream))

		if jsConfig.Consumer != "" {
			opts = append(opts, otelnats.WithReceiverConsumerName(jsConfig.Consumer))
		}
		if jsConfig.AckWait > 0 {
			opts = append(opts, otelnats.WithReceiverAckWait(jsConfig.AckWait))
		}
		if jsConfig.BacklogSize > 0 {
			opts = append(opts, otelnats.WithReceiverBacklogSize(jsConfig.BacklogSize))
		}
	} else {
		// Core NATS mode - use signal-specific queue group if available, otherwise connection-level

		r.downstreamErrLevel = zap.ErrorLevel

		queueGroup := r.config.QueueGroup
		if signalConfig != nil && signalConfig.QueueGroup != "" {
			queueGroup = signalConfig.QueueGroup
		}
		if queueGroup != "" {
			opts = append(opts, otelnats.WithReceiverQueueGroup(queueGroup))
		}
	}

	// Create and start SDK receiver
	sdkReceiver, err := otelnats.NewReceiver(conn, opts...)
	if err != nil {
		return fmt.Errorf("failed to create SDK receiver: %w", err)
	}
	r.sdkReceiver = sdkReceiver

	if err := r.sdkReceiver.Start(ctx); err != nil {
		return fmt.Errorf("failed to start SDK receiver: %w", err)
	}

	// Log startup info
	if jsConfig != nil {
		r.logger.Info("NATS receiver started (JetStream mode)",
			zap.String("url", r.config.URL),
			zap.String("stream", jsConfig.Stream),
			zap.String("consumer", jsConfig.Consumer),
		)
	} else {
		queueGroup := r.config.QueueGroup
		if signalConfig != nil && signalConfig.QueueGroup != "" {
			queueGroup = signalConfig.QueueGroup
		}
		r.logger.Info("NATS receiver started (core NATS mode)",
			zap.String("url", r.config.URL),
			zap.String("queue_group", queueGroup),
		)
	}

	return nil
}

func (r *natsReceiver) Shutdown(ctx context.Context) error {
	if r.sdkReceiver != nil {
		if err := r.sdkReceiver.Shutdown(ctx); err != nil {
			return err
		}
	}

	if r.conn != nil {
		r.conn.Close()
	}
	return nil
}

// Message handlers using SDK MessageSignal API (works for both core NATS and JetStream)

func (r *natsReceiver) handleTracesMessage(ctx context.Context, msg otelnats.MessageSignal[tracespb.TracesData]) error {
	ctx = r.obsrecv.StartTracesOp(ctx)

	// Choose unmarshaler based on Content-Type header
	contentType := msg.Headers().Get(otelnats.HeaderContentType)
	var traces ptrace.Traces
	var err error

	if contentType == otelnats.ContentTypeJSON {
		unmarshaler := &ptrace.JSONUnmarshaler{}
		traces, err = unmarshaler.UnmarshalTraces(msg.Data())
	} else {
		// Default to protobuf (application/x-protobuf or empty)
		traces, err = r.tracesUnmarshaler.UnmarshalTraces(msg.Data())
	}

	if err != nil {
		r.obsrecv.EndTracesOp(ctx, contentType, 0, err)
		r.logger.Error("failed to unmarshal traces",
			zap.String("subject", msg.Subject()),
			zap.String("content_type", contentType),
			zap.Error(err),
		)
		return err
	}

	spanCount := traces.SpanCount()
	err = r.tracesConsumer.ConsumeTraces(ctx, traces)
	r.obsrecv.EndTracesOp(ctx, contentType, spanCount, err)

	if err != nil {
		return receiverError{
			err:    err,
			fields: []zap.Field{zap.String("subject", msg.Subject())},
		}
	}
	return nil
}

func (r *natsReceiver) handleMetricsMessage(ctx context.Context, msg otelnats.MessageSignal[metricspb.MetricsData]) error {
	ctx = r.obsrecv.StartMetricsOp(ctx)

	// Choose unmarshaler based on Content-Type header
	contentType := msg.Headers().Get(otelnats.HeaderContentType)
	var metrics pmetric.Metrics
	var err error

	if contentType == otelnats.ContentTypeJSON {
		unmarshaler := &pmetric.JSONUnmarshaler{}
		metrics, err = unmarshaler.UnmarshalMetrics(msg.Data())
	} else {
		// Default to protobuf (application/x-protobuf or empty)
		metrics, err = r.metricsUnmarshaler.UnmarshalMetrics(msg.Data())
	}

	if err != nil {
		r.obsrecv.EndMetricsOp(ctx, contentType, 0, err)
		r.logger.Error("failed to unmarshal metrics",
			zap.String("subject", msg.Subject()),
			zap.String("content_type", contentType),
			zap.Error(err),
		)
		return err
	}

	dataPointCount := metrics.DataPointCount()
	err = r.metricsConsumer.ConsumeMetrics(ctx, metrics)
	r.obsrecv.EndMetricsOp(ctx, contentType, dataPointCount, err)

	if err != nil {
		return receiverError{
			err:    err,
			fields: []zap.Field{zap.String("subject", msg.Subject())},
		}
	}
	return nil
}

func (r *natsReceiver) handleLogsMessage(ctx context.Context, msg otelnats.MessageSignal[logspb.LogsData]) error {
	ctx = r.obsrecv.StartLogsOp(ctx)

	// Choose unmarshaler based on Content-Type header
	contentType := msg.Headers().Get(otelnats.HeaderContentType)
	var logs plog.Logs
	var err error

	if contentType == otelnats.ContentTypeJSON {
		unmarshaler := &plog.JSONUnmarshaler{}
		logs, err = unmarshaler.UnmarshalLogs(msg.Data())
	} else {
		// Default to protobuf (application/x-protobuf or empty)
		logs, err = r.logsUnmarshaler.UnmarshalLogs(msg.Data())
	}

	if err != nil {
		r.obsrecv.EndLogsOp(ctx, contentType, 0, err)
		r.logger.Error("failed to unmarshal logs",
			zap.String("subject", msg.Subject()),
			zap.String("content_type", contentType),
			zap.Error(err),
		)
		return err
	}

	logCount := logs.LogRecordCount()
	err = r.logsConsumer.ConsumeLogs(ctx, logs)
	r.obsrecv.EndLogsOp(ctx, contentType, logCount, err)

	if err != nil {
		return receiverError{
			err:    err,
			fields: []zap.Field{zap.String("subject", msg.Subject())},
		}
	}
	return nil
}

func (r *natsReceiver) handleError(err error) {
	level := zap.ErrorLevel
	fields := []zap.Field{zap.Error(err)}
	var errObj receiverError
	if errors.As(err, &errObj) {
		fields = append(fields, errObj.fields...)
	}
	if consumererror.IsDownstream(err) {
		level = r.downstreamErrLevel
	}
	r.logger.Log(level, "NATS receiver error", fields...)
}

type receiverError struct {
	err    error
	fields []zap.Field
}

func (r receiverError) Error() string {
	return r.err.Error()
}
