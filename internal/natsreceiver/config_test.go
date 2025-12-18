package natsreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalnats "github.com/mikluko/opentelemetry-collector-nats/internal/nats"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "valid config with all signals",
			cfg: &Config{
				ClientConfig: internalnats.ClientConfig{
					URL: "nats://localhost:4222",
				},
				Traces:  SignalConfig{Subject: "otel.traces", QueueGroup: "traces-group"},
				Metrics: SignalConfig{Subject: "otel.metrics", QueueGroup: "metrics-group"},
				Logs:    SignalConfig{Subject: "otel.logs", QueueGroup: "logs-group"},
			},
			wantErr: "",
		},
		{
			name: "valid config with traces only",
			cfg: &Config{
				ClientConfig: internalnats.ClientConfig{
					URL: "nats://localhost:4222",
				},
				Traces: SignalConfig{Subject: "otel.traces"},
			},
			wantErr: "",
		},
		{
			name: "valid config with wildcard subject",
			cfg: &Config{
				ClientConfig: internalnats.ClientConfig{
					URL: "nats://localhost:4222",
				},
				Traces: SignalConfig{Subject: "otel.traces.>"},
			},
			wantErr: "",
		},
		{
			name: "valid config with single-token wildcard",
			cfg: &Config{
				ClientConfig: internalnats.ClientConfig{
					URL: "nats://localhost:4222",
				},
				Traces: SignalConfig{Subject: "otel.*.traces"},
			},
			wantErr: "",
		},
		{
			name: "missing url",
			cfg: &Config{
				Traces: SignalConfig{Subject: "otel.traces"},
			},
			wantErr: "url is required",
		},
		{
			name: "no signals configured",
			cfg: &Config{
				ClientConfig: internalnats.ClientConfig{
					URL: "nats://localhost:4222",
				},
			},
			wantErr: "at least one signal subject must be configured",
		},
		{
			name: "invalid encoding",
			cfg: &Config{
				ClientConfig: internalnats.ClientConfig{
					URL: "nats://localhost:4222",
				},
				Traces: SignalConfig{
					Subject:  "otel.traces",
					Encoding: "json",
				},
			},
			wantErr: "only otlp_proto encoding is currently supported",
		},
		{
			name: "valid with explicit otlp_proto encoding",
			cfg: &Config{
				ClientConfig: internalnats.ClientConfig{
					URL: "nats://localhost:4222",
				},
				Traces: SignalConfig{
					Subject:  "otel.traces",
					Encoding: "otlp_proto",
				},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
