package tlsconfig_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/your-org/edge-agent/internal/tlsconfig"
)

const (
	certsDir = "../../../deploy/local/certs/"
	caPath   = certsDir + "ca.crt"
	certPath = certsDir + "client-edge-local-01.crt"
	keyPath  = certsDir + "client-edge-local-01.key"
)

func TestClientConfig_ValidCert(t *testing.T) {
	cfg, err := tlsconfig.ClientConfig(caPath, certPath, keyPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(cfg.Certificates))
	}
	if cfg.RootCAs == nil {
		t.Error("expected non-nil RootCAs pool")
	}
}

func TestClientConfig_MissingCA(t *testing.T) {
	_, err := tlsconfig.ClientConfig("nonexistent-ca.crt", certPath, keyPath)
	if err == nil {
		t.Fatal("expected error for missing CA, got nil")
	}
}

func TestClientConfig_InvalidKeypair(t *testing.T) {
	// Use CA cert as both cert and key — this is an invalid keypair
	_, err := tlsconfig.ClientConfig(caPath, caPath, caPath)
	if err == nil {
		t.Fatal("expected error for invalid keypair, got nil")
	}
}

func TestHTTPClient(t *testing.T) {
	cfg, err := tlsconfig.ClientConfig(caPath, certPath, keyPath)
	if err != nil {
		t.Fatalf("failed to build TLS config: %v", err)
	}

	client := tlsconfig.HTTPClient(cfg, 10*time.Second)
	if client == nil {
		t.Fatal("expected non-nil http.Client")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected Transport to be *http.Transport")
	}
	if transport.TLSClientConfig == nil {
		t.Error("expected TLSClientConfig to be set on Transport")
	}
}
