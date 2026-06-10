package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/your-org/edge-agent/internal/reporter"
	"go.uber.org/zap"
)

// ReleaseEvent is received from the NATS subject releases.published.>
type ReleaseEvent struct {
	ReleaseID  string `json:"release_id"`
	AppName    string `json:"app_name"`
	ImageRef   string `json:"image_ref"`   // e.g. harbor.local/myapp:v1.2.3
	TargetEdge string `json:"target_edge"` // empty means broadcast
}

// ApprovalEvent is received from the NATS subject approvals.APPROVED.*
type ApprovalEvent struct {
	ApprovalID string `json:"approval_id"`
	ReleaseID  string `json:"release_id"`
	ImageRef   string `json:"image_ref"`
	EdgeID     string `json:"edge_id"`
}

// approvalRequest is sent to the central API to request deployment approval.
type approvalRequest struct {
	ReleaseID string `json:"release_id"`
	EdgeID    string `json:"edge_id"`
	ImageRef  string `json:"image_ref"`
}

// approvalResponse is returned by the central API.
type approvalResponse struct {
	ApprovalID string `json:"approval_id"`
}

// Updater listens for release notifications, requests approval, then pulls
// and deploys images from the local Harbor mirror when approved.
type Updater struct {
	edgeID         string
	js             jetstream.JetStream
	centralURL     string
	harborURL      string
	reporter       *reporter.Reporter
	catalogWatcher *CatalogWatcher
	client         *http.Client
	logger         *zap.Logger
}

// New creates an Updater.
func New(
	edgeID string,
	js jetstream.JetStream,
	centralURL, harborURL string,
	rep *reporter.Reporter,
	logger *zap.Logger,
) *Updater {
	return &Updater{
		edgeID:         edgeID,
		js:             js,
		centralURL:     centralURL,
		harborURL:      harborURL,
		reporter:       rep,
		catalogWatcher: newCatalogWatcher(rep, "edgedip", logger),
		client:         &http.Client{Timeout: 30 * time.Second},
		logger:         logger,
	}
}

// Start subscribes to release and approval subjects and blocks until ctx is done.
func (u *Updater) Start(ctx context.Context) error {
	// Streams are managed by portal-api; edge-agent only creates consumers.

	// Durable consumer for release notifications (RELEASES stream: releases.>).
	relCons, err := u.js.CreateOrUpdateConsumer(ctx, "RELEASES", jetstream.ConsumerConfig{
		Durable:       fmt.Sprintf("edge-%s-releases", u.edgeID),
		FilterSubject: "releases.published.>",
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("create release consumer: %w", err)
	}

	// Durable consumer for approval events (APPROVALS stream: approvals.>).
	appCons, err := u.js.CreateOrUpdateConsumer(ctx, "APPROVALS", jetstream.ConsumerConfig{
		Durable:       fmt.Sprintf("edge-%s-approvals", u.edgeID),
		FilterSubject: "approvals.APPROVED.>",
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("create approval consumer: %w", err)
	}

	u.logger.Info("updater started, waiting for release events")

	// Consume release notifications.
	relCtx, err := relCons.Messages()
	if err != nil {
		return fmt.Errorf("start release message iterator: %w", err)
	}
	defer relCtx.Stop()

	// Consume approval events.
	appCtx, err := appCons.Messages()
	if err != nil {
		return fmt.Errorf("start approval message iterator: %w", err)
	}
	defer appCtx.Stop()

	go u.consumeReleases(ctx, relCtx)
	u.consumeApprovals(ctx, appCtx)
	return nil
}

func (u *Updater) consumeReleases(ctx context.Context, msgs jetstream.MessagesContext) {
	for {
		msg, err := msgs.Next()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			u.logger.Error("release message iterator error", zap.Error(err))
			return
		}

		var event ReleaseEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			u.logger.Warn("unmarshal release event", zap.Error(err))
			_ = msg.Nak()
			continue
		}

		// Skip if this release targets a specific edge that is not us.
		if event.TargetEdge != "" && event.TargetEdge != u.edgeID {
			_ = msg.Ack()
			continue
		}

		u.logger.Info("release notification received",
			zap.String("release_id", event.ReleaseID),
			zap.String("image", event.ImageRef),
		)

		if err := u.requestApproval(ctx, event); err != nil {
			u.logger.Error("request approval failed", zap.Error(err))
			_ = msg.Nak()
			continue
		}

		_ = msg.Ack()
	}
}

func (u *Updater) consumeApprovals(ctx context.Context, msgs jetstream.MessagesContext) {
	for {
		msg, err := msgs.Next()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			u.logger.Error("approval message iterator error", zap.Error(err))
			return
		}

		var event ApprovalEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			u.logger.Warn("unmarshal approval event", zap.Error(err))
			_ = msg.Nak()
			continue
		}

		u.logger.Info("approval received, deploying",
			zap.String("approval_id", event.ApprovalID),
			zap.String("image", event.ImageRef),
		)

		deployErr := u.deploy(ctx, event)
		phase := "COMPLETED"
		progressPct := 100
		errCode := ""
		errMessage := ""
		if deployErr != nil {
			phase = "FAILED"
			progressPct = 0
			errCode = "DEPLOY_ERROR"
			errMessage = deployErr.Error()
			u.logger.Error("deployment failed", zap.Error(deployErr))

			_ = u.reporter.Report(ctx, reporter.DeploymentResult{
				ApprovalID:   event.ApprovalID,
				Phase:        phase,
				ProgressPct:  progressPct,
				ErrorCode:    errCode,
				ErrorMessage: errMessage,
			})

			// Re-queue for retry instead of ACKing on failure.
			_ = msg.Nak()
			continue
		}

		resourceName := sanitizeName(event.ReleaseID)
		if catalogErr := u.applyToCatalog(ctx, event); catalogErr != nil {
			u.logger.Warn("apply to catalog failed (non-fatal)", zap.Error(catalogErr))
			// Fall through to report COMPLETED from deploy() — catalog is best-effort.
		} else {
			// CatalogWatcher will report Ready/Failed/RolledBack when Operator finishes.
			u.catalogWatcher.Watch(ctx, resourceName, event.ApprovalID)
		}

		_ = u.reporter.Report(ctx, reporter.DeploymentResult{
			ApprovalID:  event.ApprovalID,
			Phase:       phase,
			ProgressPct: progressPct,
		})

		// Only ACK on success.
		_ = msg.Ack()
	}
}

// requestApproval sends a deployment approval request to the central API.
func (u *Updater) requestApproval(ctx context.Context, event ReleaseEvent) error {
	payload, err := json.Marshal(approvalRequest{
		ReleaseID: event.ReleaseID,
		EdgeID:    u.edgeID,
		ImageRef:  event.ImageRef,
	})
	if err != nil {
		return fmt.Errorf("marshal approval request: %w", err)
	}

	url := u.centralURL + "/agent/v1/approval-requests"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build approval request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("POST approval request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("central API approval request returned %s", resp.Status)
	}

	var ar approvalResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return fmt.Errorf("decode approval response: %w", err)
	}

	u.logger.Info("approval request created", zap.String("approval_id", ar.ApprovalID))
	return nil
}

// safeFieldPattern matches values safe to embed directly into YAML.
var safeFieldPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-\.\:/\@\_]*$`)

// applyToCatalog applies a CatalogPackage CRD to the cluster via kubectl.
func (u *Updater) applyToCatalog(ctx context.Context, event ApprovalEvent) error {
	// Validate fields to prevent YAML injection.
	if !safeFieldPattern.MatchString(event.ImageRef) {
		return fmt.Errorf("invalid ImageRef format: %q", event.ImageRef)
	}
	if !safeFieldPattern.MatchString(event.ApprovalID) {
		return fmt.Errorf("invalid ApprovalID format: %q", event.ApprovalID)
	}
	if !safeFieldPattern.MatchString(event.ReleaseID) {
		return fmt.Errorf("invalid ReleaseID format: %q", event.ReleaseID)
	}

	yaml := fmt.Sprintf(`apiVersion: edgedip.io/v1alpha1
kind: CatalogPackage
metadata:
  name: %s
  namespace: edgedip
spec:
  packageName: %s
  approvedVersion: "%s"
  approvalId: "%s"
  harborUrl: "%s"
  imageRef: "%s"
  namespace: edgedip
  autoRollback: true
  healthCheckTimeout: "5m"
`, sanitizeName(event.ReleaseID), extractAppName(event.ImageRef), extractVersion(event.ImageRef), event.ApprovalID, u.harborURL, event.ImageRef)

	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(yaml)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply CatalogPackage: %w\noutput: %s", err, string(out))
	}

	u.logger.Info("CatalogPackage applied", zap.String("approval_id", event.ApprovalID))
	return nil
}

func sanitizeName(id string) string {
	// Use first 16 chars of UUID (dashes removed) to reduce collision risk.
	clean := strings.ReplaceAll(id, "-", "")
	if len(clean) > 16 {
		clean = clean[:16]
	}
	return "pkg-" + strings.ToLower(clean)
}

func extractAppName(imageRef string) string {
	parts := strings.Split(imageRef, "/")
	last := parts[len(parts)-1]
	return strings.Split(last, ":")[0]
}

func extractVersion(imageRef string) string {
	parts := strings.Split(imageRef, ":")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return "latest"
}

// deploy pulls the container image from the local Harbor mirror using docker/nerdctl.
func (u *Updater) deploy(ctx context.Context, event ApprovalEvent) error {
	imageRef := event.ImageRef

	// Prefer nerdctl (containerd) if available, fall back to docker.
	runtime := "docker"
	if _, err := exec.LookPath("nerdctl"); err == nil {
		runtime = "nerdctl"
	}

	pullCmd := exec.CommandContext(ctx, runtime, "pull", imageRef)
	out, err := pullCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s pull %s: %w\noutput: %s", runtime, imageRef, err, string(out))
	}

	u.logger.Info("image pulled successfully",
		zap.String("image", imageRef),
		zap.String("runtime", runtime),
	)
	return nil
}
