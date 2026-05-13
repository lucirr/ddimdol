# EdgeDIP 전체 구현 세션 기록 (2026-05-13)

## 개요

portal-api + portal-web + edge-agent + update-operator 전체 구현 및 통합 검증 세션.
이전 세션(session_approval_workflow_20260513.md) 이후 이어진 작업.

---

## Q&A 기록

---

### Q: 승인관리 ui 에서 승인요청한 건에 대해 승인하면 그 이후 프로세스가 어떻게 처리되나

**A:**
현재는 승인 후 아무것도 일어나지 않음. DB에 status만 APPROVED로 바뀌고 끝.

설계상 승인 이후 전체 흐름:
```
승인 클릭
   │
   ▼
approval_requests.status = APPROVED  ← 당시 여기까지만 구현
   │
   ▼  (미구현) NATS "APPROVALS" 스트림 이벤트 발행
   ▼  (미구현) edge-agent NATS 구독 → 패키지 다운로드
   ▼  (미구현) update-operator CatalogPackage CRD 생성/업데이트
   ▼  (미구현) Kubernetes 배포
   ▼  (미구현) edge-agent → /agent/v1/deployment-result 보고
   ▼  (미구현) deployment_records 테이블 업데이트
```

---

### Q: 대시보드상의 에지 상태는 실시간 동기화 될 수 있는가

**A:**
현재는 단순 1회 조회, 실시간 아님. TanStack Query 기본 staleTime=0으로 포커스 시 재조회는 되지만 진정한 실시간은 아님.

**실시간 방법:**
- 폴링: `refetchInterval: 10000` 추가 (단순)
- SSE/WebSocket: portal-api에 SSE 엔드포인트 추가 (진짜 실시간)

heartbeat 파이프라인은 설계되어 있음:
```
edge-agent → NATS "edge.heartbeat.>" → portal-api SubscribeHeartbeats() → DB 업데이트
```
단, AgentHandler.Heartbeat가 TODO 스텁 상태라 파이프라인 끊겨 있었음.

---

### Q: ui 에서 릴리즈 등록시 메타정보만 db 에 저장하는 것 같은데, 릴리즈 아티팩트 다운로드는 무엇을 한다는 건가

**A:**
`artifact_digest`가 가리키는 실제 아티팩트는 Harbor(OCI 레지스트리)에 있어야 함.

설계된 흐름:
```
개발자
  → CI 파이프라인 (Gitea Actions)
    → Docker 이미지 빌드 + Harbor에 push
    → Cosign으로 서명
    → portal-api에 릴리즈 메타 등록 (artifact_digest, image_ref)

승인 후
  → edge-agent updater.go의 deploy()
    → nerdctl/docker pull harbor.{edgeName}.internal/myapp:v1.2.3
    → update-operator가 CatalogPackage CRD에 새 imageRef 기록
    → K8s가 Pod 재시작
```

당시 UI 릴리즈 등록 폼에 `image_ref` 필드 없어서 추가 필요 확인.

---

### Q: 에지의 카탈로그 배포는 argocd 기반의 gitops 방식을 생각하고 있는데, 현재 로컬환경은 git 기반으로 되어 있는가

**A:**
현재 로컬 환경: Git 기반 아님.

`edge-agent-appset.yaml`의 `repoURL: http://gitea.internal/edgedip/infra` 및 `destination.server`가 에지 K8s 클러스터를 가리키지만 로컬 docker-compose에는 Gitea, ArgoCD, K8s, Harbor 모두 없음.

| 필요한 것 | 로컬 여부 |
|-----------|---------|
| Gitea | ❌ |
| ArgoCD | ❌ |
| 에지 K8s | ❌ |
| Harbor | ❌ |

**현실적 접근:** 로컬은 NATS 이벤트 → edge-agent docker pull 수준까지만 검증, 스테이징/운영에서 GitOps 전체 검증.

---

### Q: update-operator 는 argocd 와 관련있는 건가?

**A:**
**직접 관련 없음.** 완전히 별개의 컴포넌트.

```
ArgoCD                          update-operator
──────────────────────────────  ──────────────────────────────────────
"Git 저장소 → K8s 동기화"        "승인 이벤트 → 패키지 배포"
(GitOps 인프라 도구)             (도메인 전용 K8s Operator)
```

ArgoCD = update-operator 자체를 에지 클러스터에 설치하는 용도.
update-operator = CatalogPackage CRD 감시 → Harbor pull → Helm 배포 실행.

---

### Q: CatalogPackage crd 는 여기서 만든 소스인가?

**A:**
네, 이 프로젝트에서 직접 만든 소스.

| 파일 | 역할 |
|------|------|
| `update-operator/internal/api/v1alpha1/catalogpackage_types.go` | Go 타입 정의 |
| `update-operator/config/crd/catalogpackage.yaml` | K8s CRD YAML |
| `update-operator/internal/controller/catalogpackage_controller.go` | Reconcile 루프 |

group: `edgedip.io`, 실제 클러스터 적용: `kubectl apply -f config/crd/catalogpackage.yaml`

---

### Q: 에지에서 중앙의 harbor 에 접근 가능해야 하는 건가?

**A:**
**아님.** 에지는 중앙 Harbor에 직접 접근하지 않음. 에지 로컬에 Harbor 미러가 있는 설계.

```
중앙 Harbor → replication → 에지 Harbor Mirror
                                    ↓
                              update-operator pull (로컬)
                                    ↓
                              K8s Pod 배포
```

`CatalogPackage.Spec.HarborURL`이 `harbor.{edgeName}.internal` (로컬 주소)로 설정됨.

**미구현 부분:** 중앙 Harbor → 에지 Harbor 복제 트리거 로직 없음. Harbor Replication 기능 활용 예정.

---

### Q: ultrawork 현재 까지 미완성된 요건과 추가 다음 요건 모두 구현해줘

**미완성 요건:**
- NATS 이벤트 발행 (승인 후)
- deployment-result 수신
- edge-agent: NATS 구독 → 릴리즈 아티팩트 다운로드 → update-operator에 적용
- update-operator: CatalogPackage CRD reconcile → Helm/ArgoCD 배포 트리거
- portal-api: /agent/v1/deployment-result 핸들러 → deployment_records 저장
- edge-agent 배포 이벤트 수신 + 결과 보고

**추가 요건:**
- websocket 기반 에지 상태 실시간 모니터링
- 릴리즈 메타 관련 image_ref 필드 추가
- 중앙에서 릴리즈 승인 시 에지 Harbor로 push replication 트리거

**A (구현 완료):**

#### Wave 1 - portal-api

**image_ref 필드 추가:**
- `domain/release.go`: `ImageRef string` 필드 추가
- `handler/release.go`: 요청 struct에 image_ref 추가
- `repository/postgres/release.go`: SELECT/INSERT에 image_ref 추가

**NATS 승인이벤트 발행:**
- `service/nats.go`: ApprovalEvent에 ReleaseID, ImageRef 필드 추가, PublishHeartbeatEvent 추가
- `handler/approval.go`: ApprovalHandler에 rel, edgeRepo, nats, harbor 필드 추가. Approve 후 PublishApprovalEvent 호출, Harbor replication 트리거
- `service/harbor.go` 신규: HarborService.TriggerReplication() - Harbor API v2로 replication policy 조회 후 실행

**deployment-result 핸들러:**
- `handler/agent.go` 완전 재작성: Heartbeat (edge 상태 업데이트 + NATS 발행), DeploymentResult (deployment_records 저장 + approval APPLIED 전환)
- `repository/postgres/deployment.go` 신규: FindByApprovalID, Save(upsert), UpdatePhase
- `config/config.go`: HarborURL, HarborUser, HarborPassword 환경변수 추가

**WebSocket:**
- `internal/hub/hub.go` 신규: WebSocket Hub (broadcast, register/unregister, client read/write pump)
- `handler/ws.go` 신규: WsHandler - HTTP→WebSocket upgrade, `/api/v1/ws/edges`
- `cmd/server/main.go`: Hub 초기화, NATS heartbeat→WebSocket broadcast 연결

#### Wave 2 - edge-agent

- `updater.go`: ApprovalEvent에 EdgeID 추가, APPROVALS 스트림 consumer로 변경, applyToCatalog() 추가 (kubectl apply CatalogPackage YAML)
- `reporter.go`: ErrorMessage, ProgressPct 필드 portal-api 형식에 맞게 수정

#### Wave 2 - DB 마이그레이션

`migrations/007_add_image_ref_and_applied_status.sql`:
```sql
ALTER TABLE releases ADD COLUMN IF NOT EXISTS image_ref TEXT NOT NULL DEFAULT '';
ALTER TYPE approval_status ADD VALUE IF NOT EXISTS 'APPLIED';
```

#### Wave 2 - 프론트엔드

- `types/release.ts`: image_ref 필드 추가
- `hooks/useReleases.ts`: useCreateRelease payload에 image_ref 추가
- `pages/ReleasesPage.tsx`: 이미지 참조 입력 필드 + 테이블 컬럼 추가
- `hooks/useEdgeWebSocket.ts` 신규: WebSocket 연결 훅, edge.heartbeat 이벤트 수신 시 edges 쿼리 invalidate, 5초 재연결
- `pages/DashboardPage.tsx`: useEdgeWebSocket 호출, 에지 노드 현황 테이블 추가

#### 통합 검증 결과

```
=== CREATE RELEASE ===
image_ref=harbor.central.internal/edge-app:v4.0.0 ✓

=== APPROVE ===
status=APPROVED ✓

=== DEPLOYMENT RESULT ===
phase=COMPLETED finished=2026-05-13T15:17:04 ✓

=== APPROVAL APPLIED ===
status=APPLIED version=2 ✓

=== WS UPGRADE ===
101 (WebSocket 업그레이드 성공) ✓
```

---

### Q: ui 에서 승인을 했는데, nats 수신이 되었는지. docker pull 은 제대로 되고 있는지은 어떻게 확인가능하지?

**A:**

**NATS 확인 방법:**
```bash
# brew로 nats CLI 설치
brew install nats-io/nats-tools/nats

# 스트림 목록
nats --server nats://localhost:4222 stream ls

# APPROVALS 스트림 메시지 내용 확인
nats --server nats://localhost:4222 stream get APPROVALS 1
```

확인 결과:
```
Stream APPROVALS: msgs=2, last=2m46s
Item: APPROVALS#1 on Subject approvals.APPROVED.398ed156-...
{"approval_id":"...","release_id":"...","edge_id":"...","status":"APPROVED","reason":"검증","image_ref":"harbor.central.internal/edge-app:v4.0.0"}
```

**docker pull 확인:**
edge-agent를 실행해야 pull 시도. Harbor 없으면 FAILED로 portal-api에 보고됨.
```bash
cd edge-agent
EDGE_ID="..." EDGE_NAME="..." NATS_URL="nats://localhost:4222" \
CENTRAL_API_URL="http://localhost:8081" HARBOR_URL="harbor.local" \
./bin/edge-agent
```

---

### Q: edge-agent 실행하면 다음 에러 발생한다 - subjects overlap with an existing stream

**A:**
원인: edge-agent의 `heartbeat.New()`가 `EDGE_HEARTBEAT` 스트림(subject: `edge.heartbeat.>`)을 새로 만들려는데, portal-api의 `EDGE_EVENTS` 스트림이 이미 `edge.>` 상위 패턴으로 같은 subject를 커버해서 충돌.

**수정: `internal/heartbeat/sender.go`**
스트림 생성 코드 제거. portal-api가 이미 관리하는 스트림에 publish만 함:
```go
// Before: CreateOrUpdateStream("EDGE_HEARTBEAT", ...) → 충돌
// After: 스트림 생성 없이 바로 Sender 반환
func New(...) (*Sender, error) {
    return &Sender{...}, nil  // 스트림 생성 제거
}
```

같은 문제가 `updater.go`에도 있었음. `RELEASES` 스트림도 portal-api가 관리하므로 생성 코드 제거:
```go
// Before: CreateOrUpdateStream("RELEASES", subjects: ["releases.published.>", "approvals.>"]) → 충돌
// After: consumer만 생성
// approval consumer도 RELEASES → APPROVALS 스트림으로 변경
appCons, err := u.js.CreateOrUpdateConsumer(ctx, "APPROVALS", ...)
```

---

### Q: edge agent 를 새로 띄웠는데 edge_nodes 테이블에는 데이터가 없는것이 맞나?

**A:**
아님. edge_nodes 테이블에 레코드가 **미리 있어야** heartbeat로 업데이트됨. edge-agent가 heartbeat를 NATS로 발행하면, portal-api의 SubscribeHeartbeats()가 수신해서 `SELECT * FROM edge_nodes WHERE id = <EDGE_ID>`로 조회 후 업데이트하는 구조.

확인 결과:
```sql
id                                   | name          | status | last_heartbeat_at
ee239f35-f585-4bfc-8a13-3900f6514ed5 | edge-busan-01 | UP     | 2026-05-13 06:40:51  ← edge-agent 실행 중
38506c95-fcc9-439f-a6d2-df3f18f85691 | edge-seoul-01 | UP     | null                 ← agent 미실행
```

현재 설계(방식 1): 중앙에서 에지를 사전 등록 → edge-agent가 heartbeat로 상태 업데이트.
보안상 더 안전 (승인되지 않은 에지가 자동 등록되는 것을 방지).

---

### Q: 에지를 중앙에서 등록하는 ui 가 없는 것 같은데?

**A:**
맞음. 구현 완료:

**백엔드 - `handler/edge.go`에 CreateEdge 추가:**
```go
func (h *EdgeHandler) CreateEdge(c *gin.Context) {
    // name, region (required), tenant_id (optional)
    // EdgeStatusUnknown으로 생성
    // repo.Save() → 201 반환
}
```
`cmd/server/main.go`에 `edges.POST("", edgeH.CreateEdge)` 라우트 추가.

**프론트엔드:**
- `hooks/useEdges.ts`: useCreateEdge() 훅 추가
- `pages/EdgesPage.tsx`: "+ 에지 등록" 버튼 + Dialog (name, region 입력)

**테스트:**
```bash
curl -X POST http://localhost:8080/api/v1/edges \
  -H "Content-Type: application/json" \
  -d '{"name":"edge-daejeon-01","region":"daejeon"}'
# → {"data":{"id":"35f6854d-...","status":"UNKNOWN",...}}
```

---

### Q: edge-agent 를 해당 EDGE_ID로 실행 하려면 어떻게 실행해야 하지

**A:**
```bash
cd /Users/lucirr/workspace/didimdol/edge-agent

# 빌드 (arm64)
GOARCH=arm64 GOOS=darwin go build -o bin/edge-agent ./cmd/agent/

# 실행 (portal에서 등록한 EDGE_ID 사용)
EDGE_ID="35f6854d-5e49-466d-ad68-92811a16a794" \
EDGE_NAME="edge-daejeon-01" \
EDGE_REGION="daejeon" \
NATS_URL="nats://localhost:4222" \
CENTRAL_API_URL="http://localhost:8081" \
HARBOR_URL="harbor.local" \
./bin/edge-agent
```

EDGE_ID가 portal-api의 edge_nodes.id와 일치해야 heartbeat가 DB에 반영됨.
불일치 시 "edge not found" 에러로 업데이트 무시.

---

### Q: EDGE_ID 는 실제 관리만 되는 값인가?

**A:**
네. `EDGE_ID = portal-api의 edge_nodes.id`와 정확히 일치해야 함.

| 용도 | 설명 |
|------|------|
| NATS subject | `edge.heartbeat.<EDGE_ID>` |
| heartbeat payload | `{"edge_id": "<EDGE_ID>"}` |
| portal-api DB 조회 | `WHERE id = <EDGE_ID>` |
| NATS consumer 이름 | `edge-<EDGE_ID>-approvals` |
| deployment-result | approval 조회 시 edge_id 매칭 |

실제 운영에서는 ArgoCD ApplicationSet에서 Helm values로 주입:
```yaml
env:
  EDGE_ID: "{{edgeId}}"  # list generator에서 관리
```
`deploy/argocd/edge-agent-appset.yaml`이 이 방식으로 구성되어 있음.

---

## 최종 구현 상태

| 컴포넌트 | 기능 | 상태 |
|---------|------|------|
| portal-api | Edge 등록/목록/조회 | ✅ |
| portal-api | Release CRUD (image_ref 포함) | ✅ |
| portal-api | Approval 생성/목록/조회/승인/거부 | ✅ |
| portal-api | 승인 후 NATS 이벤트 발행 | ✅ |
| portal-api | Harbor replication 트리거 (nil-safe) | ✅ |
| portal-api | /agent/v1/heartbeat 처리 | ✅ |
| portal-api | /agent/v1/deployment-result → deployment_records | ✅ |
| portal-api | WebSocket /api/v1/ws/edges | ✅ |
| portal-web | 에지 등록 UI | ✅ |
| portal-web | Release 등록 (image_ref 포함) | ✅ |
| portal-web | Approval 요청/승인/거부 | ✅ |
| portal-web | WebSocket 실시간 에지 상태 | ✅ |
| portal-web | 대시보드 에지 현황 테이블 | ✅ |
| edge-agent | NATS heartbeat 발신 (EDGE_EVENTS 스트림) | ✅ |
| edge-agent | NATS APPROVALS 구독 → docker pull | ✅ |
| edge-agent | kubectl CatalogPackage 적용 (non-fatal) | ✅ |
| edge-agent | deployment-result 보고 | ✅ |
| update-operator | CatalogPackage reconcile 구조 | ✅ (구조만) |
| update-operator | 실제 Helm 배포 | ❌ (exec 스텁) |
| Harbor replication | 실제 복제 실행 | ❌ (로컬 Harbor 없음) |
| ArgoCD GitOps | 실제 K8s 배포 | ❌ (로컬 K8s 없음) |

## 로컬 실행 명령 요약

```bash
# 1. 인프라
cd /Users/lucirr/workspace/didimdol/deploy/local && docker compose up -d

# 2. portal-api
cd /Users/lucirr/workspace/didimdol/portal-api
GOARCH=arm64 GOOS=darwin go build -o bin/portal-api ./cmd/server/...
DEV_MODE=true \
DATABASE_URL="postgres://edgedip:edgedip_secret@localhost:5432/edgedip?sslmode=disable" \
NATS_URL="nats://localhost:4222" \
./bin/portal-api

# 3. portal-web
cd /Users/lucirr/workspace/didimdol/portal-web && yarn dev

# 4. edge-agent (에지 등록 후)
cd /Users/lucirr/workspace/didimdol/edge-agent
GOARCH=arm64 GOOS=darwin go build -o bin/edge-agent ./cmd/agent/
EDGE_ID="<portal에서 등록한 ID>" \
EDGE_NAME="<에지명>" \
EDGE_REGION="<리전>" \
NATS_URL="nats://localhost:4222" \
CENTRAL_API_URL="http://localhost:8081" \
HARBOR_URL="harbor.local" \
./bin/edge-agent

# 5. NATS 모니터링
nats --server nats://localhost:4222 stream ls
nats --server nats://localhost:4222 stream get APPROVALS 1
```
