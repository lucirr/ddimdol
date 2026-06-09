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
)

type AgentHandler struct {
	edgeRepo       repository.EdgeRepository
	approvalRepo   repository.ApprovalRepository
	deploymentRepo repository.DeploymentRepository
	logger         *zap.Logger
}

func NewAgentHandler(
	edgeRepo repository.EdgeRepository,
	approvalRepo repository.ApprovalRepository,
	deploymentRepo repository.DeploymentRepository,
	logger *zap.Logger,
) *AgentHandler {
	return &AgentHandler{
		edgeRepo:       edgeRepo,
		approvalRepo:   approvalRepo,
		deploymentRepo: deploymentRepo,
		logger:         logger,
	}
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

	if err := h.approvalRepo.Save(c.Request.Context(), approval); err != nil {
		h.logger.Error("save approval request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"approval_id": approval.ID}})
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
