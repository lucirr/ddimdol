# 승인 워크플로우 구현 세션 기록 (2026-05-13)

## 개요

portal-api Go 백엔드 + portal-web React 프론트엔드의 승인 워크플로우 완성 및 런타임 검증 세션.

---

## 주요 작업 내역

### 1. portal-api 빌드 및 실행 환경 정상화

**문제**: Apple Silicon(arm64) 환경에서 amd64 빌드된 바이너리 실행 시 `dyld LC_UUID` 오류  
**해결**: `GOARCH=arm64 GOOS=darwin go build -o bin/portal-api ./cmd/server/...`

**문제**: 이전 프로세스가 포트 8080을 점유  
**해결**: `lsof -ti :8080 | xargs kill -9`

**실행 명령**:
```bash
cd /Users/lucirr/workspace/didimdol/portal-api
GOARCH=arm64 GOOS=darwin go build -o bin/portal-api ./cmd/server/...
DEV_MODE=true \
  DATABASE_URL="postgres://edgedip:edgedip_secret@localhost:5432/edgedip?sslmode=disable" \
  NATS_URL="nats://localhost:4222" \
  ./bin/portal-api
```

---

### 2. decision_reason 미저장 버그 수정

**문제**: 승인/거부 시 `reason`이 응답에는 포함되지만 DB에는 저장되지 않음  
(GET으로 다시 조회하면 `decision_reason`이 빈 문자열)

**원인**: `UpdateStatus` SQL이 `status`, `version`, `updated_at`만 업데이트하고 `decision_reason`은 누락

**수정 파일들**:

#### `internal/repository/interfaces.go`
```go
// 변경 전
UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ApprovalStatus, version int) error

// 변경 후
UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ApprovalStatus, reason string, version int) error
```

#### `internal/repository/postgres/approval.go`
```go
// 변경 전
func (r *ApprovalRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ApprovalStatus, version int) error {
    res, err := r.db.ExecContext(ctx, `
        UPDATE approval_requests
        SET status = $1, version = version + 1, updated_at = NOW()
        WHERE id = $2 AND version = $3
    `, status, id, version)

// 변경 후
func (r *ApprovalRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ApprovalStatus, reason string, version int) error {
    res, err := r.db.ExecContext(ctx, `
        UPDATE approval_requests
        SET status = $1, decision_reason = $2, version = version + 1, updated_at = NOW()
        WHERE id = $3 AND version = $4
    `, status, reason, id, version)
```

#### `internal/handler/approval.go` - Approve
```go
// 변경 전
if err := h.repo.UpdateStatus(c.Request.Context(), id, domain.ApprovalStatusApproved, approval.Version); err != nil {

// 변경 후
if err := h.repo.UpdateStatus(c.Request.Context(), id, domain.ApprovalStatusApproved, req.Reason, approval.Version); err != nil {
```

#### `internal/handler/approval.go` - Reject
```go
// reason을 approval에 미리 세팅하고 UpdateStatus에도 전달
approval.DecisionReason = req.Reason
if err := h.repo.UpdateStatus(c.Request.Context(), id, domain.ApprovalStatusRejected, req.Reason, approval.Version); err != nil {
```

#### `internal/service/approval.go`
```go
// Approve
return s.approvals.UpdateStatus(ctx, id, domain.ApprovalStatusApproved, reason, approval.Version)

// Reject
return s.approvals.UpdateStatus(ctx, id, domain.ApprovalStatusRejected, reason, approval.Version)

// Defer
return s.approvals.UpdateStatus(ctx, id, domain.ApprovalStatusDeferred, "", approval.Version)
```

---

### 3. 통합 테스트 결과

```bash
# 1. Health check
GET /health → {"status":"ok"}

# 2. Edges (DB에 2건 조회됨)
GET /api/v1/edges → [edge-busan-01(DOWN), edge-seoul-01(UP)]

# 3. Release 생성
POST /api/v1/releases
  body: {"package_name":"myapp","version":"2.0.0","artifact_digest":"sha256:def456"}
  → 201, id: 82ec44af-...

# 4. Approval 요청 생성
POST /api/v1/approvals
  body: {"release_id":"82ec44af-...","edge_id":"ee239f35-..."}
  → status: PENDING, version: 0

# 5. 승인
POST /api/v1/approvals/{id}/approve
  body: {"reason":"테스트 승인 사유입니다"}
  → status: APPROVED, decision_reason: "테스트 승인 사유입니다"

# 6. 재조회로 DB 저장 확인
GET /api/v1/approvals/{id}
  → status: APPROVED, decision_reason: "테스트 승인 사유입니다", version: 1 ✓
```

---

### 4. 승인 이후 미구현 프로세스 (현황 파악)

현재 승인 후 프로세스:
```
승인 클릭
   │
   ▼
approval_requests.status = APPROVED  ← 여기까지만 구현됨
   │
   ▼  [미구현] NATS "APPROVALS" 스트림 이벤트 발행
   ▼  [미구현] edge-agent NATS 구독 → 패키지 다운로드
   ▼  [미구현] update-operator CatalogPackage CRD 생성/업데이트
   ▼  [미구현] Kubernetes 배포
   ▼  [미구현] edge-agent → /agent/v1/deployment-result 보고
   ▼  [미구현] deployment_records 테이블 업데이트
```

다음 구현 우선순위:
1. portal-api: 승인 후 `NatsService.PublishApprovalEvent` 호출
2. portal-api: `/agent/v1/deployment-result` → `deployment_records` 저장
3. edge-agent: NATS 구독 + 배포 결과 보고
4. update-operator: CatalogPackage reconcile 루프

---

## 인프라 실행 상태

```bash
# docker-compose로 인프라 기동
cd /Users/lucirr/workspace/didimdol/deploy/local
docker compose up -d

# 서비스 포트
# PostgreSQL: localhost:5432 (user=edgedip, password=edgedip_secret, db=edgedip)
# Redis:      localhost:6379
# NATS:       localhost:4222 (JetStream 활성화)
# Keycloak:   localhost:8180

# portal-api: localhost:8080 (public API), localhost:8081 (mTLS agent)
# portal-web: localhost:5173 (yarn dev)
```

---

## 현재 구현 완료 범위

| 컴포넌트 | 기능 | 상태 |
|---------|------|------|
| portal-api | Edge 목록 조회 | ✅ |
| portal-api | Release CRUD | ✅ |
| portal-api | Approval 생성/목록/조회 | ✅ |
| portal-api | Approve/Reject (reason 저장, 낙관적 잠금) | ✅ |
| portal-api | DEV_MODE 인증 우회 | ✅ |
| portal-api | NATS 이벤트 발행 (승인 후) | ❌ |
| portal-api | deployment-result 수신 | ❌ |
| portal-web | Dashboard | ✅ |
| portal-web | Edge 목록 | ✅ |
| portal-web | Release 목록 + 생성 폼 | ✅ |
| portal-web | Approval 목록 + 요청 폼 | ✅ |
| portal-web | Approve/Reject 다이얼로그 (reason 입력) | ✅ |
| edge-agent | NATS heartbeat 발신 | ✅ (구조만) |
| edge-agent | 배포 이벤트 수신 + 결과 보고 | ❌ |
| update-operator | CatalogPackage reconcile | ✅ (구조만) |
