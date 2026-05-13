package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/didimdol/portal-api/internal/domain"
	"github.com/didimdol/portal-api/internal/repository"
	"github.com/didimdol/portal-api/internal/service"
)

type ApprovalHandler struct {
	repo     repository.ApprovalRepository
	rel      repository.ReleaseRepository
	edgeRepo repository.EdgeRepository
	nats     *service.NatsService
	harbor   *service.HarborService
	logger   *zap.Logger
}

func NewApprovalHandler(
	repo repository.ApprovalRepository,
	rel repository.ReleaseRepository,
	edgeRepo repository.EdgeRepository,
	nats *service.NatsService,
	harbor *service.HarborService,
	logger *zap.Logger,
) *ApprovalHandler {
	return &ApprovalHandler{repo: repo, rel: rel, edgeRepo: edgeRepo, nats: nats, harbor: harbor, logger: logger}
}

func (h *ApprovalHandler) CreateApproval(c *gin.Context) {
	var req struct {
		ReleaseID      string `json:"release_id" binding:"required"`
		EdgeID         string `json:"edge_id" binding:"required"`
		IdempotencyKey string `json:"idempotency_key"`
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

	idempotencyKey := req.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = uuid.New().String()
	}

	existing, err := h.repo.FindByIdempotencyKey(c.Request.Context(), idempotencyKey)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		h.logger.Error("idempotency lookup", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}
	if existing != nil {
		c.JSON(http.StatusOK, gin.H{"data": existing})
		return
	}

	requestedBy := uuid.Nil
	if uidStr := c.GetString("user_id"); uidStr != "" {
		if parsed, perr := uuid.Parse(uidStr); perr == nil {
			requestedBy = parsed
		} else {
			h.logger.Warn("user_id is not a valid UUID, using nil", zap.String("user_id", uidStr))
		}
	}

	approval := &domain.ApprovalRequest{
		ID:             uuid.New(),
		ReleaseID:      releaseID,
		EdgeID:         edgeID,
		RequestedBy:    requestedBy,
		Status:         domain.ApprovalStatusPending,
		IdempotencyKey: idempotencyKey,
		Version:        0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.repo.Save(c.Request.Context(), approval); err != nil {
		h.logger.Error("create approval", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": approval})
}

func (h *ApprovalHandler) ListApprovals(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	if page < 1 {
		page = 1
	}

	result, err := h.repo.FindPaged(c.Request.Context(), page, limit)
	if err != nil {
		h.logger.Error("list approvals", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": result.Items,
		"meta": gin.H{"total": result.Total, "page": page, "limit": limit},
	})
}

func (h *ApprovalHandler) GetApproval(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	approval, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "approval not found"})
			return
		}
		h.logger.Error("get approval", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": approval})
}

func (h *ApprovalHandler) Approve(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	approval, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "approval not found"})
		return
	}
	if approval.Status != domain.ApprovalStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "approval is not in PENDING status"})
		return
	}

	if h.rel != nil {
		release, err := h.rel.FindByID(c.Request.Context(), approval.ReleaseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch release"})
			return
		}
		if release.Status == "DRAFT" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot approve: release has not been scanned (status: DRAFT)"})
			return
		}
		getCriticalCount := func(v any) float64 {
			switch n := v.(type) {
			case float64:
				return n
			case int:
				return float64(n)
			case int64:
				return float64(n)
			}
			return 0
		}
		if critical, ok := release.CveReport["critical"]; ok {
			if getCriticalCount(critical) > 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "cannot approve: release has CRITICAL CVEs"})
				return
			}
		}
	}

	approval.Status = domain.ApprovalStatusApproved
	approval.DecisionReason = req.Reason
	approval.UpdatedAt = time.Now()

	if err := h.repo.UpdateStatus(c.Request.Context(), id, domain.ApprovalStatusApproved, req.Reason, approval.Version); err != nil {
		if strings.Contains(err.Error(), "version conflict") {
			c.JSON(http.StatusConflict, gin.H{"error": "version conflict"})
			return
		}
		h.logger.Error("approve", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.nats != nil {
		release, _ := h.rel.FindByID(c.Request.Context(), approval.ReleaseID)
		imageRef := ""
		if release != nil {
			imageRef = release.ImageRef
		}
		_ = h.nats.PublishApprovalEvent(c.Request.Context(), service.ApprovalEvent{
			ApprovalID: id.String(),
			ReleaseID:  approval.ReleaseID.String(),
			EdgeID:     approval.EdgeID.String(),
			Status:     "APPROVED",
			Reason:     req.Reason,
			ImageRef:   imageRef,
		})
	}

	if h.harbor != nil {
		edge, _ := h.edgeRepo.FindByID(c.Request.Context(), approval.EdgeID)
		if edge != nil {
			go func() {
				ctx2 := context.Background()
				if _, err := h.harbor.TriggerReplication(ctx2, edge.Name); err != nil {
					h.logger.Warn("harbor replication trigger failed", zap.Error(err))
				}
			}()
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": approval})
}

func (h *ApprovalHandler) Reject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	approval, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "approval not found"})
		return
	}
	if approval.Status != domain.ApprovalStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "approval is not in PENDING status"})
		return
	}

	approval.DecisionReason = req.Reason
	if err := h.repo.UpdateStatus(c.Request.Context(), id, domain.ApprovalStatusRejected, req.Reason, approval.Version); err != nil {
		if strings.Contains(err.Error(), "version conflict") {
			c.JSON(http.StatusConflict, gin.H{"error": "version conflict"})
			return
		}
		h.logger.Error("reject", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	approval.Status = domain.ApprovalStatusRejected
	c.JSON(http.StatusOK, gin.H{"data": approval})
}

func (h *ApprovalHandler) Defer(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": nil, "message": "TODO: defer"})
}

func (h *ApprovalHandler) ListEvents(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []any{}})
}
