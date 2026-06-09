package tlsconfig

import (
	"crypto/tls"
	"testing"
)

const (
	testCAPath   = "../../../deploy/local/certs/ca.crt"
	testCertPath = "../../../deploy/local/certs/server.crt"
	testKeyPath  = "../../../deploy/local/certs/server.key"
)

func TestServerConfig_ValidCerts(t *testing.T) {
	cfg, err := ServerConfig(testCAPath, testCertPath, testKeyPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("expected ClientAuth = RequireAndVerifyClientCert, got: %v", cfg.ClientAuth)
	}
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected MinVersion = TLS 1.3, got: %v", cfg.MinVersion)
	}
	if cfg.ClientCAs == nil {
		t.Error("expected non-nil ClientCAs pool")
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got: %d", len(cfg.Certificates))
	}
}

func TestServerConfig_MissingCA(t *testing.T) {
	_, err := ServerConfig("/nonexistent/ca.crt", testCertPath, testKeyPath)
	if err == nil {
		t.Fatal("expected error for missing CA path, got nil")
	}
}

func TestServerConfig_InvalidKeypair(t *testing.T) {
	// Pass CA cert as both cert and key — they won't form a valid keypair.
	_, err := ServerConfig(testCAPath, testCAPath, testCAPath)
	if err == nil {
		t.Fatal("expected error for invalid cert/key pair, got nil")
	}
}
