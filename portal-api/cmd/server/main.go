package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/didimdol/portal-api/internal/config"
	"github.com/didimdol/portal-api/internal/handler"
	"github.com/didimdol/portal-api/internal/hub"
	"github.com/didimdol/portal-api/internal/middleware"
	"github.com/didimdol/portal-api/internal/repository/postgres"
	"github.com/didimdol/portal-api/internal/service"
	"github.com/didimdol/portal-api/internal/tlsconfig"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Errorf("failed to initialize logger: %w", err))
	}
	defer logger.Sync() //nolint:errcheck

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// DB connection
	db, err := postgres.New(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// NATS connection (optional — warn and continue if unavailable)
	var natsSvc *service.NatsService
	if cfg.NatsURL != "" {
		natsSvc, err = service.NewNatsService(cfg.NatsURL, logger)
		if err != nil {
			logger.Warn("failed to connect to NATS, event publishing disabled", zap.Error(err))
		} else {
			defer natsSvc.Close()
		}
	}

	// Harbor service (optional — only when HARBOR_URL is configured)
	var harborSvc *service.HarborService
	if cfg.HarborURL != "" {
		harborSvc = service.NewHarborService(cfg.HarborURL, cfg.HarborUser, cfg.HarborPassword, logger)
	}

	// Repositories
	edgeRepo := postgres.NewEdgeRepository(db)
	releaseRepo := postgres.NewReleaseRepository(db)
	approvalRepo := postgres.NewApprovalRepository(db)
	deploymentRepo := postgres.NewDeploymentRepository(db)
	auditRepo := postgres.NewAuditRepository(db)

	// WebSocket hub
	wsHub := hub.New(logger)
	go wsHub.Run()

	// Public API router (JWT-protected)
	apiRouter := gin.New()
	apiRouter.Use(gin.Recovery())
	apiRouter.Use(middleware.AuditLogger())

	apiRouter.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	edgeH := handler.NewEdgeHandler(edgeRepo, logger)
	releaseH := handler.NewReleaseHandler(releaseRepo, natsSvc, logger)
	approvalH := handler.NewApprovalHandler(approvalRepo, releaseRepo, edgeRepo, deploymentRepo, natsSvc, harborSvc, logger)
	sessionH := handler.NewSessionHandler()
	auditH := handler.NewAuditHandler(auditRepo, logger)
	wsH := handler.NewWsHandler(wsHub, logger)

	v1 := apiRouter.Group("/api/v1", middleware.Auth())
	{
		edges := v1.Group("/edges")
		{
			edges.POST("", edgeH.CreateEdge)
			edges.GET("", edgeH.ListEdges)
			edges.GET("/:id", edgeH.GetEdge)
			edges.GET("/:id/heartbeats", edgeH.ListHeartbeats)
			edges.POST("/:id/commands", edgeH.SendCommand)
			edges.GET("/:id/catalog", edgeH.GetCatalog)
		}

		releases := v1.Group("/releases")
		{
			// pipeline-bot: CI/CD 파이프라인 서비스 계정 (DRAFT 생성, 스캔 결과 기록, 서명)
			// central-operator: 관리자 (위 모든 권한 + publish)
			releases.POST("", middleware.RequireRole("central-operator", "pipeline-bot"), releaseH.CreateRelease)
			releases.GET("", releaseH.ListReleases)
			releases.GET("/:id", releaseH.GetRelease)
			releases.PATCH("/:id/cve-report", middleware.RequireRole("central-operator", "pipeline-bot"), releaseH.UpdateCveReport)
			releases.PATCH("/:id/sign", middleware.RequireRole("central-operator", "pipeline-bot"), releaseH.SignRelease)
			releases.POST("/:id/request-publish", middleware.RequireRole("central-operator", "pipeline-bot"), releaseH.RequestPublish)
			releases.POST("/:id/approve-publish", middleware.RequireRole("central-operator"), releaseH.ApprovePublish)
		}

		approvals := v1.Group("/approvals")
		{
			approvals.POST("", approvalH.CreateApproval)
			approvals.GET("", approvalH.ListApprovals)
			approvals.GET("/:id", approvalH.GetApproval)
			approvals.POST("/:id/approve", approvalH.Approve)
			approvals.POST("/:id/reject", approvalH.Reject)
			approvals.POST("/:id/defer", approvalH.Defer)
			approvals.GET("/:id/events", approvalH.ListEvents)
			approvals.GET("/:id/deployments", approvalH.ListDeployments)
		}

		sessions := v1.Group("/remote-sessions")
		{
			sessions.POST("", sessionH.CreateSession)
			sessions.GET("", sessionH.ListSessions)
			sessions.POST("/:id/activate", sessionH.ActivateSession)
			sessions.POST("/:id/terminate", sessionH.TerminateSession)
			sessions.GET("/:id/recording", sessionH.GetRecording)
		}

		audit := v1.Group("/audit-logs")
		{
			audit.GET("", auditH.ListAuditLogs)
			audit.GET("/export", auditH.ExportAuditLogs)
		}

		v1.GET("/ws/edges", wsH.HandleEdgeEvents)
	}

	// mTLS-only agent router (separate port)
	agentRouter := gin.New()
	agentRouter.Use(gin.Recovery())
	agentRouter.Use(middleware.AgentMTLSIdentity())

	agentH := handler.NewAgentHandler(edgeRepo, approvalRepo, deploymentRepo, releaseRepo, natsSvc, wsHub, logger)
	agentV1 := agentRouter.Group("/agent/v1")
	{
		agentV1.POST("/heartbeat", agentH.Heartbeat)
		agentV1.POST("/approval-requests", agentH.CreateApprovalRequest)
		agentV1.POST("/download-progress", agentH.DownloadProgress)
		agentV1.POST("/deployment-result", agentH.DeploymentResult)
	}

	apiAddr := fmt.Sprintf(":%d", cfg.ServerPort)
	agentAddr := fmt.Sprintf(":%d", cfg.AgentPort)

	logger.Info("starting portal-api server", zap.String("addr", apiAddr))

	errCh := make(chan error, 2)

	go func() {
		if err := http.ListenAndServe(apiAddr, apiRouter); err != nil {
			errCh <- fmt.Errorf("api server error: %w", err)
		}
	}()

	if cfg.AgentTLSEnabled {
		logger.Info("starting agent server (mTLS enabled)", zap.String("addr", agentAddr))
		tlsCfg, err := tlsconfig.ServerConfig(cfg.AgentTLSCAPath, cfg.AgentTLSCertPath, cfg.AgentTLSKeyPath)
		if err != nil {
			logger.Fatal("failed to build agent TLS config", zap.Error(err))
		}
		agentSrv := &http.Server{
			Addr:      agentAddr,
			Handler:   agentRouter,
			TLSConfig: tlsCfg,
		}
		go func() {
			if err := agentSrv.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("agent server error: %w", err)
			}
		}()
	} else {
		// TLS disabled: bind to loopback only so the agent port is never reachable
		// from outside the host. This mode is for local development only.
		loopbackAddr := fmt.Sprintf("127.0.0.1:%d", cfg.AgentPort)
		logger.Warn("agent server TLS DISABLED — bound to loopback only (127.0.0.1). DO NOT USE IN PRODUCTION.",
			zap.String("addr", loopbackAddr))
		go func() {
			if err := http.ListenAndServe(loopbackAddr, agentRouter); err != nil {
				errCh <- fmt.Errorf("agent server error: %w", err)
			}
		}()
	}

	if err := <-errCh; err != nil {
		logger.Fatal("server terminated", zap.Error(err))
	}
}
