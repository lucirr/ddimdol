package controller

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/your-org/update-operator/internal/api/v1alpha1"
)

type CatalogReleaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger *zap.Logger
}

func (r *CatalogReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger.With(zap.String("name", req.NamespacedName.String()))

	var release v1alpha1.CatalogRelease
	if err := r.Get(ctx, req.NamespacedName, &release); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 이미 발행된 릴리즈는 스킵
	if release.Status.Phase == v1alpha1.ReleasePhasePublished ||
		release.Status.Phase == v1alpha1.ReleasePhaseDeprecated {
		return ctrl.Result{}, nil
	}

	log.Info("reconciling catalog release",
		zap.String("release", release.Spec.ReleaseName),
		zap.String("version", release.Spec.Version))

	// CVE gate 검사
	if release.Spec.CVEReport != nil && release.Spec.CVEReport.Critical > 0 {
		release.Status.Phase = v1alpha1.ReleasePhaseDraft
		release.Status.Message = fmt.Sprintf(
			"publish blocked: %d critical CVEs detected", release.Spec.CVEReport.Critical)
		if err := r.Status().Update(ctx, &release); err != nil {
			return ctrl.Result{}, fmt.Errorf("update status: %w", err)
		}
		log.Warn("release publish blocked due to critical CVEs",
			zap.String("release", release.Spec.ReleaseName),
			zap.Int("critical", release.Spec.CVEReport.Critical))
		return ctrl.Result{}, nil
	}

	// 발행 처리
	now := metav1.Now()
	release.Status.Phase = v1alpha1.ReleasePhasePublished
	release.Status.PublishedAt = &now
	release.Status.Message = "Release published successfully"
	if err := r.Status().Update(ctx, &release); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status to Published: %w", err)
	}

	log.Info("catalog release published successfully",
		zap.String("release", release.Spec.ReleaseName),
		zap.String("version", release.Spec.Version))

	return ctrl.Result{}, nil
}

func (r *CatalogReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CatalogRelease{}).
		Complete(r)
}
