package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// DeploymentResult is the payload sent to the central API after a deployment attempt.
type DeploymentResult struct {
	ApprovalID   string `json:"approval_id"`
	EdgeID       string `json:"edge_id"`
	Phase        string `json:"phase"`                    // COMPLETED | FAILED | ROLLED_BACK
	ProgressPct  int    `json:"progress_pct"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	ReportedAt   string `json:"reported_at"`
}

// Reporter sends deployment results to the central API.
type Reporter struct {
	centralURL string
	edgeID     string
	client     *http.Client
	logger     *zap.Logger
}

// New creates a Reporter with a sensible HTTP timeout.
func New(centralURL, edgeID string, logger *zap.Logger) *Reporter {
	return &Reporter{
		centralURL: centralURL,
		edgeID:     edgeID,
		client:     &http.Client{Timeout: 15 * time.Second},
		logger:     logger,
	}
}

// Report POSTs result to /agent/v1/deployment-result.
func (r *Reporter) Report(ctx context.Context, result DeploymentResult) error {
	result.EdgeID = r.edgeID
	result.ReportedAt = time.Now().UTC().Format(time.RFC3339)

	body, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal deployment result: %w", err)
	}

	url := r.centralURL + "/agent/v1/deployment-result"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("POST deployment result: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status from central API: %s", resp.Status)
	}

	r.logger.Info("deployment result reported",
		zap.String("approval_id", result.ApprovalID),
		zap.String("phase", result.Phase),
		zap.Int("status", resp.StatusCode),
	)
	return nil
}
