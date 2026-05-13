package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/didimdol/portal-api/internal/domain"
	"github.com/didimdol/portal-api/internal/repository"
)

type EdgeHandler struct {
	repo   repository.EdgeRepository
	logger *zap.Logger
}

func NewEdgeHandler(repo repository.EdgeRepository, logger *zap.Logger) *EdgeHandler {
	return &EdgeHandler{repo: repo, logger: logger}
}

func (h *EdgeHandler) ListEdges(c *gin.Context) {
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
		h.logger.Error("list edges", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": result.Items,
		"meta": gin.H{"total": result.Total, "page": page, "limit": limit},
	})
}

func (h *EdgeHandler) GetEdge(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	edge, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "edge not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": edge})
}

func (h *EdgeHandler) ListHeartbeats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []any{}})
}

func (h *EdgeHandler) SendCommand(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": nil})
}

func (h *EdgeHandler) GetCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []any{}})
}

func (h *EdgeHandler) CreateEdge(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Region   string `json:"region" binding:"required"`
		TenantID string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := uuid.New()
	if req.TenantID != "" {
		parsed, err := uuid.Parse(req.TenantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
			return
		}
		tenantID = parsed
	}

	now := time.Now()
	edge := &domain.EdgeNode{
		ID:           uuid.New(),
		Name:         req.Name,
		Region:       req.Region,
		TenantID:     tenantID,
		Status:       domain.EdgeStatusUnknown,
		AgentVersion: "",
		K8sVersion:   "",
		Capabilities: map[string]any{},
		Labels:       map[string]any{},
		PublicKey:    "",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.repo.Save(c.Request.Context(), edge); err != nil {
		h.logger.Error("create edge", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": edge})
}
