package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/your-org/edge-agent/internal/collector"
	"go.uber.org/zap"
)

// HeartbeatPayload is the message published to NATS on every tick.
type HeartbeatPayload struct {
	EdgeID    string             `json:"edge_id"`
	EdgeName  string             `json:"edge_name"`
	Region    string             `json:"region"`
	Metrics   *collector.Metrics `json:"metrics"`
	Timestamp time.Time          `json:"timestamp"`
}

// Sender publishes heartbeat messages to NATS JetStream at a fixed interval.
type Sender struct {
	edgeID   string
	edgeName string
	region   string
	nc       *nats.Conn
	js       jetstream.JetStream
	logger   *zap.Logger
	interval time.Duration
}

// New creates a Sender. Heartbeats are published to the central EDGE_EVENTS
// stream (managed by portal-api) via subject "edge.heartbeat.<edgeID>".
// No stream creation is needed here — the stream already covers "edge.>" subjects.
func New(
	edgeID, edgeName, region string,
	nc *nats.Conn,
	js jetstream.JetStream,
	interval time.Duration,
	logger *zap.Logger,
) (*Sender, error) {
	return &Sender{
		edgeID:   edgeID,
		edgeName: edgeName,
		region:   region,
		nc:       nc,
		js:       js,
		logger:   logger,
		interval: interval,
	}, nil
}

// Start blocks until ctx is cancelled, publishing heartbeats on each tick.
func (s *Sender) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.logger.Info("heartbeat sender started", zap.Duration("interval", s.interval))

	for {
		select {
		case <-ticker.C:
			s.sendHeartbeat(ctx)
		case <-ctx.Done():
			s.logger.Info("heartbeat sender stopped")
			return
		}
	}
}

func (s *Sender) sendHeartbeat(ctx context.Context) {
	metrics, err := collector.Collect()
	if err != nil {
		s.logger.Warn("metrics collection failed", zap.Error(err))
		metrics = nil
	}

	payload := HeartbeatPayload{
		EdgeID:    s.edgeID,
		EdgeName:  s.edgeName,
		Region:    s.region,
		Metrics:   metrics,
		Timestamp: time.Now().UTC(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("marshal heartbeat payload", zap.Error(err))
		return
	}

	subject := fmt.Sprintf("edge.heartbeat.%s", s.edgeID)
	if _, err = s.js.Publish(ctx, subject, data); err != nil {
		s.logger.Error("publish heartbeat", zap.String("subject", subject), zap.Error(err))
		return
	}

	s.logger.Debug("heartbeat sent", zap.String("subject", subject))
}
