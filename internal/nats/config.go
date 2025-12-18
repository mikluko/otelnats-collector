// Package nats provides shared NATS client configuration and connection utilities
// for the OpenTelemetry Collector NATS receiver and exporter components.
package nats

import (
	"time"

	"go.opentelemetry.io/collector/config/configtls"
)

// ClientConfig holds NATS client configuration shared between
// exporter and receiver components.
type ClientConfig struct {
	// URL is the NATS server URL (e.g., "nats://localhost:4222").
	// Multiple URLs can be comma-separated for cluster support.
	URL string `mapstructure:"url"`

	// TLS configuration for secure connections.
	TLS *configtls.ClientConfig `mapstructure:"tls,omitempty"`

	// Auth holds authentication configuration.
	Auth AuthConfig `mapstructure:"auth,omitempty"`

	// ConnectionTimeout for initial connection (default: 10s).
	ConnectionTimeout time.Duration `mapstructure:"connection_timeout"`

	// ReconnectWait is the wait time between reconnection attempts (default: 2s).
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"`

	// MaxReconnects is the maximum number of reconnection attempts.
	// -1 means unlimited (default).
	MaxReconnects int `mapstructure:"max_reconnects"`
}

// AuthConfig holds NATS authentication options.
// Only one authentication method should be configured.
type AuthConfig struct {
	// UserInfo authentication (username/password).
	UserInfo *UserInfoAuth `mapstructure:"user_info,omitempty"`

	// Token authentication.
	Token string `mapstructure:"token,omitempty"`

	// NKeyFile is the path to the NKey seed file.
	NKeyFile string `mapstructure:"nkey_file,omitempty"`

	// CredentialsFile is the path to the credentials file (JWT + NKey).
	CredentialsFile string `mapstructure:"credentials_file,omitempty"`
}

// UserInfoAuth holds username/password authentication.
type UserInfoAuth struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// NewDefaultClientConfig returns ClientConfig with sensible defaults.
func NewDefaultClientConfig() ClientConfig {
	return ClientConfig{
		URL:               "nats://localhost:4222",
		ConnectionTimeout: 10 * time.Second,
		ReconnectWait:     2 * time.Second,
		MaxReconnects:     -1, // unlimited
	}
}
