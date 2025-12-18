package run

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponents_HealthCheckExtensionRegistered(t *testing.T) {
	factories, err := components()
	require.NoError(t, err)

	// Verify health_check extension is registered
	_, ok := factories.Extensions["health_check"]
	assert.True(t, ok, "health_check extension should be registered")
}

func TestComponents_AllExpectedExtensions(t *testing.T) {
	factories, err := components()
	require.NoError(t, err)

	expectedExtensions := []string{"health_check", "zpages"}
	for _, ext := range expectedExtensions {
		_, ok := factories.Extensions[ext]
		assert.True(t, ok, "extension %s should be registered", ext)
	}
}

func TestComponents_AllExpectedReceivers(t *testing.T) {
	factories, err := components()
	require.NoError(t, err)

	expectedReceivers := []string{"otlp", "nats"}
	for _, rcv := range expectedReceivers {
		_, ok := factories.Receivers[rcv]
		assert.True(t, ok, "receiver %s should be registered", rcv)
	}
}

func TestComponents_AllExpectedExporters(t *testing.T) {
	factories, err := components()
	require.NoError(t, err)

	expectedExporters := []string{"otlp", "otlphttp", "nats", "debug"}
	for _, exp := range expectedExporters {
		_, ok := factories.Exporters[exp]
		assert.True(t, ok, "exporter %s should be registered", exp)
	}
}

func TestComponents_AllExpectedProcessors(t *testing.T) {
	factories, err := components()
	require.NoError(t, err)

	expectedProcessors := []string{"batch", "memory_limiter"}
	for _, proc := range expectedProcessors {
		_, ok := factories.Processors[proc]
		assert.True(t, ok, "processor %s should be registered", proc)
	}
}
