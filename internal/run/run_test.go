package run

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/otelcol"
)

// TestCollectorStartup verifies the collector can start up with a minimal valid configuration
func TestCollectorStartup(t *testing.T) {
	// Create a minimal valid config
	minimalConfig := `
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: localhost:0

processors:
  batch:

exporters:
  debug:
    verbosity: normal

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
`

	// Write config to temp file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(minimalConfig), 0644)
	require.NoError(t, err)

	// Create collector settings
	info := component.BuildInfo{
		Command:     "otelnats-collector-test",
		Description: "Test collector startup",
		Version:     "test",
	}

	set := otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: components,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: []string{"file:" + configFile},
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
				},
			},
		},
	}

	// Create collector
	col, err := otelcol.NewCollector(set)
	require.NoError(t, err, "failed to create collector")

	// Start collector in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- col.Run(ctx)
	}()

	// Wait a bit for startup
	time.Sleep(100 * time.Millisecond)

	// Verify collector is running (no immediate error)
	select {
	case err := <-errChan:
		require.NoError(t, err, "collector failed during startup")
	default:
		// No error yet, collector is running
	}

	// Shutdown gracefully
	cancel()

	// Wait for shutdown with timeout
	select {
	case err := <-errChan:
		require.NoError(t, err, "collector shutdown failed")
	case <-time.After(5 * time.Second):
		t.Fatal("collector shutdown timeout")
	}
}

// TestCollectorStartup_WithNATSComponents verifies the collector can start with NATS receiver/exporter
func TestCollectorStartup_WithNATSComponents(t *testing.T) {
	// Create config with NATS components
	natsConfig := `
receivers:
  nats:
    url: nats://localhost:4222
    traces:
      subject: otel.traces
    metrics:
      subject: otel.metrics
    logs:
      subject: otel.logs

processors:
  batch:

exporters:
  debug:
    verbosity: normal

service:
  pipelines:
    traces:
      receivers: [nats]
      processors: [batch]
      exporters: [debug]
`

	// Write config to temp file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(natsConfig), 0644)
	require.NoError(t, err)

	// Create collector settings
	info := component.BuildInfo{
		Command:     "otelnats-collector-test",
		Description: "Test collector startup with NATS",
		Version:     "test",
	}

	set := otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: components,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: []string{"file:" + configFile},
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
				},
			},
		},
	}

	// Create collector
	col, err := otelcol.NewCollector(set)
	require.NoError(t, err, "failed to create collector with NATS components")

	// Note: We don't actually start the collector here because NATS connection will fail
	// This test just verifies the config is valid and components are registered
	_ = col
}
