package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/didimdol/portal-api/internal/domain"
	"github.com/didimdol/portal-api/internal/repository"
	"github.com/didimdol/portal-api/internal/service"
)

type ReleaseHandler struct {
	repo   repository.ReleaseRepository
	nats   *service.NatsService
	logger *zap.Logger
}

func NewReleaseHandler(repo repository.ReleaseRepository, nats *service.NatsService, logger *zap.Logger) *ReleaseHandler {
	return &ReleaseHandler{repo: repo, nats: nats, logger: logger}
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

	if req.ImageRef != "" {
		if err := validateImageRef(req.ImageRef); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image_ref format"})
			return
		}
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

// imageRefPattern matches valid Harbor/OCI image references and rejects leading
// hyphens that could be interpreted as flags by cosign/trivy/syft.
// Format: host[:port]/path/name(:tag|@sha256:digest)
var imageRefPattern = regexp.MustCompile(
	`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?(:[0-9]+)?(/[a-zA-Z0-9._-]+)+(@sha256:[a-f0-9]{64}|:[a-zA-Z0-9._-]+)$`,
)

// validateImageRef returns an error if imageRef does not match the expected
// registry-reference shape or could be misinterpreted as a CLI flag.
func validateImageRef(imageRef string) error {
	if !imageRefPattern.MatchString(imageRef) {
		return fmt.Errorf("invalid image reference format")
	}
	return nil
}

func (h *ReleaseHandler) SignRelease(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Signature is the cosign digest reference set by Actions.
	// ImageRef must match the value stored in the DB — Actions echoes back
	// what portal-api returned so the server can confirm the two inputs are
	// for the same resource and no cross-approval occurred.
	var req struct {
		Signature string `json:"signature" binding:"required"`
		ImageRef  string `json:"image_ref" binding:"required"`
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

	if release.Status != domain.ReleaseStatusScanned {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release must be SCANNED before signing"})
		return
	}

	// Confirm the image_ref from Actions matches what is stored for this release.
	// Prevents cross-approval: signing a different (valid) image to flip this release's status.
	if req.ImageRef != release.ImageRef {
		h.logger.Warn("image_ref mismatch in sign request",
			zap.String("request", req.ImageRef),
			zap.String("stored", release.ImageRef))
		c.JSON(http.StatusBadRequest, gin.H{"error": "image_ref does not match release record"})
		return
	}

	// Derive signed_by from the authenticated identity, not the request body.
	callerID, _ := c.Get("user_id")
	signedBy, _ := callerID.(string)
	if signedBy == "" {
		signedBy = "unknown"
	}

	release.Signature = req.Signature
	release.SignedBy = signedBy
	release.Status = domain.ReleaseStatusSigned

	if err := h.repo.Save(c.Request.Context(), release); err != nil {
		h.logger.Error("sign release", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": release})
}

// RequestPublish transitions a SCANNED/SIGNED release to PENDING_APPROVAL.
// Only pipeline-bot or central-operator may call this; a separate ApprovePublish
// step (central-operator only) is required before the release reaches PUBLISHED.
func (h *ReleaseHandler) RequestPublish(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "release must be SCANNED or SIGNED to request publish"})
		return
	}

	if critical, ok := release.CveReport["critical"]; ok {
		switch v := critical.(type) {
		case float64:
			if v > 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "critical CVEs found, cannot request publish"})
				return
			}
		case int:
			if v > 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "critical CVEs found, cannot request publish"})
				return
			}
		}
	}

	release.Status = domain.ReleaseStatusPendingApproval
	if err := h.repo.Save(c.Request.Context(), release); err != nil {
		h.logger.Error("request publish", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": release})
}

// ApprovePublish transitions a PENDING_APPROVAL release to PUBLISHED and fires
// the NATS release notification. Only central-operator may call this.
func (h *ReleaseHandler) ApprovePublish(c *gin.Context) {
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

	if release.Status != domain.ReleaseStatusPendingApproval {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release must be PENDING_APPROVAL to approve publish"})
		return
	}

	now := time.Now()
	release.Status = domain.ReleaseStatusPublished
	release.PublishedAt = &now

	if err := h.repo.Save(c.Request.Context(), release); err != nil {
		h.logger.Error("approve publish", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.nats != nil {
		_ = h.nats.PublishReleaseNotification(c.Request.Context(), service.ReleasePublishedEvent{
			ReleaseID:   release.ID.String(),
			PackageName: release.PackageName,
			Version:     release.Version,
			ImageRef:    release.ImageRef,
			PublishedAt: now,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": release})
}
