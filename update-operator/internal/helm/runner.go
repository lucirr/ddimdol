package helm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/client-go/rest"
)

// Runner wraps the Helm Go SDK for install/upgrade/rollback operations.
type Runner struct {
	restCfg *rest.Config
	logger  *zap.Logger
}

func NewRunner(restCfg *rest.Config, logger *zap.Logger) *Runner {
	return &Runner{restCfg: restCfg, logger: logger}
}

// InstallOrUpgrade installs the chart if no release exists, or upgrades an existing one.
// chartPath: local chart directory or OCI reference (oci://harbor/charts/name:version)
// namespace: target k8s namespace
// releaseName: helm release name
// imageRef: override image in values (values.image.repository + tag derived from imageRef)
// valuesJSON: additional values override as JSON string (optional)
func (r *Runner) InstallOrUpgrade(
	ctx context.Context,
	chartPath, namespace, releaseName, imageRef, valuesJSON string,
	timeout time.Duration,
) (*release.Release, error) {
	cfg, err := r.actionConfig(namespace)
	if err != nil {
		return nil, fmt.Errorf("helm action config: %w", err)
	}

	vals, err := parseValues(imageRef, valuesJSON)
	if err != nil {
		return nil, fmt.Errorf("parse values: %w", err)
	}

	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("load chart %q: %w", chartPath, err)
	}

	// Check if release already exists
	histClient := action.NewHistory(cfg)
	histClient.Max = 1
	_, histErr := histClient.Run(releaseName)

	if histErr == driver.ErrReleaseNotFound {
		// First install
		install := action.NewInstall(cfg)
		install.ReleaseName = releaseName
		install.Namespace = namespace
		install.CreateNamespace = true
		install.Timeout = timeout
		install.Wait = true
		install.Atomic = false // we handle rollback ourselves

		r.logger.Info("helm install", zap.String("release", releaseName), zap.String("chart", chartPath))
		rel, err := install.RunWithContext(ctx, ch, vals)
		if err != nil {
			return nil, fmt.Errorf("helm install %q: %w", releaseName, err)
		}
		return rel, nil
	}

	if histErr != nil {
		return nil, fmt.Errorf("helm history %q: %w", releaseName, histErr)
	}

	// Upgrade existing release
	upgrade := action.NewUpgrade(cfg)
	upgrade.Namespace = namespace
	upgrade.Timeout = timeout
	upgrade.Wait = true
	upgrade.Atomic = false
	upgrade.CleanupOnFail = false

	r.logger.Info("helm upgrade", zap.String("release", releaseName), zap.String("chart", chartPath))
	rel, err := upgrade.RunWithContext(ctx, releaseName, ch, vals)
	if err != nil {
		return nil, fmt.Errorf("helm upgrade %q: %w", releaseName, err)
	}
	return rel, nil
}

// Rollback rolls the release back to the previous revision.
func (r *Runner) Rollback(ctx context.Context, namespace, releaseName string) error {
	cfg, err := r.actionConfig(namespace)
	if err != nil {
		return fmt.Errorf("helm action config: %w", err)
	}

	rb := action.NewRollback(cfg)
	rb.Version = 0 // 0 = previous revision
	rb.Wait = true
	rb.Timeout = 5 * time.Minute

	r.logger.Info("helm rollback", zap.String("release", releaseName))
	if err := rb.Run(releaseName); err != nil {
		return fmt.Errorf("helm rollback %q: %w", releaseName, err)
	}
	return nil
}

// CurrentRevision returns the current helm revision number for a release, or 0 if not installed.
func (r *Runner) CurrentRevision(namespace, releaseName string) (int, error) {
	cfg, err := r.actionConfig(namespace)
	if err != nil {
		return 0, fmt.Errorf("helm action config: %w", err)
	}
	get := action.NewGet(cfg)
	rel, err := get.Run(releaseName)
	if err != nil {
		if err == driver.ErrReleaseNotFound {
			return 0, nil
		}
		return 0, fmt.Errorf("helm get %q: %w", releaseName, err)
	}
	return rel.Version, nil
}

func (r *Runner) actionConfig(namespace string) (*action.Configuration, error) {
	cfg := new(action.Configuration)
	getter := newRESTClientGetter(r.restCfg, namespace)
	if err := cfg.Init(getter, namespace, "secret", func(format string, v ...interface{}) {
		r.logger.Sugar().Debugf("[helm] "+format, v...)
	}); err != nil {
		return nil, err
	}
	return cfg, nil
}

// parseValues builds a helm values map.
// imageRef (e.g. "harbor.local/myapp:1.2.3") is injected under image.repository / image.tag.
// valuesJSON is merged on top.
func parseValues(imageRef, valuesJSON string) (map[string]interface{}, error) {
	vals := map[string]interface{}{}

	if imageRef != "" {
		repo, tag := splitImageRef(imageRef)
		vals["image"] = map[string]interface{}{
			"repository": repo,
			"tag":        tag,
		}
	}

	if valuesJSON != "" {
		var extra map[string]interface{}
		if err := json.Unmarshal([]byte(valuesJSON), &extra); err != nil {
			return nil, fmt.Errorf("invalid values JSON: %w", err)
		}
		for k, v := range extra {
			vals[k] = v
		}
	}
	return vals, nil
}

// splitImageRef splits "registry/repo/name:tag" into (registry/repo/name, tag).
func splitImageRef(ref string) (string, string) {
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == ':' {
			return ref[:i], ref[i+1:]
		}
		if ref[i] == '/' {
			break
		}
	}
	return ref, "latest"
}
