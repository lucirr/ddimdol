# Edge DIP 전체 구현 결과

> 작성일: 2026-05-13  
> 기반 문서: [edgedip_impl_plan_20260512.md](./edgedip_impl_plan_20260512.md)

---

## 전체 프로젝트 구조

```
didimdol/
├── portal-api/              # 중앙 관제 백엔드 (Go 1.22 + Gin)
├── portal-web/              # 관제 포털 프론트엔드 (React 18 + TypeScript)
├── edge-agent/              # 에지 노드 에이전트 (Go 1.22 정적 바이너리)
├── update-operator/         # 카탈로그 자동 업데이트 k8s Operator (Go)
├── deploy/
│   ├── local/               # 로컬 개발 인프라 (docker-compose)
│   ├── helm/
│   │   ├── portal-api/      # 중앙 Helm 차트
│   │   └── edge-agent/      # 에지 Helm 차트 (DaemonSet)
│   └── argocd/
│       ├── portal-api-app.yaml         # ArgoCD Application
│       └── edge-agent-appset.yaml      # ArgoCD ApplicationSet (다수 에지)
├── pipelines/
│   └── gitea-actions/
│       ├── portal-api-ci.yml    # Go 빌드 + SBOM/CVE/서명
│       ├── edge-agent-ci.yml    # 정적 바이너리 멀티아치 빌드
│       └── release-gate.yml     # 릴리즈 보안 게이트
├── policies/
│   ├── approval.rego        # 승인 RBAC 정책
│   ├── release.rego         # 릴리즈 권한 + CVE 게이트
│   ├── session.rego         # JIT 원격 세션 정책
│   └── test/                # OPA 정책 테스트
└── docs/
```

---

## 컴포넌트별 구현 상세

### portal-api (Go 1.22 + Gin)
- **Repository**: sqlx 실제 구현 (EdgeNode, Release, ApprovalRequest, AuditLog)
  - 낙관적 락(version 필드), SHA-256 해시 체인(AuditLog)
  - JSONB 필드 JSON 마샬링
- **NATS Service**: JetStream 스트림 3개 (RELEASES, EDGE_EVENTS, APPROVALS)
  - 릴리즈 알림 발행, 승인 이벤트 발행, heartbeat 구독 (durable consumer)
- **Handler**: 22개 엔드포인트 완성 (stub → 실제 service 연결)
- **Port**: 8080 (Public API + JWT), 8081 (mTLS Agent 전용)

### portal-web (React 18 + TypeScript)
- TanStack Query hooks (useEdges, useApprovals, useReleases)
- 4개 페이지: Dashboard, Edges, Approvals, Releases
- shadcn/ui 컴포넌트 직접 구현 (Badge, Card, Button)
- Vite proxy: `/api` → `localhost:8080`

### edge-agent (Go 1.22 정적 바이너리)
- **Heartbeat**: 10초 주기 `edge.heartbeat.<edge-id>` NATS 발행 (gopsutil 실측)
- **Updater**: `releases.published.>` 구독 → 중앙 API에 승인 요청 → `approvals.APPROVED.*` 구독 → Harbor pull
- **Reporter**: 배포 결과 `/agent/v1/deployment-result` POST
- 빌드: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` 7.2MB 정적 바이너리

### update-operator (controller-runtime v0.17)
- **CRD**: `CatalogPackage`, `CatalogRelease` (edgedip.io/v1alpha1)
- **Reconcile**: Idle → Downloading → Applying → HealthCheck → Ready/RolledBack
- **CVE 게이트**: critical > 0이면 CatalogRelease 발행 차단
- CRD YAML 매니페스트 포함

### deploy/helm
- portal-api: Deployment 2 replicas, 듀얼 포트 Service, ConfigMap
- edge-agent: DaemonSet (hostPID, /proc /sys 마운트)
- ArgoCD ApplicationSet: list generator로 N개 에지 자동 배포

### pipelines/gitea-actions
- **portal-api-ci**: 테스트(커버리지 ≥80%) → Docker 빌드 → Syft SBOM → Trivy+Grype → Cosign 서명 → Harbor push
- **edge-agent-ci**: 테스트 → 멀티아치 정적 빌드(amd64/arm64) → SHA256 → Cosign blob 서명
- **release-gate**: CRITICAL=0, HIGH≤3 게이트 → 버전 태그 → Portal API 알림

### policies/OPA
- **approval.rego**: central-operator(전체), edge-admin(자기 에지), auditor(읽기)
- **release.rego**: 발행 권한 + CVE critical > 0 차단
- **session.rego**: TTL ≤ 1800s, reason 필수, auditor 녹화 조회
- 테스트: approval 5케이스, session 9케이스

---

## 빌드 검증 결과

| 컴포넌트 | 빌드 | 비고 |
|---------|------|------|
| portal-api | PASS | `go build ./...` |
| portal-web | PASS | `yarn build` (0 TS errors) |
| edge-agent | PASS | `go build ./...` + 정적 바이너리 7.2MB |
| update-operator | PASS | `go build ./...` |

---

## 로컬 실행

```bash
# 1. 인프라 (PG + Redis + NATS + Keycloak)
cd deploy/local && make up

# 2. 백엔드
cd portal-api && make run        # :8080 / :8081

# 3. 프론트엔드
cd portal-web && yarn dev        # :5173

# 4. 에지 에이전트 (에지 노드에서)
cd edge-agent && make build-static
EDGE_ID=<uuid> EDGE_NAME=edge-01 NATS_URL=nats://central:4222 ./bin/edge-agent-linux-amd64
```

---

## 남은 작업 (실제 구현 필요)

| 항목 | 상태 | 우선순위 |
|------|------|---------|
| Keycloak OIDC JWT 실제 검증 | stub | HIGH |
| Helm upgrade exec (update-operator) | 구조만 | HIGH |
| Harbor 이미지 pull 실행 (edge-agent) | 구조만 | HIGH |
| Prometheus/Thanos 메트릭 파이프라인 | 미구현 | MEDIUM |
| Apache Guacamole JIT 세션 | 미구현 | MEDIUM (Phase 4) |
| 성능 시험 자동화 스크립트 | 미구현 | LOW (Phase 5) |
