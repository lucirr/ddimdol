package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// CatalogPackageSpec은 설치할 패키지를 정의합니다
type CatalogPackageSpec struct {
	// PackageName은 Harbor 레지스트리의 패키지명
	PackageName string `json:"packageName"`
	// ApprovedVersion은 승인된 버전 (approval ID로부터 설정)
	ApprovedVersion string `json:"approvedVersion"`
	// ApprovalID는 이 업데이트를 승인한 ApprovalRequest ID
	ApprovalID string `json:"approvalId"`
	// HarborURL은 로컬 Harbor 미러 주소
	HarborURL string `json:"harborUrl"`
	// AutoRollback이 true면 헬스체크 실패 시 자동 롤백
	AutoRollback bool `json:"autoRollback,omitempty"`
}

type PackagePhase string

const (
	PackagePhaseIdle        PackagePhase = "Idle"
	PackagePhaseDownloading PackagePhase = "Downloading"
	PackagePhaseApplying    PackagePhase = "Applying"
	PackagePhaseHealthCheck PackagePhase = "HealthCheck"
	PackagePhaseReady       PackagePhase = "Ready"
	PackagePhaseFailed      PackagePhase = "Failed"
	PackagePhaseRolledBack  PackagePhase = "RolledBack"
)

type CatalogPackageStatus struct {
	Phase            PackagePhase `json:"phase,omitempty"`
	InstalledVersion string       `json:"installedVersion,omitempty"`
	LastAppliedAt    *metav1.Time `json:"lastAppliedAt,omitempty"`
	Message          string       `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Package",type=string,JSONPath=`.spec.packageName`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.approvedVersion`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
type CatalogPackage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CatalogPackageSpec   `json:"spec,omitempty"`
	Status            CatalogPackageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type CatalogPackageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CatalogPackage `json:"items"`
}
