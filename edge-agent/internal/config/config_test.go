package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		wantErr     bool
		wantCfg     *Config
	}{
		{
			name:    "missing EDGE_ID",
			env:     map[string]string{"EDGE_NAME": "node-1"},
			wantErr: true,
		},
		{
			name:    "missing EDGE_NAME",
			env:     map[string]string{"EDGE_ID": "abc-123"},
			wantErr: true,
		},
		{
			name: "invalid HEARTBEAT_INTERVAL",
			env: map[string]string{
				"EDGE_ID":            "abc-123",
				"EDGE_NAME":          "node-1",
				"HEARTBEAT_INTERVAL": "not-a-duration",
			},
			wantErr: true,
		},
		{
			name: "minimal valid config uses defaults",
			env: map[string]string{
				"EDGE_ID":   "abc-123",
				"EDGE_NAME": "node-1",
			},
			wantCfg: &Config{
				EdgeID:            "abc-123",
				EdgeName:          "node-1",
				EdgeRegion:        "default",
				NatsURL:           "nats://localhost:4222",
				CentralAPIURL:     "http://localhost:8081",
				HarborURL:         "https://harbor.local",
				HeartbeatInterval: 10 * time.Second,
			},
		},
		{
			name: "all values overridden",
			env: map[string]string{
				"EDGE_ID":            "edge-42",
				"EDGE_NAME":          "prod-node",
				"EDGE_REGION":        "eu-west",
				"NATS_URL":           "nats://central:4222",
				"CENTRAL_API_URL":    "https://api.central.local",
				"HARBOR_URL":         "https://harbor.prod.local",
				"HEARTBEAT_INTERVAL": "30s",
			},
			wantCfg: &Config{
				EdgeID:            "edge-42",
				EdgeName:          "prod-node",
				EdgeRegion:        "eu-west",
				NatsURL:           "nats://central:4222",
				CentralAPIURL:     "https://api.central.local",
				HarborURL:         "https://harbor.prod.local",
				HeartbeatInterval: 30 * time.Second,
			},
		},
		{
			name: "TLS enabled with all paths provided",
			env: map[string]string{
				"EDGE_ID":            "tls-edge",
				"EDGE_NAME":          "tls-node",
				"AGENT_TLS_ENABLED":  "true",
				"AGENT_TLS_CA":       "/etc/certs/ca.crt",
				"AGENT_TLS_CERT":     "/etc/certs/client.crt",
				"AGENT_TLS_KEY":      "/etc/certs/client.key",
			},
			wantCfg: &Config{
				EdgeID:            "tls-edge",
				EdgeName:          "tls-node",
				EdgeRegion:        "default",
				NatsURL:           "nats://localhost:4222",
				CentralAPIURL:     "http://localhost:8081",
				HarborURL:         "https://harbor.local",
				HeartbeatInterval: 10 * time.Second,
				TLSEnabled:        true,
				TLSCAPath:         "/etc/certs/ca.crt",
				TLSCertPath:       "/etc/certs/client.crt",
				TLSKeyPath:        "/etc/certs/client.key",
			},
		},
		{
			name: "TLS enabled with empty CA path",
			env: map[string]string{
				"EDGE_ID":           "tls-edge",
				"EDGE_NAME":         "tls-node",
				"AGENT_TLS_ENABLED": "true",
				"AGENT_TLS_CERT":    "/etc/certs/client.crt",
				"AGENT_TLS_KEY":     "/etc/certs/client.key",
			},
			wantErr: true,
		},
		{
			name: "TLS enabled with empty cert path",
			env: map[string]string{
				"EDGE_ID":           "tls-edge",
				"EDGE_NAME":         "tls-node",
				"AGENT_TLS_ENABLED": "true",
				"AGENT_TLS_CA":      "/etc/certs/ca.crt",
				"AGENT_TLS_KEY":     "/etc/certs/client.key",
			},
			wantErr: true,
		},
		{
			name: "TLS enabled with empty key path",
			env: map[string]string{
				"EDGE_ID":           "tls-edge",
				"EDGE_NAME":         "tls-node",
				"AGENT_TLS_ENABLED": "true",
				"AGENT_TLS_CA":      "/etc/certs/ca.crt",
				"AGENT_TLS_CERT":    "/etc/certs/client.crt",
			},
			wantErr: true,
		},
		{
			name: "TLS disabled uses default false",
			env: map[string]string{
				"EDGE_ID":   "edge-1",
				"EDGE_NAME": "node-1",
			},
			wantCfg: &Config{
				EdgeID:            "edge-1",
				EdgeName:          "node-1",
				EdgeRegion:        "default",
				NatsURL:           "nats://localhost:4222",
				CentralAPIURL:     "http://localhost:8081",
				HarborURL:         "https://harbor.local",
				HeartbeatInterval: 10 * time.Second,
				TLSEnabled:        false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set env vars for this test case.
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			cfg, err := Load()

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.EdgeID != tc.wantCfg.EdgeID {
				t.Errorf("EdgeID: got %q, want %q", cfg.EdgeID, tc.wantCfg.EdgeID)
			}
			if cfg.EdgeName != tc.wantCfg.EdgeName {
				t.Errorf("EdgeName: got %q, want %q", cfg.EdgeName, tc.wantCfg.EdgeName)
			}
			if cfg.EdgeRegion != tc.wantCfg.EdgeRegion {
				t.Errorf("EdgeRegion: got %q, want %q", cfg.EdgeRegion, tc.wantCfg.EdgeRegion)
			}
			if cfg.NatsURL != tc.wantCfg.NatsURL {
				t.Errorf("NatsURL: got %q, want %q", cfg.NatsURL, tc.wantCfg.NatsURL)
			}
			if cfg.CentralAPIURL != tc.wantCfg.CentralAPIURL {
				t.Errorf("CentralAPIURL: got %q, want %q", cfg.CentralAPIURL, tc.wantCfg.CentralAPIURL)
			}
			if cfg.HarborURL != tc.wantCfg.HarborURL {
				t.Errorf("HarborURL: got %q, want %q", cfg.HarborURL, tc.wantCfg.HarborURL)
			}
			if cfg.HeartbeatInterval != tc.wantCfg.HeartbeatInterval {
				t.Errorf("HeartbeatInterval: got %v, want %v", cfg.HeartbeatInterval, tc.wantCfg.HeartbeatInterval)
			}
			if cfg.TLSEnabled != tc.wantCfg.TLSEnabled {
				t.Errorf("TLSEnabled: got %v, want %v", cfg.TLSEnabled, tc.wantCfg.TLSEnabled)
			}
			if cfg.TLSCAPath != tc.wantCfg.TLSCAPath {
				t.Errorf("TLSCAPath: got %q, want %q", cfg.TLSCAPath, tc.wantCfg.TLSCAPath)
			}
			if cfg.TLSCertPath != tc.wantCfg.TLSCertPath {
				t.Errorf("TLSCertPath: got %q, want %q", cfg.TLSCertPath, tc.wantCfg.TLSCertPath)
			}
			if cfg.TLSKeyPath != tc.wantCfg.TLSKeyPath {
				t.Errorf("TLSKeyPath: got %q, want %q", cfg.TLSKeyPath, tc.wantCfg.TLSKeyPath)
			}
		})
	}
}
