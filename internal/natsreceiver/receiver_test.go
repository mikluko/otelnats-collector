package natsreceiver

import (
	"context"
	"testing"
	"time"

	"github.com/mikluko/otelnats"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/mikluko/otelnats-collector/internal/metadata"
	"github.com/mikluko/otelnats-collector/internal/testutil"
)

func TestE2E_ReceiveTraces(t *testing.T) {
	ns := testutil.StartEmbeddedNATS(t)
	ctx := context.Background()

	// Create sink to capture received traces
	sink := &consumertest.TracesSink{}

	// Create receiver
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.URL = ns.ClientURL()
	cfg.Traces.Subject = "test.traces"
	cfg.Traces.QueueGroup = "" // No queue group for test

	set := receivertest.NewNopSettings(metadata.Type)
	rcv, err := factory.CreateTraces(ctx, set, cfg, sink)
	require.NoError(t, err)

	err = rcv.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer rcv.Shutdown(ctx)

	// Connect to NATS and publish test traces
	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	// Create test traces
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "test-service")
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetName("test-span")
	span.SetTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	span.SetSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})

	// Marshal and publish with SDK headers
	marshaler := &ptrace.ProtoMarshaler{}
	data, err := marshaler.MarshalTraces(traces)
	require.NoError(t, err)

	// Add otelnats protocol headers for SDK receiver
	headers := otelnats.BuildHeaders(ctx, otelnats.SignalTraces, otelnats.EncodingProtobuf, nil)
	msg := &nats.Msg{
		Subject: "test.traces",
		Data:    data,
		Header:  headers,
	}

	err = nc.PublishMsg(msg)
	require.NoError(t, err)
	nc.Flush()

	// Wait for receiver to process
	require.Eventually(t, func() bool {
		return sink.SpanCount() > 0
	}, 5*time.Second, 10*time.Millisecond)

	// Verify received traces
	assert.Equal(t, 1, sink.SpanCount())
	got := sink.AllTraces()[0]
	assert.Equal(t, "test-span", got.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())
}

func TestE2E_ReceiveMetrics(t *testing.T) {
	ns := testutil.StartEmbeddedNATS(t)
	ctx := context.Background()

	sink := &consumertest.MetricsSink{}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.URL = ns.ClientURL()
	cfg.Metrics.Subject = "test.metrics"
	cfg.Metrics.QueueGroup = ""

	set := receivertest.NewNopSettings(metadata.Type)
	rcv, err := factory.CreateMetrics(ctx, set, cfg, sink)
	require.NoError(t, err)

	err = rcv.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer rcv.Shutdown(ctx)

	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	// Create test metrics
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", "test-service")
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("test.counter")
	m.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(42)

	marshaler := &pmetric.ProtoMarshaler{}
	data, err := marshaler.MarshalMetrics(metrics)
	require.NoError(t, err)

	// Add otelnats protocol headers for SDK receiver
	headers := otelnats.BuildHeaders(ctx, otelnats.SignalMetrics, otelnats.EncodingProtobuf, nil)
	msg := &nats.Msg{
		Subject: "test.metrics",
		Data:    data,
		Header:  headers,
	}

	err = nc.PublishMsg(msg)
	require.NoError(t, err)
	nc.Flush()

	require.Eventually(t, func() bool {
		return sink.DataPointCount() > 0
	}, 5*time.Second, 10*time.Millisecond)

	assert.Equal(t, 1, sink.DataPointCount())
	got := sink.AllMetrics()[0]
	assert.Equal(t, "test.counter", got.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())
}

func TestE2E_ReceiveLogs(t *testing.T) {
	ns := testutil.StartEmbeddedNATS(t)
	ctx := context.Background()

	sink := &consumertest.LogsSink{}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.URL = ns.ClientURL()
	cfg.Logs.Subject = "test.logs"
	cfg.Logs.QueueGroup = ""

	set := receivertest.NewNopSettings(metadata.Type)
	rcv, err := factory.CreateLogs(ctx, set, cfg, sink)
	require.NoError(t, err)

	err = rcv.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer rcv.Shutdown(ctx)

	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	// Create test logs
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr("service.name", "test-service")
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()
	lr.Body().SetStr("test log message")

	marshaler := &plog.ProtoMarshaler{}
	data, err := marshaler.MarshalLogs(logs)
	require.NoError(t, err)

	// Add otelnats protocol headers for SDK receiver
	headers := otelnats.BuildHeaders(ctx, otelnats.SignalLogs, otelnats.EncodingProtobuf, nil)
	msg := &nats.Msg{
		Subject: "test.logs",
		Data:    data,
		Header:  headers,
	}

	err = nc.PublishMsg(msg)
	require.NoError(t, err)
	nc.Flush()

	require.Eventually(t, func() bool {
		return sink.LogRecordCount() > 0
	}, 5*time.Second, 10*time.Millisecond)

	assert.Equal(t, 1, sink.LogRecordCount())
	got := sink.AllLogs()[0]
	assert.Equal(t, "test log message", got.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Str())
}

// TestE2E_MetricsOnlyReceiver_DoesNotSubscribeToOtherSignals verifies that a
// metrics-only receiver doesn't subscribe to logs/traces subjects even when they
// are configured in the default config. This is a regression test for the bug
// where receivers would subscribe to all configured subjects regardless of which
// signal they were created for, causing "no message handler configured" errors.
func TestE2E_MetricsOnlyReceiver_DoesNotSubscribeToOtherSignals(t *testing.T) {
	ns := testutil.StartEmbeddedNATS(t)
	ctx := context.Background()

	sink := &consumertest.MetricsSink{}

	// Create metrics-only receiver with default config (which has ALL signal subjects set)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.URL = ns.ClientURL()
	// Default config has traces, metrics, and logs subjects all configured

	set := receivertest.NewNopSettings(metadata.Type)
	rcv, err := factory.CreateMetrics(ctx, set, cfg, sink)
	require.NoError(t, err)

	err = rcv.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer rcv.Shutdown(ctx)

	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	// Publish log message to the logs subject (receiver should NOT be subscribed)
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()
	lr.Body().SetStr("this should be ignored")

	marshaler := &plog.ProtoMarshaler{}
	data, err := marshaler.MarshalLogs(logs)
	require.NoError(t, err)

	headers := otelnats.BuildHeaders(ctx, otelnats.SignalLogs, otelnats.EncodingProtobuf, nil)
	msg := &nats.Msg{
		Subject: cfg.Logs.Subject, // Using default logs subject
		Data:    data,
		Header:  headers,
	}

	err = nc.PublishMsg(msg)
	require.NoError(t, err)
	nc.Flush()

	// Wait a bit to ensure receiver would have processed if subscribed
	time.Sleep(100 * time.Millisecond)

	// Verify NO logs were received (because we shouldn't be subscribed)
	assert.Equal(t, 0, sink.DataPointCount(), "metrics receiver should not receive logs")

	// Now publish a metrics message to verify receiver is working
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", "test-service")
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("test.counter")
	m.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(42)

	metricsMarshaler := &pmetric.ProtoMarshaler{}
	metricsData, err := metricsMarshaler.MarshalMetrics(metrics)
	require.NoError(t, err)

	metricsHeaders := otelnats.BuildHeaders(ctx, otelnats.SignalMetrics, otelnats.EncodingProtobuf, nil)
	metricsMsg := &nats.Msg{
		Subject: cfg.Metrics.Subject,
		Data:    metricsData,
		Header:  metricsHeaders,
	}

	err = nc.PublishMsg(metricsMsg)
	require.NoError(t, err)
	nc.Flush()

	// Verify metrics ARE received
	require.Eventually(t, func() bool {
		return sink.DataPointCount() > 0
	}, 5*time.Second, 10*time.Millisecond)

	assert.Equal(t, 1, sink.DataPointCount())
}
