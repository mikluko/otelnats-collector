package natsreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/mikluko/otelnats-collector/internal/metadata"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory)
	assert.Equal(t, metadata.Type, factory.Type())
}

const testDefaultURL = "nats://localhost:4222"

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NotNil(t, cfg)

	natsConfig, ok := cfg.(*Config)
	require.True(t, ok)

	// Check default values
	assert.Equal(t, testDefaultURL, natsConfig.ClientConfig.URL)
}

func TestCreateTracesReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Traces.Subject = "test.traces"

	ctx := context.Background()
	set := receivertest.NewNopSettings(metadata.Type)

	rec, err := factory.CreateTraces(ctx, set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, rec)
}

func TestCreateMetricsReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Metrics.Subject = "test.metrics"

	ctx := context.Background()
	set := receivertest.NewNopSettings(metadata.Type)

	rec, err := factory.CreateMetrics(ctx, set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, rec)
}

func TestCreateLogsReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Logs.Subject = "test.logs"

	ctx := context.Background()
	set := receivertest.NewNopSettings(metadata.Type)

	rec, err := factory.CreateLogs(ctx, set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, rec)
}

func TestCreateTracesReceiver_EmptySubject(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	// Clear the default subject - receiver will still be created,
	// but will fail when Start() is called (if NATS is available)
	cfg.Traces.Subject = ""

	ctx := context.Background()
	set := receivertest.NewNopSettings(metadata.Type)

	// Receiver creation should succeed even with empty subject
	// (validation happens at Start() time when subscribing)
	rec, err := factory.CreateTraces(ctx, set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, rec)
}
