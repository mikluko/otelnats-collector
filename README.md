# NATS OpenTelemetry Collector

Custom OpenTelemetry Collector distribution with NATS receiver and exporter components for streaming telemetry data through NATS messaging infrastructure.

## Motivation

This project exists as a standalone distribution due to the [OpenTelemetry Collector Contrib sponsorship requirements](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md#becoming-a-code-owner).

- **Issue [#39540](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/39540)**: NATS receiver/exporter proposal accepted, remains open with "Sponsor Needed" label
- **PR [#42186](https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/42186)**: Implementation submitted but closed due to lack of community sponsorship

Rather than wait for sponsorship, this project delivers NATS integration as a custom collector distribution maintained independently.

## Components

### NATS (custom)

| Component | Type | Description |
|-----------|------|-------------|
| `nats` | Receiver | Subscribe to NATS subjects and ingest OTLP telemetry (Core NATS or JetStream) |
| `nats` | Exporter | Publish OTLP telemetry to NATS subjects |

### OTel Contrib

| Component | Type | Description |
|-----------|------|-------------|
| `prometheus` | Receiver | Scrape Prometheus metrics endpoints |
| `filelog` | Receiver | Collect container logs from node filesystem |
| `hostmetrics` | Receiver | Collect node-level system metrics |
| `k8sattributes` | Processor | Enrich telemetry with Kubernetes metadata |
| `resourcedetection` | Processor | Detect cloud/infrastructure metadata |
| `transform` | Processor | Modify telemetry using OTTL statements |

### OTel Core

| Component | Type |
|-----------|------|
| `otlp` | Receiver |
| `otlp`, `otlphttp` | Exporters |
| `debug` | Exporter |
| `batch`, `memory_limiter` | Processors |
| `health_check`, `zpages` | Extensions |

## Kubernetes Deployment

The intended deployment model uses the official [opentelemetry-collector](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-collector) Helm chart with the custom image override. No custom Helm chart is needed.

### Image

Container images are published to GHCR:

```
ghcr.io/mikluko/otelnats-collector:<version>
```

The image is a drop-in replacement for the standard `otel/opentelemetry-collector-contrib` image. Override it in the chart values:

```yaml
image:
  repository: ghcr.io/mikluko/otelnats-collector
  tag: "0.3.1"
```

### Helm Setup

```bash
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
```

The chart supports two deployment modes via the `mode` value: `deployment` and `daemonset`. Different roles in the telemetry pipeline call for different modes.

### Deployment Mode

Use `mode: deployment` for stateless collector instances that receive telemetry via OTLP or consume from NATS and forward to backends.

**Gateway (OTLP -> NATS)** — accepts OTLP from applications, publishes to NATS subjects:

```yaml
image:
  repository: ghcr.io/mikluko/otelnats-collector
  tag: "0.3.1"

mode: deployment

config:
  receivers:
    otlp:
      protocols:
        grpc:
          endpoint: 0.0.0.0:4317
        http:
          endpoint: 0.0.0.0:4318
  exporters:
    nats:
      url: nats://nats.nats-system:4222
      auth:
        credentials_file: /mnt/secrets/nats.creds
      traces:
        subject: otel.traces
      metrics:
        subject: otel.metrics
      logs:
        subject: otel.logs
  service:
    pipelines:
      traces:
        receivers: [otlp]
        processors: [memory_limiter, batch]
        exporters: [nats]
      metrics:
        receivers: [otlp]
        processors: [memory_limiter, batch]
        exporters: [nats]
      logs:
        receivers: [otlp]
        processors: [memory_limiter, batch]
        exporters: [nats]

extraVolumes:
  - name: nats-creds
    secret:
      secretName: nats-creds
extraVolumeMounts:
  - name: nats-creds
    mountPath: /mnt/secrets/nats.creds
    subPath: nats.creds
    readOnly: true
```

**Ingest (NATS -> Backend)** — consumes from NATS and exports to observability backends via OTLP:

```yaml
image:
  repository: ghcr.io/mikluko/otelnats-collector
  tag: "0.3.1"

mode: deployment
replicaCount: 2

config:
  receivers:
    nats:
      url: nats://nats.nats-system:4222
      auth:
        credentials_file: /mnt/secrets/nats.creds
      traces:
        subject: "otel.traces.>"
        jetstream:
          stream: OTEL
          consumer: signal-traces
          ack_wait: 60s
          rate_limit: 1000  # messages/sec (optional)
          rate_burst: 100   # token bucket capacity
      metrics:
        subject: "otel.metrics.>"
        jetstream:
          stream: OTEL
          consumer: signal-metrics
          ack_wait: 60s
          rate_limit: 1000
          rate_burst: 100
      logs:
        subject: "otel.logs.>"
        jetstream:
          stream: OTEL
          consumer: signal-logs
          ack_wait: 60s
          rate_limit: 1000
          rate_burst: 100
  processors:
    batch:
      send_batch_size: 8192
      timeout: 1s
  exporters:
    otlphttp:
      endpoint: http://backend.observability:4318
  service:
    pipelines:
      traces:
        receivers: [nats]
        processors: [batch]
        exporters: [otlphttp]
      metrics:
        receivers: [nats]
        processors: [batch]
        exporters: [otlphttp]
      logs:
        receivers: [nats]
        processors: [batch]
        exporters: [otlphttp]

extraVolumes:
  - name: nats-creds
    secret:
      secretName: nats-creds
extraVolumeMounts:
  - name: nats-creds
    mountPath: /mnt/secrets/nats.creds
    subPath: nats.creds
    readOnly: true
```

The NATS receiver supports both Core NATS (with `queue_group` for load balancing) and JetStream (with `jetstream` block for at-least-once delivery). See [examples/helm/](./examples/helm/) for both variants.

**JetStream Rate Limiting**: Use `rate_limit` and `rate_burst` to throttle message consumption. This prevents CPU/memory spikes when catching up on backlogs after restarts. Rate limiting uses a token bucket algorithm — tokens are acquired *before* fetching messages to avoid wasting ACK timeout on buffered messages.

### DaemonSet Mode

Use `mode: daemonset` to collect telemetry directly from Kubernetes nodes — scraping Prometheus endpoints, tailing container logs — and forward everything to NATS.

```yaml
image:
  repository: ghcr.io/mikluko/otelnats-collector
  tag: "0.3.1"

mode: daemonset

extraEnvs:
  - name: NODE_NAME
    valueFrom:
      fieldRef:
        fieldPath: spec.nodeName

config:
  receivers:
    prometheus:
      config:
        scrape_configs:
          - job_name: kubernetes-pods
            scrape_interval: 30s
            kubernetes_sd_configs:
              - role: pod
            relabel_configs:
              - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
                action: keep
                regex: "true"
              - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_port]
                action: replace
                target_label: __address__
                regex: ([^:]+)(?::\d+)?;(\d+)
                replacement: $1:$2
                source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
    filelog:
      include:
        - /var/log/pods/*/*/*.log
      exclude:
        - /var/log/pods/*/otelnats-collector*/*.log
      start_at: end
      include_file_path: true
  exporters:
    nats:
      url: nats://nats.nats-system:4222
      auth:
        credentials_file: /mnt/secrets/nats.creds
      metrics:
        subject: otel.metrics.my-cluster
      logs:
        subject: otel.logs.my-cluster
  service:
    pipelines:
      metrics:
        receivers: [prometheus]
        processors: [memory_limiter, k8sattributes, batch]
        exporters: [nats]
      logs:
        receivers: [filelog]
        processors: [memory_limiter, k8sattributes, batch]
        exporters: [nats]

extraVolumes:
  - name: varlogpods
    hostPath:
      path: /var/log/pods
  - name: nats-creds
    secret:
      secretName: nats-creds
extraVolumeMounts:
  - name: varlogpods
    mountPath: /var/log/pods
    readOnly: true
  - name: nats-creds
    mountPath: /mnt/secrets/nats.creds
    subPath: nats.creds
    readOnly: true

tolerations:
  - operator: Exists

securityContext:
  runAsUser: 0
  runAsGroup: 0

clusterRole:
  create: true
  rules:
    - apiGroups: [""]
      resources: ["pods", "namespaces", "nodes"]
      verbs: ["get", "watch", "list"]
    - apiGroups: ["apps"]
      resources: ["replicasets", "deployments", "daemonsets", "statefulsets"]
      verbs: ["get", "watch", "list"]

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

DaemonSet mode requires `runAsUser: 0` to read host log files and RBAC rules for the `k8sattributes` processor and Prometheus service discovery.

### GitOps / Flux

For Flux CD deployments, define a `HelmRepository` and `HelmRelease`:

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: opentelemetry
spec:
  interval: 24h
  url: https://open-telemetry.github.io/opentelemetry-helm-charts
---
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: otelnats
spec:
  interval: 1h
  chart:
    spec:
      chart: opentelemetry-collector
      version: "0.x"
      sourceRef:
        kind: HelmRepository
        name: opentelemetry
  values:
    image:
      repository: ghcr.io/mikluko/otelnats-collector
      tag: "0.3.1"
    mode: deployment
    # ... collector config
```

Use Kustomize overlays to layer cluster-specific values (NATS subjects, credentials, resource limits) on top of a shared base.

## Configuration Examples

See [examples/](./examples/) directory:

| File | Description |
|------|-------------|
| `examples/helm/gateway-values.yaml` | OTLP -> NATS gateway (Deployment) |
| `examples/helm/ingest-values.yaml` | NATS -> backend with Core NATS queue groups (Deployment) |
| `examples/helm/ingest-jetstream-values.yaml` | NATS -> backend with JetStream (Deployment) |
| `examples/helm/daemonset-values.yaml` | Node scraping -> NATS (DaemonSet) |
| `examples/gateway/config.yaml` | Standalone gateway config |
| `examples/ingest/config.yaml` | Standalone ingest config |
| `examples/daemonset/config.yaml` | Standalone daemonset config |

## Development

```bash
make build    # build binary
make test     # run tests
make lint     # run linter
```

## License

Apache 2.0
