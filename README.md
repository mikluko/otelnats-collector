# NATS OpenTelemetry Collector

Custom OpenTelemetry Collector distribution with NATS receiver and exporter components for streaming telemetry data through NATS messaging infrastructure.

## Motivation

### Why Not OpenTelemetry Contrib?

This project exists as a standalone distribution due to the [OpenTelemetry Collector Contrib sponsorship requirements](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md#becoming-a-code-owner).

**Background:**
- **Issue [#39540](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/39540)**: NATS receiver/exporter proposal was accepted as legitimate and remains open with "Sponsor Needed" label
- **PR [#42186](https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/42186)**: Initial implementation was submitted but closed due to lack of community sponsorship

The upstream project requires new components to have an established sponsor (code owner) who commits to long-term maintenance. Without organizational backing from a recognized contributor, components cannot be merged into the contrib repository.

Rather than wait indefinitely for sponsorship, this project delivers NATS integration as a custom collector distribution that can be maintained independently.

## Features

- **NATS Receiver**: Ingest telemetry data (traces, metrics, logs) from NATS subjects
- **NATS Exporter**: Stream telemetry data to NATS subjects
- **Standard OTel Ecosystem**: Full compatibility with OpenTelemetry Collector processors, exporters, and extensions
- **Production Ready**: Includes Kubernetes Helm chart with HPA, PDB, and topology spread constraints

## Components

### Receiver: `nats`
Subscribes to NATS subjects and ingests telemetry data into the collector pipeline.

### Exporter: `nats`
Publishes telemetry data to NATS subjects for downstream consumption.

## Installation

### Binary

```bash
# Build from source
make build

# Binary will be available at ./bin/nats-otel-collector
./bin/nats-otel-collector --config=config.yaml
```

### Docker

```bash
# Build Docker image
make docker-build

# Run with configuration
docker run -v $(pwd)/config.yaml:/etc/otel/config.yaml \
  nats-otel-collector:latest --config=/etc/otel/config.yaml
```

### Kubernetes (Helm)

```bash
helm install nats-otel-collector ./deploy/helm/nats-otel-collector \
  --set config.receivers.nats.url="nats://nats:4222"
```

## Configuration

See [examples/](./examples/) directory for complete configuration examples:
- `examples/gateway/`: NATS as telemetry gateway
- `examples/ingest/`: NATS as telemetry source

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Build binary
make build
```

## License

Apache 2.0

## Contributing

Contributions welcome. This project follows standard Go and OpenTelemetry conventions.
