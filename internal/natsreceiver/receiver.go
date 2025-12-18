package natsreceiver

import (
	"context"
	"sync"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"

	internalnats "github.com/mikluko/opentelemetry-collector-nats/internal/nats"
)

type natsReceiver struct {
	config       *Config
	signalConfig SignalConfig
	settings     receiver.Settings
	logger       *zap.Logger
	obsrecv      *receiverhelper.ObsReport

	conn *nats.Conn
	sub  *nats.Subscription

	tracesUnmarshaler  ptrace.Unmarshaler
	metricsUnmarshaler pmetric.Unmarshaler
	logsUnmarshaler    plog.Unmarshaler

	tracesConsumer  consumer.Traces
	metricsConsumer consumer.Metrics
	logsConsumer    consumer.Logs

	shutdownWG sync.WaitGroup
}

func newNatsReceiver(
	cfg *Config,
	set receiver.Settings,
	signalConfig SignalConfig,
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
		signalConfig:    signalConfig,
		settings:        set,
		logger:          set.Logger,
		obsrecv:         obsrecv,
		tracesConsumer:  tracesConsumer,
		metricsConsumer: metricsConsumer,
		logsConsumer:    logsConsumer,
	}, nil
}

func (r *natsReceiver) Start(ctx context.Context, _ component.Host) error {
	conn, err := internalnats.Connect(ctx, r.config.ClientConfig, r.logger)
	if err != nil {
		return err
	}
	r.conn = conn

	// Initialize unmarshalers
	r.tracesUnmarshaler = &ptrace.ProtoUnmarshaler{}
	r.metricsUnmarshaler = &pmetric.ProtoUnmarshaler{}
	r.logsUnmarshaler = &plog.ProtoUnmarshaler{}

	// Subscribe with queue group for load balancing
	var sub *nats.Subscription
	if r.signalConfig.QueueGroup != "" {
		sub, err = conn.QueueSubscribe(
			r.signalConfig.Subject,
			r.signalConfig.QueueGroup,
			r.handleMessage,
		)
	} else {
		sub, err = conn.Subscribe(
			r.signalConfig.Subject,
			r.handleMessage,
		)
	}
	if err != nil {
		return err
	}
	r.sub = sub

	r.logger.Info("NATS receiver started",
		zap.String("url", r.config.URL),
		zap.String("subject", r.signalConfig.Subject),
		zap.String("queue_group", r.signalConfig.QueueGroup),
	)
	return nil
}

func (r *natsReceiver) Shutdown(ctx context.Context) error {
	if r.sub != nil {
		// Drain unsubscribes and processes remaining messages
		if err := r.sub.Drain(); err != nil {
			r.logger.Warn("error draining subscription", zap.Error(err))
		}
	}

	// Wait for in-flight message handlers to complete
	r.shutdownWG.Wait()

	if r.conn != nil {
		r.conn.Close()
	}
	return nil
}

func (r *natsReceiver) handleMessage(msg *nats.Msg) {
	r.shutdownWG.Add(1)
	defer r.shutdownWG.Done()

	ctx := context.Background()

	r.logger.Debug("received NATS message",
		zap.String("subject", msg.Subject),
		zap.Int("size", len(msg.Data)),
	)

	var err error
	switch {
	case r.tracesConsumer != nil:
		err = r.processTraces(ctx, msg.Data)
	case r.metricsConsumer != nil:
		err = r.processMetrics(ctx, msg.Data)
	case r.logsConsumer != nil:
		err = r.processLogs(ctx, msg.Data)
	}

	if err != nil {
		r.logger.Error("failed to process message",
			zap.String("subject", msg.Subject),
			zap.Error(err),
		)
	}
}

func (r *natsReceiver) processTraces(ctx context.Context, data []byte) error {
	ctx = r.obsrecv.StartTracesOp(ctx)

	traces, err := r.tracesUnmarshaler.UnmarshalTraces(data)
	if err != nil {
		r.obsrecv.EndTracesOp(ctx, r.signalConfig.Encoding, 0, err)
		return consumererror.NewPermanent(err)
	}

	spanCount := traces.SpanCount()
	err = r.tracesConsumer.ConsumeTraces(ctx, traces)
	r.obsrecv.EndTracesOp(ctx, r.signalConfig.Encoding, spanCount, err)
	return err
}

func (r *natsReceiver) processMetrics(ctx context.Context, data []byte) error {
	ctx = r.obsrecv.StartMetricsOp(ctx)

	metrics, err := r.metricsUnmarshaler.UnmarshalMetrics(data)
	if err != nil {
		r.obsrecv.EndMetricsOp(ctx, r.signalConfig.Encoding, 0, err)
		return consumererror.NewPermanent(err)
	}

	dataPointCount := metrics.DataPointCount()
	err = r.metricsConsumer.ConsumeMetrics(ctx, metrics)
	r.obsrecv.EndMetricsOp(ctx, r.signalConfig.Encoding, dataPointCount, err)
	return err
}

func (r *natsReceiver) processLogs(ctx context.Context, data []byte) error {
	ctx = r.obsrecv.StartLogsOp(ctx)

	logs, err := r.logsUnmarshaler.UnmarshalLogs(data)
	if err != nil {
		r.obsrecv.EndLogsOp(ctx, r.signalConfig.Encoding, 0, err)
		return consumererror.NewPermanent(err)
	}

	logCount := logs.LogRecordCount()
	err = r.logsConsumer.ConsumeLogs(ctx, logs)
	r.obsrecv.EndLogsOp(ctx, r.signalConfig.Encoding, logCount, err)
	return err
}
