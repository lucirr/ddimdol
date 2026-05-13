package reporter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestReport_Success(t *testing.T) {
	var received DeploymentResult

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		if r.URL.Path != "/agent/v1/deployment-result" {
			t.Errorf("path: got %s, want /agent/v1/deployment-result", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rep := New(srv.URL, "edge-42", zap.NewNop())

	result := DeploymentResult{
		ApprovalID: "appr-001",
		Phase:      "COMPLETED",
	}

	if err := rep.Report(context.Background(), result); err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	if received.ApprovalID != "appr-001" {
		t.Errorf("ApprovalID: got %q, want %q", received.ApprovalID, "appr-001")
	}
	if received.EdgeID != "edge-42" {
		t.Errorf("EdgeID: got %q, want %q", received.EdgeID, "edge-42")
	}
	if received.Phase != "COMPLETED" {
		t.Errorf("Phase: got %q, want %q", received.Phase, "COMPLETED")
	}
	if received.ReportedAt == "" {
		t.Error("ReportedAt should not be empty")
	}
}

func TestReport_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	rep := New(srv.URL, "edge-42", zap.NewNop())

	err := rep.Report(context.Background(), DeploymentResult{
		ApprovalID: "appr-002",
		Phase:      "FAILED",
	})

	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestReport_InvalidURL(t *testing.T) {
	rep := New("http://127.0.0.1:0", "edge-42", zap.NewNop())

	err := rep.Report(context.Background(), DeploymentResult{
		ApprovalID: "appr-003",
		Phase:      "FAILED",
	})

	if err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}
}
