package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/your-org/edge-agent/internal/collector"
	"go.uber.org/zap"
)

// payload is the body sent to POST /agent/v1/heartbeat.
type payload struct {
	EdgeName  string             `json:"edge_name"`
	Region    string             `json:"region"`
	Metrics   *collector.Metrics `json:"metrics,omitempty"`
	Timestamp time.Time          `json:"timestamp"`
}

// Sender posts heartbeat messages to the central Agent API at a fixed interval.
type Sender struct {
	edgeName   string
	region     string
	centralURL string
	client     *http.Client
	logger     *zap.Logger
	interval   time.Duration
}

func New(
	edgeName, region string,
	centralURL string,
	client *http.Client,
	interval time.Duration,
	logger *zap.Logger,
) *Sender {
	return &Sender{
		edgeName:   edgeName,
		region:     region,
		centralURL: centralURL,
		client:     client,
		logger:     logger,
		interval:   interval,
	}
}

// Start blocks until ctx is cancelled, posting heartbeats on each tick.
func (s *Sender) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.logger.Info("heartbeat sender started", zap.Duration("interval", s.interval))

	for {
		select {
		case <-ticker.C:
			s.send(ctx)
		case <-ctx.Done():
			s.logger.Info("heartbeat sender stopped")
			return
		}
	}
}

func (s *Sender) send(ctx context.Context) {
	metrics, err := collector.Collect()
	if err != nil {
		s.logger.Warn("metrics collection failed", zap.Error(err))
		metrics = nil
	}

	body, err := json.Marshal(payload{
		EdgeName:  s.edgeName,
		Region:    s.region,
		Metrics:   metrics,
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		s.logger.Error("marshal heartbeat payload", zap.Error(err))
		return
	}

	url := s.centralURL + "/agent/v1/heartbeat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		s.logger.Error("build heartbeat request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn("heartbeat POST failed", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.logger.Warn("heartbeat unexpected status", zap.String("status", fmt.Sprintf("%d", resp.StatusCode)))
		return
	}

	s.logger.Debug("heartbeat sent")
}
