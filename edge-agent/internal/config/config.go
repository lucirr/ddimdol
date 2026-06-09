package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all runtime configuration for the edge agent.
type Config struct {
	EdgeID            string
	EdgeName          string
	EdgeRegion        string
	NatsURL           string
	CentralAPIURL     string
	HarborURL         string
	HeartbeatInterval time.Duration
	TLSEnabled        bool   // AGENT_TLS_ENABLED (default: false)
	TLSCAPath         string // AGENT_TLS_CA
	TLSCertPath       string // AGENT_TLS_CERT
	TLSKeyPath        string // AGENT_TLS_KEY
}

// Load reads configuration from environment variables.
// EDGE_ID and EDGE_NAME are required; all others have defaults.
func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("")
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("NATS_URL", "nats://localhost:4222")
	v.SetDefault("CENTRAL_API_URL", "http://localhost:8081")
	v.SetDefault("HARBOR_URL", "https://harbor.local")
	v.SetDefault("HEARTBEAT_INTERVAL", "10s")
	v.SetDefault("EDGE_REGION", "default")
	v.SetDefault("AGENT_TLS_ENABLED", false)

	edgeID := v.GetString("EDGE_ID")
	if edgeID == "" {
		return nil, fmt.Errorf("EDGE_ID environment variable is required")
	}

	edgeName := v.GetString("EDGE_NAME")
	if edgeName == "" {
		return nil, fmt.Errorf("EDGE_NAME environment variable is required")
	}

	intervalStr := v.GetString("HEARTBEAT_INTERVAL")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid HEARTBEAT_INTERVAL %q: %w", intervalStr, err)
	}

	tlsEnabled := v.GetBool("AGENT_TLS_ENABLED")
	tlsCAPath := v.GetString("AGENT_TLS_CA")
	tlsCertPath := v.GetString("AGENT_TLS_CERT")
	tlsKeyPath := v.GetString("AGENT_TLS_KEY")

	if tlsEnabled && (tlsCAPath == "" || tlsCertPath == "" || tlsKeyPath == "") {
		return nil, fmt.Errorf("AGENT_TLS_CA, AGENT_TLS_CERT, and AGENT_TLS_KEY are required when AGENT_TLS_ENABLED is true")
	}

	return &Config{
		EdgeID:            edgeID,
		EdgeName:          edgeName,
		EdgeRegion:        v.GetString("EDGE_REGION"),
		NatsURL:           v.GetString("NATS_URL"),
		CentralAPIURL:     v.GetString("CENTRAL_API_URL"),
		HarborURL:         v.GetString("HARBOR_URL"),
		HeartbeatInterval: interval,
		TLSEnabled:        tlsEnabled,
		TLSCAPath:         tlsCAPath,
		TLSCertPath:       tlsCertPath,
		TLSKeyPath:        tlsKeyPath,
	}, nil
}
