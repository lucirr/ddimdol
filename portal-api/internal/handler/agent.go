package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/didimdol/portal-api/internal/domain"
	"github.com/didimdol/portal-api/internal/repository"
	"github.com/didimdol/portal-api/internal/service"
)

type AgentHandler struct {
	edgeRepo       repository.EdgeRepository
	approvalRepo   repository.ApprovalRepository
	deploymentRepo repository.DeploymentRepository
	releaseRepo    repository.ReleaseRepository
	nats           *service.NatsService
	logger         *zap.Logger
}

func NewAgentHandler(
	edgeRepo repository.EdgeRepository,
	approvalRepo repository.ApprovalRepository,
	deploymentRepo repository.DeploymentRepository,
	releaseRepo repository.ReleaseRepository,
	nats *service.NatsService,
	logger *zap.Logger,
) *AgentHandler {
	return &AgentHandler{
		edgeRepo:       edgeRepo,
		approvalRepo:   approvalRepo,
		deploymentRepo: deploymentRepo,
		releaseRepo:    releaseRepo,
		nats:           nats,
		logger:         logger,
	}
}

func (h *AgentHandler) Heartbeat(c *gin.Context) {
	var req struct {
		EdgeID       string  `json:"edge_id" binding:"required"`
		CPUPct       float64 `json:"cpu_pct"`
		MemPct       float64 `json:"mem_pct"`
		DiskPct      float64 `json:"disk_pct"`
		AgentVersion string  `json:"agent_version"`
		K8sVersion   string  `json:"k8s_version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	edgeID, err := uuid.Parse(req.EdgeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid edge_id"})
		return
	}

	edge, err := h.edgeRepo.FindByID(c.Request.Context(), edgeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "edge not found"})
		return
	}

	now := time.Now()
	edge.Status = domain.EdgeStatusUp
	edge.LastHeartbeatAt = &now
	if req.AgentVersion != "" {
		edge.AgentVersion = req.AgentVersion
	}
	if req.K8sVersion != "" {
		edge.K8sVersion = req.K8sVersion
	}
	edge.UpdatedAt = now

	if err := h.edgeRepo.Save(c.Request.Context(), edge); err != nil {
		h.logger.Error("save heartbeat", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.nats != nil {
		_ = h.nats.PublishHeartbeatEvent(c.Request.Context(), service.HeartbeatEvent{
			EdgeID:    req.EdgeID,
			Timestamp: now,
			CPUPct:    req.CPUPct,
			MemPct:    req.MemPct,
			DiskPct:   req.DiskPct,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "ok"}})
}

func (h *AgentHandler) CreateApprovalRequest(c *gin.Context) {
	var req struct {
		ReleaseID string `json:"release_id" binding:"required"`
		EdgeID    string `json:"edge_id" binding:"required"`
		ImageRef  string `json:"image_ref" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	releaseID, err := uuid.Parse(req.ReleaseID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid release_id"})
		return
	}

	edgeID, err := uuid.Parse(req.EdgeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid edge_id"})
		return
	}

	idempotencyKey := fmt.Sprintf("agent-%s-%s", req.ReleaseID, req.EdgeID)

	existing, err := h.approvalRepo.FindByIdempotencyKey(c.Request.Context(), idempotencyKey)
	if err == nil && existing != nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"approval_id": existing.ID}})
		return
	}

	now := time.Now()
	approval := &domain.ApprovalRequest{
		ID:             uuid.New(),
		ReleaseID:      releaseID,
		EdgeID:         edgeID,
		RequestedBy:    uuid.Nil,
		Status:         domain.ApprovalStatusPending,
		IdempotencyKey: idempotencyKey,
		Version:        0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// 긴급 패치 자동 승인: 아래 조건을 모두 만족해야 함
	// 1) release.is_urgent == true
	// 2) release가 PUBLISHED 상태 (DRAFT/SCANNED 릴리스 자동 배포 방지)
	// 3) edgeRepo.FindByID 성공 → DB에 등록된 에지임을 검증
	// NOTE: release는 중앙 공용 아티팩트로 TenantID를 갖지 않음.
	// 크로스-테넌트 남용 방지는 edge_id를 mTLS 클라이언트 인증서 CN에
	// 바인딩하는 방식으로 강화 예정 (TODO: tlsconfig에서 CN 추출 후 주입).
	release, releaseErr := h.releaseRepo.FindByID(c.Request.Context(), releaseID)
	edge, edgeErr := h.edgeRepo.FindByID(c.Request.Context(), edgeID)

	var autoApproveImageRef string
	if releaseErr == nil && edgeErr == nil &&
		release.IsUrgent &&
		release.Status == domain.ReleaseStatusPublished &&
		edge.ID == edgeID { // DB 조회 성공으로 등록된 에지임을 보장
		approval.Status = domain.ApprovalStatusApproved
		autoApproveImageRef = release.ImageRef
		h.logger.Info("urgent patch auto-approved",
			zap.String("release_id", releaseID.String()),
			zap.String("edge_id", edgeID.String()),
			zap.String("tenant_id", edge.TenantID.String()),
		)
	}

	if err := h.approvalRepo.Save(c.Request.Context(), approval); err != nil {
		h.logger.Error("save approval request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 긴급 자동 승인 시 NATS 이벤트 즉시 발행 → edge-agent가 바로 배포 시작
	if approval.Status == domain.ApprovalStatusApproved && h.nats != nil {
		if err := h.nats.PublishApprovalEvent(c.Request.Context(), service.ApprovalEvent{
			ApprovalID: approval.ID.String(),
			ReleaseID:  releaseID.String(),
			EdgeID:     edgeID.String(),
			Status:     "APPROVED",
			Reason:     "urgent patch: auto-approved",
			ImageRef:   autoApproveImageRef,
		}); err != nil {
			h.logger.Warn("failed to publish urgent approval event", zap.Error(err))
		}
	}

	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"approval_id": approval.ID, "status": approval.Status}})
}

func (h *AgentHandler) DownloadProgress(c *gin.Context) {
	var req struct {
		ApprovalID  string `json:"approval_id" binding:"required"`
		ProgressPct int    `json:"progress_pct"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	approvalID, err := uuid.Parse(req.ApprovalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid approval_id"})
		return
	}

	approval, err := h.approvalRepo.FindByID(c.Request.Context(), approvalID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "approval not found"})
		return
	}

	record := &domain.DeploymentRecord{
		ID:          uuid.New(),
		ApprovalID:  approvalID,
		EdgeID:      approval.EdgeID,
		ReleaseID:   approval.ReleaseID,
		Phase:       domain.DeploymentPhaseDownloading,
		ProgressPct: int16(req.ProgressPct),
		StartedAt:   time.Now(),
	}

	if err := h.deploymentRepo.Save(c.Request.Context(), record); err != nil {
		h.logger.Error("save download progress", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"received": true}})
}

func (h *AgentHandler) DeploymentResult(c *gin.Context) {
	var req struct {
		ApprovalID   string `json:"approval_id" binding:"required"`
		Phase        string `json:"phase" binding:"required"`
		ProgressPct  int    `json:"progress_pct"`
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	approvalID, err := uuid.Parse(req.ApprovalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid approval_id"})
		return
	}

	approval, err := h.approvalRepo.FindByID(c.Request.Context(), approvalID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "approval not found"})
		return
	}

	phase := domain.DeploymentPhase(req.Phase)
	now := time.Now()

	record := &domain.DeploymentRecord{
		ID:           uuid.New(),
		ApprovalID:   approvalID,
		EdgeID:       approval.EdgeID,
		ReleaseID:    approval.ReleaseID,
		Phase:        phase,
		ProgressPct:  int16(req.ProgressPct),
		ErrorCode:    req.ErrorCode,
		ErrorMessage: req.ErrorMessage,
		StartedAt:    now,
	}

	if phase == domain.DeploymentPhaseCompleted || phase == domain.DeploymentPhaseFailed || phase == domain.DeploymentPhaseRolledBack {
		record.FinishedAt = &now
	}

	if err := h.deploymentRepo.Save(c.Request.Context(), record); err != nil {
		h.logger.Error("save deployment result", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if phase == domain.DeploymentPhaseCompleted {
		if err := h.approvalRepo.UpdateStatus(c.Request.Context(), approvalID, domain.ApprovalStatusApplied, "deployed successfully", approval.Version); err != nil {
			h.logger.Error("failed to update approval status", zap.Error(err))
			// Don't return error to agent — log it and continue
		}
	} else if phase == domain.DeploymentPhaseFailed {
		if err := h.approvalRepo.UpdateStatus(c.Request.Context(), approvalID, domain.ApprovalStatusRejected, req.ErrorMessage, approval.Version); err != nil {
			h.logger.Error("failed to update approval status", zap.Error(err))
			// Don't return error to agent — log it and continue
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": record})
}
