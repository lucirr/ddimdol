package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/your-org/edge-agent/internal/reporter"
	"go.uber.org/zap"
)

const (
	watchInterval    = 15 * time.Second
	terminalTimeout  = 30 * time.Minute // give up watching after this
)

// catalogPackageStatus is a partial unmarshal of a CatalogPackage object from kubectl.
type catalogPackageStatus struct {
	Spec struct {
		ApprovalID string `json:"approvalId"`
	} `json:"spec"`
	Status struct {
		Phase   string `json:"phase"`
		Message string `json:"message"`
	} `json:"status"`
}

// watchEntry tracks a single CatalogPackage pending terminal phase.
type watchEntry struct {
	resourceName string
	approvalID   string
	startedAt    time.Time
}

// CatalogWatcher polls CatalogPackage resources and reports terminal phases to central.
type CatalogWatcher struct {
	reporter  *reporter.Reporter
	namespace string
	logger    *zap.Logger
}

func newCatalogWatcher(rep *reporter.Reporter, namespace string, logger *zap.Logger) *CatalogWatcher {
	return &CatalogWatcher{
		reporter:  rep,
		namespace: namespace,
		logger:    logger,
	}
}

// Watch registers a new CatalogPackage to watch for terminal status.
// It spawns a goroutine per entry so multiple packages can be tracked concurrently.
func (w *CatalogWatcher) Watch(ctx context.Context, resourceName, approvalID string) {
	entry := watchEntry{
		resourceName: resourceName,
		approvalID:   approvalID,
		startedAt:    time.Now(),
	}
	go w.watchOne(ctx, entry)
}

func (w *CatalogWatcher) watchOne(ctx context.Context, e watchEntry) {
	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if time.Since(e.startedAt) > terminalTimeout {
				w.logger.Warn("catalog watch timed out, giving up",
					zap.String("resource", e.resourceName),
					zap.String("approval_id", e.approvalID))
				return
			}

			phase, msg, err := w.getPhase(e.resourceName)
			if err != nil {
				w.logger.Warn("catalog watch poll failed",
					zap.String("resource", e.resourceName),
					zap.Error(err))
				continue
			}

			switch phase {
			case "Ready":
				w.report(ctx, e.approvalID, "COMPLETED", 100, "", "")
				return
			case "Failed":
				w.report(ctx, e.approvalID, "FAILED", 0, "DEPLOY_ERROR", msg)
				return
			case "RolledBack":
				w.report(ctx, e.approvalID, "ROLLED_BACK", 0, "ROLLBACK", msg)
				return
			default:
				w.logger.Debug("catalog package still in progress",
					zap.String("resource", e.resourceName),
					zap.String("phase", phase))
			}
		}
	}
}

func (w *CatalogWatcher) getPhase(resourceName string) (phase, message string, err error) {
	out, err := exec.Command(
		"kubectl", "get",
		"catalogpackage", resourceName,
		"-n", w.namespace,
		"-o", "json",
	).Output()
	if err != nil {
		return "", "", fmt.Errorf("kubectl get catalogpackage %s: %w", resourceName, err)
	}

	var cp catalogPackageStatus
	if err := json.Unmarshal(out, &cp); err != nil {
		return "", "", fmt.Errorf("unmarshal catalogpackage: %w", err)
	}
	return cp.Status.Phase, cp.Status.Message, nil
}

func (w *CatalogWatcher) report(ctx context.Context, approvalID, phase string, pct int, code, msg string) {
	if err := w.reporter.Report(ctx, reporter.DeploymentResult{
		ApprovalID:   approvalID,
		Phase:        phase,
		ProgressPct:  pct,
		ErrorCode:    code,
		ErrorMessage: msg,
	}); err != nil {
		w.logger.Error("failed to report catalog result",
			zap.String("approval_id", approvalID),
			zap.String("phase", phase),
			zap.Error(err))
	} else {
		w.logger.Info("catalog result reported",
			zap.String("approval_id", approvalID),
			zap.String("phase", phase))
	}
}
