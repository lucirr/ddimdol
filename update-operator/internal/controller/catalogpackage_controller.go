package controller

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/your-org/update-operator/internal/api/v1alpha1"
	helmrunner "github.com/your-org/update-operator/internal/helm"
)

const (
	defaultNamespace          = "default"
	defaultHealthCheckTimeout = 5 * time.Minute
	requeueInterval           = 5 * time.Minute
)

type CatalogPackageReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Logger  *zap.Logger
	RestCfg *rest.Config
}

func (r *CatalogPackageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger.With(zap.String("name", req.NamespacedName.String()))

	var pkg v1alpha1.CatalogPackage
	if err := r.Get(ctx, req.NamespacedName, &pkg); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 이미 최신 버전이면 주기적 재확인만
	if pkg.Status.InstalledVersion == pkg.Spec.ApprovedVersion &&
		pkg.Status.Phase == v1alpha1.PackagePhaseReady {
		return ctrl.Result{RequeueAfter: requeueInterval}, nil
	}

	// 이미 진행 중인 다른 reconcile이 실패한 경우 재시도
	if pkg.Status.Phase == v1alpha1.PackagePhaseFailed {
		log.Info("retrying previously failed package",
			zap.String("package", pkg.Spec.PackageName))
	}

	log.Info("reconciling catalog package",
		zap.String("package", pkg.Spec.PackageName),
		zap.String("current", pkg.Status.InstalledVersion),
		zap.String("target", pkg.Spec.ApprovedVersion))

	return r.reconcilePackage(ctx, log, &pkg)
}

func (r *CatalogPackageReconciler) reconcilePackage(
	ctx context.Context,
	log *zap.Logger,
	pkg *v1alpha1.CatalogPackage,
) (ctrl.Result, error) {
	helm := helmrunner.NewRunner(r.RestCfg, r.Logger)

	namespace := pkg.Spec.Namespace
	if namespace == "" {
		namespace = defaultNamespace
	}
	releaseName := pkg.Spec.HelmReleaseName
	if releaseName == "" {
		releaseName = pkg.Spec.PackageName
	}
	chartPath := pkg.Spec.ChartPath
	if chartPath == "" {
		// 차트 경로 미지정 시 Harbor OCI 레퍼런스에서 유도
		chartPath = fmt.Sprintf("oci://%s/charts/%s", pkg.Spec.HarborURL, pkg.Spec.PackageName)
	}

	timeout := defaultHealthCheckTimeout
	if pkg.Spec.HealthCheckTimeout != "" {
		if d, err := time.ParseDuration(pkg.Spec.HealthCheckTimeout); err == nil {
			timeout = d
		}
	}

	// 이전 버전 기록
	previousVersion := pkg.Status.InstalledVersion

	// --- Phase: Downloading ---
	if err := r.setPhase(ctx, pkg, v1alpha1.PackagePhaseDownloading, "Pulling chart from Harbor"); err != nil {
		return ctrl.Result{}, err
	}
	log.Info("downloading chart",
		zap.String("chart", chartPath),
		zap.String("image", pkg.Spec.ImageRef))

	// --- Phase: Applying (Helm install/upgrade) ---
	if err := r.setPhase(ctx, pkg, v1alpha1.PackagePhaseApplying, "Running helm install/upgrade"); err != nil {
		return ctrl.Result{}, err
	}

	rel, err := helm.InstallOrUpgrade(
		ctx,
		chartPath,
		namespace,
		releaseName,
		pkg.Spec.ImageRef,
		pkg.Spec.Values,
		timeout,
	)
	if err != nil {
		log.Error("helm install/upgrade failed",
			zap.String("package", pkg.Spec.PackageName),
			zap.Error(err))

		if pkg.Spec.AutoRollback && previousVersion != "" {
			return r.performRollback(ctx, log, helm, pkg, namespace, releaseName, err.Error())
		}

		return ctrl.Result{}, r.setFailed(ctx, pkg, fmt.Sprintf("helm failed: %v", err))
	}

	helmRevision := 0
	if rel != nil {
		helmRevision = rel.Version
	}

	// --- Phase: HealthCheck ---
	if err := r.setPhase(ctx, pkg, v1alpha1.PackagePhaseHealthCheck, "Waiting for pods to become ready"); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.waitForRollout(ctx, log, namespace, releaseName, timeout); err != nil {
		log.Error("health check failed",
			zap.String("package", pkg.Spec.PackageName),
			zap.Error(err))

		if pkg.Spec.AutoRollback && previousVersion != "" {
			return r.performRollback(ctx, log, helm, pkg, namespace, releaseName, err.Error())
		}
		return ctrl.Result{}, r.setFailed(ctx, pkg, fmt.Sprintf("health check failed: %v", err))
	}

	// --- Phase: Ready ---
	now := metav1.Now()
	pkg.Status.Phase = v1alpha1.PackagePhaseReady
	pkg.Status.PreviousVersion = previousVersion
	pkg.Status.InstalledVersion = pkg.Spec.ApprovedVersion
	pkg.Status.HelmRevision = helmRevision
	pkg.Status.LastAppliedAt = &now
	pkg.Status.Message = fmt.Sprintf("Successfully upgraded to %s", pkg.Spec.ApprovedVersion)

	if err := r.Status().Update(ctx, pkg); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, fmt.Errorf("update status Ready: %w", err)
	}

	log.Info("catalog package ready",
		zap.String("package", pkg.Spec.PackageName),
		zap.String("version", pkg.Spec.ApprovedVersion),
		zap.Int("helm_revision", helmRevision))

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *CatalogPackageReconciler) performRollback(
	ctx context.Context,
	log *zap.Logger,
	helm *helmrunner.Runner,
	pkg *v1alpha1.CatalogPackage,
	namespace, releaseName, reason string,
) (ctrl.Result, error) {
	log.Warn("initiating auto-rollback",
		zap.String("package", pkg.Spec.PackageName),
		zap.String("reason", reason))

	if err := helm.Rollback(ctx, namespace, releaseName); err != nil {
		log.Error("rollback failed", zap.Error(err))
		return ctrl.Result{}, r.setFailed(ctx, pkg,
			fmt.Sprintf("upgrade failed (%s) and rollback also failed: %v", reason, err))
	}

	pkg.Status.Phase = v1alpha1.PackagePhaseRolledBack
	pkg.Status.InstalledVersion = pkg.Status.PreviousVersion
	pkg.Status.Message = fmt.Sprintf("Rolled back due to: %s", reason)
	if err := r.Status().Update(ctx, pkg); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, fmt.Errorf("update status RolledBack: %w", err)
	}

	log.Info("rollback completed",
		zap.String("package", pkg.Spec.PackageName),
		zap.String("restored_version", pkg.Status.PreviousVersion))

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// waitForRollout polls until all pods in the helm release are ready or timeout.
// Uses kubectl rollout status via the k8s API through controller-runtime client.
func (r *CatalogPackageReconciler) waitForRollout(
	ctx context.Context,
	log *zap.Logger,
	namespace, releaseName string,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			ready, err := r.checkDeploymentsReady(ctx, namespace, releaseName)
			if err != nil {
				log.Warn("health check poll error", zap.Error(err))
			}
			if ready {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("health check timed out after %s", timeout)
			}
			log.Info("waiting for rollout",
				zap.String("release", releaseName),
				zap.String("remaining", time.Until(deadline).Round(time.Second).String()))
		}
	}
}

// checkDeploymentsReady checks if all Deployments with the helm release label are fully ready.
func (r *CatalogPackageReconciler) checkDeploymentsReady(ctx context.Context, namespace, releaseName string) (bool, error) {
	var deployList appsv1.DeploymentList
	if err := r.List(ctx, &deployList,
		client.InNamespace(namespace),
		client.MatchingLabels{"app.kubernetes.io/instance": releaseName},
	); err != nil {
		return false, fmt.Errorf("list deployments: %w", err)
	}

	if len(deployList.Items) == 0 {
		// No deployments found yet — give it time
		return false, nil
	}

	for i := range deployList.Items {
		d := &deployList.Items[i]
		desired := int32(1)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		if d.Status.ReadyReplicas < desired {
			return false, nil
		}
	}
	return true, nil
}

func (r *CatalogPackageReconciler) setPhase(ctx context.Context, pkg *v1alpha1.CatalogPackage, phase v1alpha1.PackagePhase, msg string) error {
	pkg.Status.Phase = phase
	pkg.Status.Message = msg
	if err := r.Status().Update(ctx, pkg); err != nil {
		if apierrors.IsConflict(err) {
			return nil // will retry
		}
		return fmt.Errorf("update status %s: %w", phase, err)
	}
	return nil
}

func (r *CatalogPackageReconciler) setFailed(ctx context.Context, pkg *v1alpha1.CatalogPackage, msg string) error {
	pkg.Status.Phase = v1alpha1.PackagePhaseFailed
	pkg.Status.Message = msg
	if err := r.Status().Update(ctx, pkg); err != nil {
		if apierrors.IsConflict(err) {
			return nil
		}
		return fmt.Errorf("update status Failed: %w", err)
	}
	return nil
}

func (r *CatalogPackageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CatalogPackage{}).
		Complete(r)
}
