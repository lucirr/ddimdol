package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

// HarborService triggers image replication to edge Harbor mirrors.
type HarborService struct {
	centralURL string // e.g. http://harbor.central.internal
	username   string
	password   string
	client     *http.Client
	logger     *zap.Logger
}

func NewHarborService(centralURL, username, password string, logger *zap.Logger) *HarborService {
	return &HarborService{
		centralURL: centralURL,
		username:   username,
		password:   password,
		client:     &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
	}
}

type ReplicationTriggerResult struct {
	PolicyID   int    `json:"policy_id"`
	PolicyName string `json:"policy_name"`
	Triggered  bool   `json:"triggered"`
	Error      string `json:"error,omitempty"`
}

// TriggerReplication triggers Harbor replication policies that match the edge.
// edgeName: e.g. "edge-seoul-01" → triggers policies named "replicate-to-edge-seoul-01"
func (s *HarborService) TriggerReplication(ctx context.Context, edgeName string) (*ReplicationTriggerResult, error) {
	// List replication policies
	listURL := fmt.Sprintf("%s/api/v2.0/replication/policies?name=replicate-to-%s", s.centralURL, url.QueryEscape(edgeName))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build list policies request: %w", err)
	}
	req.SetBasicAuth(s.username, s.password)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list replication policies: %w", err)
	}
	defer resp.Body.Close()

	var policies []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&policies); err != nil {
		return nil, fmt.Errorf("decode policies: %w", err)
	}

	if len(policies) == 0 {
		s.logger.Warn("no replication policy found for edge", zap.String("edge", edgeName))
		return &ReplicationTriggerResult{Triggered: false}, nil
	}

	policy := policies[0]

	// Trigger execution
	execURL := fmt.Sprintf("%s/api/v2.0/replication/executions", s.centralURL)
	body, _ := json.Marshal(map[string]any{"policy_id": policy.ID})
	triggerReq, err := http.NewRequestWithContext(ctx, http.MethodPost, execURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build trigger request: %w", err)
	}
	triggerReq.SetBasicAuth(s.username, s.password)
	triggerReq.Header.Set("Content-Type", "application/json")

	triggerResp, err := s.client.Do(triggerReq)
	if err != nil {
		return nil, fmt.Errorf("trigger replication: %w", err)
	}
	defer triggerResp.Body.Close()

	if triggerResp.StatusCode >= 300 {
		return nil, fmt.Errorf("harbor replication trigger returned %s", triggerResp.Status)
	}

	s.logger.Info("harbor replication triggered",
		zap.String("edge", edgeName),
		zap.String("policy", policy.Name),
	)

	return &ReplicationTriggerResult{
		PolicyID:   policy.ID,
		PolicyName: policy.Name,
		Triggered:  true,
	}, nil
}
