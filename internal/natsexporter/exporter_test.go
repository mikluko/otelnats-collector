package natsexporter

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/mikluko/otelnats-collector/internal/metadata"
	"github.com/mikluko/otelnats-collector/internal/testutil"
)

func TestE2E_Traces(t *testing.T) {
	ns := testutil.StartEmbeddedNATS(t)
	ctx := context.Background()

	// Subscribe to receive traces
	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	received := make(chan []byte, 1)
	sub, err := nc.Subscribe("test.traces", func(msg *nats.Msg) {
		received <- msg.Data
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Create exporter
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.URL = ns.ClientURL()
	cfg.Traces.Subject = "test.traces"

	set := exportertest.NewNopSettings(metadata.Type)
	exp, err := factory.CreateTraces(ctx, set, cfg)
	require.NoError(t, err)

	err = exp.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer exp.Shutdown(ctx)

	// Create and send test traces
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "test-service")
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetName("test-span")
	span.SetTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	span.SetSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})

	err = exp.ConsumeTraces(ctx, traces)
	require.NoError(t, err)

	// Verify received
	select {
	case data := <-received:
		unmarshaler := &ptrace.ProtoUnmarshaler{}
		got, err := unmarshaler.UnmarshalTraces(data)
		require.NoError(t, err)
		assert.Equal(t, 1, got.SpanCount())
		assert.Equal(t, "test-span", got.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name())
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for traces")
	}
}

func TestE2E_Metrics(t *testing.T) {
	ns := testutil.StartEmbeddedNATS(t)
	ctx := context.Background()

	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	received := make(chan []byte, 1)
	sub, err := nc.Subscribe("test.metrics", func(msg *nats.Msg) {
		received <- msg.Data
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.URL = ns.ClientURL()
	cfg.Metrics.Subject = "test.metrics"

	set := exportertest.NewNopSettings(metadata.Type)
	exp, err := factory.CreateMetrics(ctx, set, cfg)
	require.NoError(t, err)

	err = exp.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer exp.Shutdown(ctx)

	// Create and send test metrics
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", "test-service")
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("test.counter")
	m.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(42)

	err = exp.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	select {
	case data := <-received:
		unmarshaler := &pmetric.ProtoUnmarshaler{}
		got, err := unmarshaler.UnmarshalMetrics(data)
		require.NoError(t, err)
		assert.Equal(t, 1, got.DataPointCount())
		assert.Equal(t, "test.counter", got.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name())
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for metrics")
	}
}

func TestE2E_Logs(t *testing.T) {
	ns := testutil.StartEmbeddedNATS(t)
	ctx := context.Background()

	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	received := make(chan []byte, 1)
	sub, err := nc.Subscribe("test.logs", func(msg *nats.Msg) {
		received <- msg.Data
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.ClientConfig.URL = ns.ClientURL()
	cfg.Logs.Subject = "test.logs"

	set := exportertest.NewNopSettings(metadata.Type)
	exp, err := factory.CreateLogs(ctx, set, cfg)
	require.NoError(t, err)

	err = exp.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	defer exp.Shutdown(ctx)

	// Create and send test logs
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr("service.name", "test-service")
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()
	lr.Body().SetStr("test log message")

	err = exp.ConsumeLogs(ctx, logs)
	require.NoError(t, err)

	select {
	case data := <-received:
		unmarshaler := &plog.ProtoUnmarshaler{}
		got, err := unmarshaler.UnmarshalLogs(data)
		require.NoError(t, err)
		assert.Equal(t, 1, got.LogRecordCount())
		assert.Equal(t, "test log message", got.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Str())
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for logs")
	}
}
