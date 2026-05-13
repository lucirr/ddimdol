package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

// NatsService wraps a NATS JetStream connection for publishing and subscribing
// to domain events.
type NatsService struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	logger *zap.Logger
}

// NewNatsService connects to NATS, configures JetStream streams, and returns a
// ready-to-use NatsService.
func NewNatsService(url string, logger *zap.Logger) (*NatsService, error) {
	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats jetstream init: %w", err)
	}

	svc := &NatsService{nc: nc, js: js, logger: logger}
	if err := svc.ensureStreams(context.Background()); err != nil {
		nc.Close()
		return nil, err
	}
	return svc, nil
}

func (s *NatsService) ensureStreams(ctx context.Context) error {
	streams := []jetstream.StreamConfig{
		{
			Name:     "RELEASES",
			Subjects: []string{"releases.>"},
			MaxAge:   7 * 24 * time.Hour,
		},
		{
			Name:     "EDGE_EVENTS",
			Subjects: []string{"edge.>"},
			MaxAge:   24 * time.Hour,
		},
		{
			Name:     "APPROVALS",
			Subjects: []string{"approvals.>"},
			MaxAge:   30 * 24 * time.Hour,
		},
	}
	for _, cfg := range streams {
		if _, err := s.js.CreateOrUpdateStream(ctx, cfg); err != nil {
			return fmt.Errorf("nats ensure stream %s: %w", cfg.Name, err)
		}
	}
	return nil
}

// ReleasePublishedEvent is emitted when a release transitions to PUBLISHED.
type ReleasePublishedEvent struct {
	ReleaseID   string    `json:"release_id"`
	PackageName string    `json:"package_name"`
	Version     string    `json:"version"`
	PublishedAt time.Time `json:"published_at"`
}

// PublishReleaseNotification publishes a release-published event to JetStream.
func (s *NatsService) PublishReleaseNotification(ctx context.Context, evt ReleasePublishedEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal release event: %w", err)
	}
	subject := fmt.Sprintf("releases.published.%s", evt.ReleaseID)
	if _, err := s.js.Publish(ctx, subject, data); err != nil {
		s.logger.Error("publish release notification", zap.String("subject", subject), zap.Error(err))
		return fmt.Errorf("publish release notification: %w", err)
	}
	return nil
}

// ApprovalEvent is emitted when an approval changes state.
type ApprovalEvent struct {
	ApprovalID string `json:"approval_id"`
	ReleaseID  string `json:"release_id"`
	EdgeID     string `json:"edge_id"`
	Status     string `json:"status"`
	Reason     string `json:"reason"`
	ImageRef   string `json:"image_ref"`
}

// PublishApprovalEvent publishes an approval state-change event to JetStream.
func (s *NatsService) PublishApprovalEvent(ctx context.Context, evt ApprovalEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal approval event: %w", err)
	}
	subject := fmt.Sprintf("approvals.%s.%s", evt.Status, evt.ApprovalID)
	if _, err := s.js.Publish(ctx, subject, data); err != nil {
		s.logger.Error("publish approval event", zap.String("subject", subject), zap.Error(err))
		return fmt.Errorf("publish approval event: %w", err)
	}
	return nil
}

// HeartbeatEvent carries telemetry from an edge agent heartbeat.
type HeartbeatEvent struct {
	EdgeID    string    `json:"edge_id"`
	Timestamp time.Time `json:"timestamp"`
	CPUPct    float64   `json:"cpu_pct"`
	MemPct    float64   `json:"mem_pct"`
	DiskPct   float64   `json:"disk_pct"`
}

// PublishHeartbeatEvent publishes an edge heartbeat telemetry event to JetStream.
func (s *NatsService) PublishHeartbeatEvent(ctx context.Context, evt HeartbeatEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal heartbeat event: %w", err)
	}
	subject := fmt.Sprintf("edge.heartbeat.%s", evt.EdgeID)
	if _, err := s.js.Publish(ctx, subject, data); err != nil {
		return fmt.Errorf("publish heartbeat: %w", err)
	}
	return nil
}

// SubscribeHeartbeats starts a durable JetStream consumer for edge heartbeat
// events and calls handler for each message. Blocks until ctx is cancelled or
// the consumer fails to start.
func (s *NatsService) SubscribeHeartbeats(ctx context.Context, handler func(HeartbeatEvent)) error {
	cons, err := s.js.CreateOrUpdateConsumer(ctx, "EDGE_EVENTS", jetstream.ConsumerConfig{
		Durable:       "central-heartbeat-consumer",
		FilterSubject: "edge.heartbeat.>",
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("nats create heartbeat consumer: %w", err)
	}

	_, err = cons.Consume(func(msg jetstream.Msg) {
		var evt HeartbeatEvent
		if err := json.Unmarshal(msg.Data(), &evt); err != nil {
			s.logger.Error("unmarshal heartbeat event", zap.Error(err))
			if nakErr := msg.Nak(); nakErr != nil {
				s.logger.Error("nak heartbeat message", zap.Error(nakErr))
			}
			return
		}
		handler(evt)
		if ackErr := msg.Ack(); ackErr != nil {
			s.logger.Error("ack heartbeat message", zap.Error(ackErr))
		}
	})
	return err
}

// Close drains the NATS connection gracefully.
func (s *NatsService) Close() {
	if err := s.nc.Drain(); err != nil {
		s.logger.Error("nats drain", zap.Error(err))
	}
}
