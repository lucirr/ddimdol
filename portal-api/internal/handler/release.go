package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/didimdol/portal-api/internal/domain"
	"github.com/didimdol/portal-api/internal/repository"
)

type ReleaseHandler struct {
	repo   repository.ReleaseRepository
	logger *zap.Logger
}

func NewReleaseHandler(repo repository.ReleaseRepository, logger *zap.Logger) *ReleaseHandler {
	return &ReleaseHandler{repo: repo, logger: logger}
}

func (h *ReleaseHandler) CreateRelease(c *gin.Context) {
	var req struct {
		PackageName    string         `json:"package_name" binding:"required"`
		Version        string         `json:"version" binding:"required"`
		ArtifactDigest string         `json:"artifact_digest" binding:"required"`
		ImageRef       string         `json:"image_ref"`
		SbomURI        string         `json:"sbom_uri"`
		CveReport      map[string]any `json:"cve_report"`
		Signature      string         `json:"signature"`
		SignedBy       string         `json:"signed_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	release := &domain.Release{
		ID:             uuid.New(),
		PackageName:    req.PackageName,
		Version:        req.Version,
		ArtifactDigest: req.ArtifactDigest,
		ImageRef:       req.ImageRef,
		SbomURI:        req.SbomURI,
		CveReport:      req.CveReport,
		Signature:      req.Signature,
		SignedBy:       req.SignedBy,
		Status:         domain.ReleaseStatusDraft,
		CreatedAt:      time.Now(),
	}
	if release.CveReport == nil {
		release.CveReport = map[string]any{}
	}

	if err := h.repo.Save(c.Request.Context(), release); err != nil {
		h.logger.Error("create release", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": release})
}

func (h *ReleaseHandler) ListReleases(c *gin.Context) {
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
		h.logger.Error("list releases", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": result.Items,
		"meta": gin.H{"total": result.Total, "page": page, "limit": limit},
	})
}

func (h *ReleaseHandler) GetRelease(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	release, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
			return
		}
		h.logger.Error("get release", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": release})
}

func (h *ReleaseHandler) UpdateCveReport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Critical int            `json:"critical"`
		High     int            `json:"high"`
		Medium   int            `json:"medium"`
		Low      int            `json:"low"`
		SbomURI  string         `json:"sbom_uri"`
		Raw      map[string]any `json:"raw"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	release, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}

	release.CveReport = map[string]any{
		"critical":   req.Critical,
		"high":       req.High,
		"medium":     req.Medium,
		"low":        req.Low,
		"sbom_uri":   req.SbomURI,
		"scanned_at": time.Now().UTC().Format(time.RFC3339),
	}
	if req.SbomURI != "" {
		release.SbomURI = req.SbomURI
	}
	release.Status = domain.ReleaseStatusScanned

	if err := h.repo.Save(c.Request.Context(), release); err != nil {
		h.logger.Error("update cve report", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": release})
}

func (h *ReleaseHandler) PublishRelease(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	release, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "release not found"})
		return
	}

	if release.Status != domain.ReleaseStatusScanned && release.Status != domain.ReleaseStatusSigned {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release must be SCANNED or SIGNED to publish"})
		return
	}

	if critical, ok := release.CveReport["critical"]; ok {
		switch v := critical.(type) {
		case float64:
			if v > 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "critical CVEs found, cannot publish"})
				return
			}
		case int:
			if v > 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "critical CVEs found, cannot publish"})
				return
			}
		}
	}

	now := time.Now()
	release.Status = domain.ReleaseStatusPublished
	release.PublishedAt = &now

	if err := h.repo.Save(c.Request.Context(), release); err != nil {
		h.logger.Error("publish release", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": release})
}
