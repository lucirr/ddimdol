package controller

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/your-org/update-operator/internal/api/v1alpha1"
)

type CatalogPackageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger *zap.Logger
}

func (r *CatalogPackageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger.With(zap.String("name", req.NamespacedName.String()))

	var pkg v1alpha1.CatalogPackage
	if err := r.Get(ctx, req.NamespacedName, &pkg); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 이미 최신 버전이면 스킵
	if pkg.Status.InstalledVersion == pkg.Spec.ApprovedVersion &&
		pkg.Status.Phase == v1alpha1.PackagePhaseReady {
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	log.Info("reconciling catalog package",
		zap.String("package", pkg.Spec.PackageName),
		zap.String("target", pkg.Spec.ApprovedVersion))

	// 상태를 Downloading으로 업데이트
	pkg.Status.Phase = v1alpha1.PackagePhaseDownloading
	if err := r.Status().Update(ctx, &pkg); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status: %w", err)
	}

	// TODO(CRITICAL): This is a STUB implementation. The actual image pull and Helm upgrade
	// logic must be implemented before production use. Currently this controller marks every
	// CatalogPackage as Ready instantly without performing any real operation.
	// See: https://helm.sh/docs/topics/advanced/#programmatic-access-to-helm-commands

	// Harbor에서 이미지 pull (실제 구현에서는 exec 또는 k8s Job)
	log.Info("pulling image from harbor",
		zap.String("url", pkg.Spec.HarborURL),
		zap.String("package", pkg.Spec.PackageName),
		zap.String("version", pkg.Spec.ApprovedVersion))

	// Helm upgrade 실행 (실제: helm Go SDK 또는 exec)
	pkg.Status.Phase = v1alpha1.PackagePhaseApplying
	if err := r.Status().Update(ctx, &pkg); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status to Applying: %w", err)
	}

	// 헬스체크 (30초 대기 후 확인)
	pkg.Status.Phase = v1alpha1.PackagePhaseHealthCheck
	if err := r.Status().Update(ctx, &pkg); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status to HealthCheck: %w", err)
	}

	// 성공
	now := metav1.Now()
	pkg.Status.Phase = v1alpha1.PackagePhaseReady
	pkg.Status.InstalledVersion = pkg.Spec.ApprovedVersion
	pkg.Status.LastAppliedAt = &now
	pkg.Status.Message = "Successfully applied"
	if err := r.Status().Update(ctx, &pkg); err != nil {
		return ctrl.Result{}, fmt.Errorf("update status to Ready: %w", err)
	}

	log.Info("catalog package reconciled successfully",
		zap.String("package", pkg.Spec.PackageName),
		zap.String("version", pkg.Spec.ApprovedVersion))

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *CatalogPackageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CatalogPackage{}).
		Complete(r)
}
