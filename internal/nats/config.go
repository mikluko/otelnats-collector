// Package nats provides shared NATS client configuration and connection utilities
// for the OpenTelemetry Collector NATS receiver and exporter components.
package nats

import (
	"errors"
	"net/url"
	"regexp"
	"time"

	"go.opentelemetry.io/collector/config/configopaque"
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
	Token configopaque.String `mapstructure:"token,omitempty"`

	// NKeyFile is the path to the NKey seed file.
	NKeyFile string `mapstructure:"nkey_file,omitempty"`

	// CredentialsFile is the path to the credentials file (JWT + NKey).
	CredentialsFile string `mapstructure:"credentials_file,omitempty"`
}

// UserInfoAuth holds username/password authentication.
type UserInfoAuth struct {
	Username string                `mapstructure:"username"`
	Password configopaque.String   `mapstructure:"password"`
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

// Validate checks that the AuthConfig is valid.
// Only one authentication method can be configured at a time.
func (c *AuthConfig) Validate() error {
	count := 0
	if c.UserInfo != nil {
		count++
	}
	if c.Token != "" {
		count++
	}
	if c.NKeyFile != "" {
		count++
	}
	if c.CredentialsFile != "" {
		count++
	}
	if count > 1 {
		return errors.New("only one authentication method can be configured")
	}
	return nil
}

// ValidateURL checks that the URL is a valid NATS URL.
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("url is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return errors.New("invalid url format")
	}
	if u.Scheme != "nats" && u.Scheme != "tls" && u.Scheme != "nats+tls" {
		return errors.New("url scheme must be nats, tls, or nats+tls")
	}
	if u.Host == "" {
		return errors.New("url must contain a host")
	}
	return nil
}

// subjectRegex validates NATS subject format.
// Allows alphanumeric, dots, dashes, underscores, and wildcards (* and >).
var subjectRegex = regexp.MustCompile(`^[a-zA-Z0-9._*>-]+$`)

// ValidateSubject checks that a subject string is valid for NATS.
// Allows wildcards (* and >) for subscription subjects.
func ValidateSubject(subject string) error {
	if subject == "" {
		return errors.New("subject cannot be empty")
	}
	if !subjectRegex.MatchString(subject) {
		return errors.New("subject contains invalid characters")
	}
	return nil
}

// ValidatePublishSubject checks that a subject is valid for publishing.
// Unlike ValidateSubject, this disallows wildcards since you cannot
// publish to wildcard subjects in NATS.
func ValidatePublishSubject(subject string) error {
	if err := ValidateSubject(subject); err != nil {
		return err
	}
	for _, c := range subject {
		if c == '*' || c == '>' {
			return errors.New("publish subject cannot contain wildcards")
		}
	}
	return nil
}
