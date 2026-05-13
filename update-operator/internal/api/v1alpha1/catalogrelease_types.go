package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// CatalogReleaseSpec은 릴리즈를 정의합니다
type CatalogReleaseSpec struct {
	// ReleaseName은 릴리즈 식별자
	ReleaseName string `json:"releaseName"`
	// Version은 릴리즈 버전
	Version string `json:"version"`
	// HarborURL은 소스 Harbor 레지스트리 주소
	HarborURL string `json:"harborUrl"`
	// Packages는 이 릴리즈에 포함된 패키지 목록
	Packages []ReleasePackage `json:"packages"`
	// CVEReport는 CVE 스캔 결과 요약
	CVEReport *CVEReport `json:"cveReport,omitempty"`
}

type ReleasePackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Digest  string `json:"digest,omitempty"`
}

type CVEReport struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

type ReleasePhase string

const (
	ReleasePhaseDraft      ReleasePhase = "Draft"
	ReleasePhasePublished  ReleasePhase = "Published"
	ReleasePhaseDeprecated ReleasePhase = "Deprecated"
)

type CatalogReleaseStatus struct {
	Phase       ReleasePhase `json:"phase,omitempty"`
	PublishedAt *metav1.Time `json:"publishedAt,omitempty"`
	Message     string       `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Release",type=string,JSONPath=`.spec.releaseName`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
type CatalogRelease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CatalogReleaseSpec   `json:"spec,omitempty"`
	Status            CatalogReleaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type CatalogReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CatalogRelease `json:"items"`
}
