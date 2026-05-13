package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/didimdol/portal-api/internal/repository"
)

type AuditHandler struct {
	repo   repository.AuditRepository
	logger *zap.Logger
}

func NewAuditHandler(repo repository.AuditRepository, logger *zap.Logger) *AuditHandler {
	return &AuditHandler{repo: repo, logger: logger}
}

func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	logs, err := h.repo.FindAll(c.Request.Context(), repository.AuditFilter{Limit: 100})
	if err != nil {
		h.logger.Error("list audit logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs})
}

func (h *AuditHandler) ExportAuditLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": nil, "message": "TODO: export audit logs"})
}
