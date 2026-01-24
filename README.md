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
- **Production Ready**: Deploys via official OpenTelemetry Collector Helm chart with custom image

## Components

### Custom Components

| Component | Type | Description |
|-----------|------|-------------|
| `nats` | Receiver | Subscribe to NATS subjects and ingest telemetry |
| `nats` | Exporter | Publish telemetry to NATS subjects |

### Included OTel Contrib Components

For DaemonSet/scraping use cases:

| Component | Type | Description |
|-----------|------|-------------|
| `prometheus` | Receiver | Scrape Prometheus metrics from pods |
| `filelog` | Receiver | Collect container logs from node filesystem |
| `hostmetrics` | Receiver | Collect node-level system metrics |
| `k8sattributes` | Processor | Enrich telemetry with Kubernetes metadata |
| `resourcedetection` | Processor | Detect cloud provider metadata |

Standard processors (`batch`, `memory_limiter`, `transform`) and exporters (`otlp`, `otlphttp`, `debug`) are also included.

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

Use the official OpenTelemetry Collector Helm chart with the custom NATS-enabled image:

```bash
# Add the official OpenTelemetry Helm repository
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
```

#### Gateway Mode (OTLP → NATS)

```bash
helm install otelnats-gateway open-telemetry/opentelemetry-collector \
  -f examples/helm/gateway-values.yaml
```

#### Ingest Mode (NATS → Backend)

Core NATS with queue groups:
```bash
helm install otelnats-ingest open-telemetry/opentelemetry-collector \
  -f examples/helm/ingest-values.yaml
```

JetStream with at-least-once delivery:
```bash
helm install otelnats-ingest open-telemetry/opentelemetry-collector \
  -f examples/helm/ingest-jetstream-values.yaml
```

#### DaemonSet Mode (Node Scraping → NATS)

Deploy as a DaemonSet to collect metrics and logs directly from Kubernetes nodes:

```bash
helm install otelnats-daemonset open-telemetry/opentelemetry-collector \
  -f examples/helm/daemonset-values.yaml
```

This mode enables:
- Prometheus metrics scraping from pods with `prometheus.io/scrape: "true"` annotations
- Container log collection from `/var/log/pods`
- Node-level metrics (CPU, memory, disk, network)
- Kubernetes metadata enrichment (pod, namespace, deployment labels)

##### RBAC Requirements

The DaemonSet mode requires additional RBAC permissions for the `k8sattributes` processor and Prometheus service discovery:

```yaml
rules:
  - apiGroups: [""]
    resources: ["pods", "namespaces", "nodes"]
    verbs: ["get", "watch", "list"]
  - apiGroups: ["apps"]
    resources: ["replicasets", "deployments", "daemonsets", "statefulsets"]
    verbs: ["get", "watch", "list"]
```

##### Security Considerations

DaemonSet deployment requires elevated privileges:

| Requirement | Purpose |
|-------------|---------|
| `hostPath` volumes | Access `/var/log/pods` for container logs |
| `runAsUser: 0` | Read log files owned by root |
| Network access to kubelet | Prometheus service discovery |

**Recommendations:**
- Use dedicated ServiceAccount with minimal required permissions
- Consider PodSecurityPolicy/PodSecurityStandard exemptions for the collector namespace
- Restrict NATS credentials to publish-only permissions
- Use network policies to limit collector egress to NATS endpoints only

See [examples/helm/](./examples/helm/) for complete values file examples.

## Configuration

See [examples/](./examples/) directory for complete configuration examples:
- `examples/gateway/`: NATS as telemetry gateway (OTLP → NATS)
- `examples/ingest/`: NATS as telemetry source (NATS → backend)
- `examples/daemonset/`: Node-level scraping (Prometheus/logs → NATS)

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
