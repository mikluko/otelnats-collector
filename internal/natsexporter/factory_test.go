package natsexporter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"

	"github.com/mikluko/nats-otel-collector/internal/metadata"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory)
	assert.Equal(t, metadata.Type, factory.Type())
}

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NotNil(t, cfg)

	natsConfig, ok := cfg.(*Config)
	require.True(t, ok)

	// Check default values
	assert.Equal(t, "nats://localhost:4222", natsConfig.ClientConfig.URL)
	assert.Equal(t, 5*time.Second, natsConfig.TimeoutConfig.Timeout)
}

func TestCreateTracesExporter(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Traces.Subject = "test.traces"

	ctx := context.Background()
	set := exportertest.NewNopSettings(metadata.Type)

	exp, err := factory.CreateTraces(ctx, set, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)

	// Verify it can start (won't connect without NATS, but should not panic)
	err = exp.Start(ctx, componenttest.NewNopHost())
	// Expected to fail connecting to NATS, but the exporter should be created
	assert.Error(t, err)
}

func TestCreateMetricsExporter(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Metrics.Subject = "test.metrics"

	ctx := context.Background()
	set := exportertest.NewNopSettings(metadata.Type)

	exp, err := factory.CreateMetrics(ctx, set, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
}

func TestCreateLogsExporter(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Logs.Subject = "test.logs"

	ctx := context.Background()
	set := exportertest.NewNopSettings(metadata.Type)

	exp, err := factory.CreateLogs(ctx, set, cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
}
