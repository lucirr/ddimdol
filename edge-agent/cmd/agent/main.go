package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/your-org/edge-agent/internal/config"
	"github.com/your-org/edge-agent/internal/heartbeat"
	"github.com/your-org/edge-agent/internal/reporter"
	"github.com/your-org/edge-agent/internal/updater"
	"go.uber.org/zap"
)

func main() {
	// -------------------------------------------------------------------------
	// 1. Logger
	// -------------------------------------------------------------------------
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialise logger: " + err.Error())
	}
	defer logger.Sync() //nolint:errcheck

	// -------------------------------------------------------------------------
	// 2. Configuration
	// -------------------------------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("configuration error", zap.Error(err))
	}

	logger.Info("edge agent starting",
		zap.String("edge_id", cfg.EdgeID),
		zap.String("edge_name", cfg.EdgeName),
		zap.String("region", cfg.EdgeRegion),
		zap.String("nats_url", cfg.NatsURL),
	)

	// -------------------------------------------------------------------------
	// 3. NATS connection (outbound to central cluster)
	// -------------------------------------------------------------------------
	nc, err := nats.Connect(
		cfg.NatsURL,
		nats.Name("edge-agent:"+cfg.EdgeID),
		nats.MaxReconnects(-1),              // reconnect forever
		nats.ReconnectWait(5*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			logger.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
	)
	if err != nil {
		logger.Fatal("NATS connect failed", zap.Error(err))
	}
	defer nc.Drain() //nolint:errcheck

	js, err := jetstream.New(nc)
	if err != nil {
		logger.Fatal("JetStream init failed", zap.Error(err))
	}

	// -------------------------------------------------------------------------
	// 4. Sub-systems
	// -------------------------------------------------------------------------
	rep := reporter.New(cfg.CentralAPIURL, cfg.EdgeID, logger)

	hbSender, err := heartbeat.New(
		cfg.EdgeID, cfg.EdgeName, cfg.EdgeRegion,
		nc, js,
		cfg.HeartbeatInterval,
		logger,
	)
	if err != nil {
		logger.Fatal("heartbeat sender init failed", zap.Error(err))
	}

	upd := updater.New(
		cfg.EdgeID,
		js,
		cfg.CentralAPIURL,
		cfg.HarborURL,
		rep,
		logger,
	)

	// -------------------------------------------------------------------------
	// 5. Start goroutines with a shared cancellable context
	// -------------------------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hbSender.Start(ctx)

	go func() {
		if err := upd.Start(ctx); err != nil {
			logger.Error("updater exited with error", zap.Error(err))
		}
	}()

	// -------------------------------------------------------------------------
	// 6. Wait for SIGTERM / SIGINT then graceful shutdown
	// -------------------------------------------------------------------------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	sig := <-quit
	logger.Info("shutdown signal received", zap.String("signal", sig.String()))

	cancel() // stop heartbeat and updater goroutines

	// Give goroutines a moment to finish in-flight work.
	shutdownDeadline := time.After(10 * time.Second)
	<-shutdownDeadline

	logger.Info("edge agent stopped")
}
