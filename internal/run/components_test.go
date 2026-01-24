package run

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
)

func TestComponents_HealthCheckExtensionRegistered(t *testing.T) {
	factories, err := components()
	require.NoError(t, err)

	// Verify health_check extension is registered
	_, ok := factories.Extensions[component.MustNewType("health_check")]
	assert.True(t, ok, "health_check extension should be registered")
}

func TestComponents_AllExpectedExtensions(t *testing.T) {
	factories, err := components()
	require.NoError(t, err)

	expectedExtensions := []string{"health_check", "zpages"}
	for _, ext := range expectedExtensions {
		_, ok := factories.Extensions[component.MustNewType(ext)]
		assert.True(t, ok, "extension %s should be registered", ext)
	}
}

func TestComponents_AllExpectedReceivers(t *testing.T) {
	factories, err := components()
	require.NoError(t, err)

	expectedReceivers := []string{"otlp", "nats", "prometheus", "filelog", "hostmetrics"}
	for _, rcv := range expectedReceivers {
		_, ok := factories.Receivers[component.MustNewType(rcv)]
		assert.True(t, ok, "receiver %s should be registered", rcv)
	}
}

func TestComponents_AllExpectedExporters(t *testing.T) {
	factories, err := components()
	require.NoError(t, err)

	expectedExporters := []string{"otlp", "otlphttp", "nats", "debug"}
	for _, exp := range expectedExporters {
		_, ok := factories.Exporters[component.MustNewType(exp)]
		assert.True(t, ok, "exporter %s should be registered", exp)
	}
}

func TestComponents_AllExpectedProcessors(t *testing.T) {
	factories, err := components()
	require.NoError(t, err)

	expectedProcessors := []string{"batch", "memory_limiter", "transform", "k8sattributes", "resourcedetection"}
	for _, proc := range expectedProcessors {
		_, ok := factories.Processors[component.MustNewType(proc)]
		assert.True(t, ok, "processor %s should be registered", proc)
	}
}
