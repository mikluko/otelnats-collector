package run

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/zpagesextension"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

	"github.com/mikluko/otelnats-collector/internal/natsexporter"
	"github.com/mikluko/otelnats-collector/internal/natsreceiver"
)

func components() (otelcol.Factories, error) {
	var err error
	factories := otelcol.Factories{}

	// Extensions
	factories.Extensions, err = otelcol.MakeFactoryMap[extension.Factory](
		healthcheckextension.NewFactory(),
		zpagesextension.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}

	// Receivers
	factories.Receivers, err = otelcol.MakeFactoryMap[receiver.Factory](
		otlpreceiver.NewFactory(),
		natsreceiver.NewFactory(),
		prometheusreceiver.NewFactory(),
		filelogreceiver.NewFactory(),
		hostmetricsreceiver.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}

	// Exporters
	factories.Exporters, err = otelcol.MakeFactoryMap[exporter.Factory](
		debugexporter.NewFactory(),
		otlpexporter.NewFactory(),
		otlphttpexporter.NewFactory(),
		natsexporter.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}

	// Processors
	factories.Processors, err = otelcol.MakeFactoryMap[processor.Factory](
		batchprocessor.NewFactory(),
		memorylimiterprocessor.NewFactory(),
		transformprocessor.NewFactory(),
		k8sattributesprocessor.NewFactory(),
		resourcedetectionprocessor.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}

	// Connectors (empty for now, but required)
	factories.Connectors, err = otelcol.MakeFactoryMap[connector.Factory]()
	if err != nil {
		return otelcol.Factories{}, err
	}

	// Telemetry
	factories.Telemetry = otelconftelemetry.NewFactory()

	return factories, nil
}
