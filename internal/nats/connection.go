package nats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Connect establishes a NATS connection with the given configuration.
func Connect(ctx context.Context, cfg ClientConfig, logger *zap.Logger) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Name("otel-collector"),
		nats.Timeout(cfg.ConnectionTimeout),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				logger.Warn("NATS disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			logger.Info("NATS connection closed")
		}),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			logger.Error("NATS error", zap.Error(err))
		}),
	}

	// Add authentication options
	authOpts, err := authOptions(cfg.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to configure authentication: %w", err)
	}
	opts = append(opts, authOpts...)

	// Add TLS if configured
	if cfg.TLS != nil {
		tlsConfig, err := cfg.TLS.LoadTLSConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS config: %w", err)
		}
		opts = append(opts, nats.Secure(tlsConfig))
	}

	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	logger.Info("Connected to NATS",
		zap.String("url", conn.ConnectedUrl()),
		zap.String("server_id", conn.ConnectedServerId()),
	)

	return conn, nil
}

func authOptions(auth AuthConfig) ([]nats.Option, error) {
	var opts []nats.Option

	if auth.UserInfo != nil {
		opts = append(opts, nats.UserInfo(auth.UserInfo.Username, auth.UserInfo.Password))
	}
	if auth.Token != "" {
		opts = append(opts, nats.Token(auth.Token))
	}
	if auth.NKeyFile != "" {
		opt, err := nats.NkeyOptionFromSeed(auth.NKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load NKey from %s: %w", auth.NKeyFile, err)
		}
		opts = append(opts, opt)
	}
	if auth.CredentialsFile != "" {
		opts = append(opts, nats.UserCredentials(auth.CredentialsFile))
	}

	return opts, nil
}
