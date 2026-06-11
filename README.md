# Edge DIP — 엣지 배포 관리 플랫폼

엣지 노드에 대한 소프트웨어 배포를 중앙에서 안전하게 관리하는 플랫폼입니다.  
릴리스 승인, 배포 자동화, 원격 세션 제어를 통합 제공합니다.

## 시스템 아키텍처

```
=================================================================
                   중앙 (Central Control Plane)
=================================================================

  [외부 이미지]  (docker.io/vendor/app:v1.2.3)
       |
       | 관리자 수동 pull/retag/push
       v
  Harbor Registry  ←──────────────────────────────────┐
       |                                               │
       | 관리자: POST /api/v1/releases (DRAFT)          │
       v                                               │
  Portal API                                           │
       |                                               │
       | 관리자: Gitea UI → Actions → "Security Scan"  │
       |         (release_id + image_ref 입력)          │
       v                                               │
  +--------------------+  Gitea Actions workflow_dispatch
  |  보안점검 파이프라인  |
  |  ① Syft  → SBOM    |──attach──> Harbor (OCI referrer)
  |  ② Trivy → CVE     |──PATCH /releases/:id/cve-report──> Portal API (SCANNED)
  |  ③ Critical? → 중단 |  (Critical CVE 있으면 파이프라인 실패, 서명/발행 불가)
  |  ④ Cosign → 서명    |──attach──────────────────────────┘
  |                    |──PATCH /releases/:id/sign ──────> Portal API (SIGNED)
  +--------------------+
                            |
        +-------------------+-------------------+
        |                                       |
        v                                       v
  +-----------+                          +-----------+
  | 통합 포털  |  <-- Keycloak (OIDC) --> | Portal    |
  | React 18  |  JWT role 기반 인가       | API       |
  |           |  (realm_roles 클레임)     | Go/Gin    |
  |     WS    | <======================== | :8080     |
  | (실시간   |  WebSocket /ws/edges      |           |
  |  edge 상태)|                          | Agent API |
  +-----------+                          | :8081(mTLS|
                                         +-----+-----+
                                               |
                                               | PostgreSQL 16
                                               v
                                         +-----------+
                                         | PostgreSQL |
                                         | 16         |
                                         +-----------+

  Portal API  ──publish──>  NATS JetStream (중앙 Cluster)
  (릴리스 발행)              ├─ RELEASES 스트림
                            │   releases.published.<id>
  (승인 완료)                ├─ APPROVALS 스트림
                            │   approvals.APPROVED.<id>
                            │
                            │  ▲ NATS LeafNode 연결 (DMZ 환경)
                            │  │  에지 NATS가 중앙 NATS에 outbound로 붙음
                            │  │  NKey/JWT Credentials 또는 mTLS 인증
                            │  │
                       [DMZ Firewall]
                            │
========================================================================
        에지 (Edge Plane, N개 사이트)   ※ 모든 통신은 에지→중앙 outbound만
========================================================================
                            │
           ┌────────────────┘
           │  NATS outbound 연결 (에지가 중앙 NATS LeafNode 허브에 접속)
           │  일반 환경: nats://central:4222
           │  DMZ 환경:  tls://central:4222 + NATS_CREDS / NATS_TLS_CA|CERT|KEY
           │
           │  subscribe: releases.published.>  ──────────────────┐
           │  subscribe: approvals.APPROVED.>  ──────────────┐   │
           v                                                 │   │
  +-----------+    +-------------------+                     │   │
  | Edge      |    | K3s/RKE2          |                     │   │
  | Agent     |--->| + Update Operator |                     │   │
  | (Go)      |    | (CatalogPackage   |                     │   │
  |           |    |  CRD reconcile)   |  ①릴리스 알림 수신 ──┘   │
  +-----+-----+    +--------+----------+  ②승인 완료 수신 ────────┘
        |                   |
        | HTTP outbound      | kubectl apply
        | (에지→중앙)         | CatalogPackage
        |                   v
        |           +-----------+
        |           | Local     |
        |           | Harbor    |
        |           | Mirror    |
        |           | (image    |
        |           |  pull)    |
        |           +-----------+
        |
        | POST /agent/v1/heartbeat           → 하트비트 (DB 업데이트 + WebSocket broadcast)
        | POST /agent/v1/approval-requests  → 승인 요청 자동 생성
        | POST /agent/v1/download-progress  → 다운로드 진행률 보고
        | POST /agent/v1/deployment-result  → 배포 결과 보고
        v
  +-----------+
  | Central   |
  | Portal API|
  | :8081     |
  | (mTLS)    |
  +-----------+
```

**핵심 통신 원칙**:
- 에지는 항상 **outbound 클라이언트** — 중앙이 에지에 직접 inbound 접속하는 경로 없음
- NATS: 에지가 중앙 NATS 서버에 outbound 연결 후 subscribe (pull 방식)
- HTTP: 에지가 중앙 Agent API(`:8081`, mTLS)에 outbound POST

**NATS JetStream 스트림** (중앙→에지 단방향):

| 스트림 | Subject | 용도 |
|--------|---------|------|
| `RELEASES` | `releases.published.<id>` | 릴리스 발행 알림 |
| `APPROVALS` | `approvals.APPROVED.<id>` | 승인 완료 알림 |

**에지→중앙 HTTP 엔드포인트** (Agent API `:8081`, mTLS):

| 엔드포인트 | 용도 |
|-----------|------|
| `POST /agent/v1/heartbeat` | 하트비트 — DB 갱신 + WebSocket broadcast |
| `POST /agent/v1/approval-requests` | 릴리스 수신 후 승인 요청 자동 생성 |
| `POST /agent/v1/download-progress` | 이미지 다운로드 진행률 보고 |
| `POST /agent/v1/deployment-result` | 배포 결과 보고 (COMPLETED/FAILED/ROLLED_BACK) |

**DMZ 터널링 (NATS LeafNode)**:

에지가 방화벽 뒤(DMZ) 또는 격리망에 있는 경우 NATS LeafNode를 사용합니다.  
- **인프라 구성**: 중앙 NATS 서버에서 LeafNode Hub 포트(기본 7422) 활성화  
- **코드 지원**: edge-agent는 NATS_CREDS 또는 NATS_TLS_* 환경변수로 인증 방식 선택  
- **스트림 투명성**: LeafNode 연결 후에도 동일한 Subject/Stream 이름 그대로 사용 (코드 변경 없음)

| 환경변수 | 용도 |
|----------|------|
| `NATS_CREDS` | NKey/JWT credentials 파일 경로 (권장) |
| `NATS_TLS_CA` | CA 인증서 경로 (직접 mTLS 시) |
| `NATS_TLS_CERT` | 클라이언트 인증서 경로 |
| `NATS_TLS_KEY` | 클라이언트 키 경로 |

**승인 요청 생성 경로 (두 가지)**:
1. **UI 수동 생성** — 관리자가 Portal Web에서 릴리스 + 에지 노드 선택 후 `POST /api/v1/approvals` 직접 호출
2. **edge-agent 자동 생성** — `releases.published.>` 수신 시 `POST /agent/v1/approval-requests` 자동 호출

**배포 프로세스 (일반, is_urgent=false)**:
```
[중앙] CI 파이프라인: 이미지 Harbor push
[중앙] CI 파이프라인: 보안점검 실행 (Trivy CVE 스캔 + Cosign 서명)
[중앙] CI 파이프라인: 점검 통과 시 Draft 릴리즈 등록 → PENDING_APPROVAL 상태로 발행 승인 요청
[중앙] central-operator: Portal UI에서 POST /api/v1/releases/:id/approve-publish
  → 릴리즈 PUBLISHED 전환 + NATS releases.published.<id> 발행
[에지] edge-agent: releases.published.> 수신
[에지] edge-agent: POST /agent/v1/approval-requests → 중앙에 자동 승인 요청 (PENDING)
[중앙] edge-admin: Portal UI에서 POST /api/v1/approvals/:id/approve
  ※ edge-admin은 자신의 테넌트(현장) 소속 에지만 승인 가능
  → NATS approvals.APPROVED.<id> 발행
[에지] edge-agent: approvals.APPROVED.> 수신
[에지] edge-agent: Local Harbor에서 image pull (docker/nerdctl)
[에지] kubectl apply CatalogPackage CRD → Update Operator reconcile:
        Downloading → Applying (Helm install/upgrade)
          → HealthCheck (pod readiness polling, timeout: 5m)
            ┌─ 성공 → phase: Ready
            └─ 실패 + autoRollback=true → Helm rollback → phase: RolledBack
[에지] CatalogWatcher (15초 간격 polling):
        Ready      → POST /agent/v1/deployment-result (phase: COMPLETED)
        Failed     → POST /agent/v1/deployment-result (phase: FAILED)
        RolledBack → POST /agent/v1/deployment-result (phase: ROLLED_BACK)
```

**배포 프로세스 (긴급 패치, is_urgent=true)**:
```
[중앙] CI 파이프라인: 이미지 Harbor push
[중앙] CI 파이프라인: 보안점검 실행 (Trivy CVE 스캔 + Cosign 서명)
[중앙] CI 파이프라인: 점검 통과 시 Draft 릴리즈 등록 → PENDING_APPROVAL 상태로 발행 승인 요청
[중앙] central-operator: POST /api/v1/releases/:id/approve-publish (is_urgent=true 포함)
  → 릴리즈 PUBLISHED 전환 + NATS releases.published.<id> 발행
[에지] edge-agent: releases.published.> 수신
[에지] edge-agent: POST /agent/v1/approval-requests
  → portal-api: is_urgent 확인 → 즉시 APPROVED 처리 + NATS approvals.APPROVED.<id> 발행
[에지] edge-agent: approvals.APPROVED.> 수신 (edge-admin 개입 없음)
[에지] Local Harbor에서 image pull → kubectl apply CatalogPackage CRD
[에지] Update Operator reconcile (동일: HealthCheck + AutoRollback 포함)
[에지] CatalogWatcher → POST /agent/v1/deployment-result
```

## 컴포넌트

| 디렉토리 | 언어/프레임워크 | 역할 |
|---|---|---|
| `portal-api/` | Go 1.22, Gin, sqlx | 중앙 제어 REST API 서버 (`:8080`) + Agent API (`:8081`, mTLS) |
| `portal-web/` | React 18, TypeScript, Vite | 관리자 웹 포털 |
| `edge-agent/` | Go 1.22, NATS JetStream | 엣지 노드 에이전트 |
| `update-operator/` | Go 1.22 | K8s 기반 배포 오퍼레이터 |
| `deploy/local/` | Docker Compose | 로컬 개발 인프라 |

## 핵심 기능

### 릴리스 관리
외부 이미지를 Harbor에 등록 후 보안 점검과 발행 승인을 거쳐 배포합니다.  
상태 흐름: `DRAFT → SCANNED → SIGNED → PENDING_APPROVAL → PUBLISHED`

| 단계 | 트리거 | 설명 |
|------|--------|------|
| `DRAFT` | CI 파이프라인 `POST /api/v1/releases` | Harbor image_ref 등록 |
| `SCANNED` | Gitea Actions (Syft + Trivy) | SBOM 생성, CVE 스캔 결과 저장. Critical CVE 있으면 이후 단계 차단 |
| `SIGNED` | Gitea Actions (Cosign) | 조직 내부 키로 서명 |
| `PENDING_APPROVAL` | CI 파이프라인 `POST /api/v1/releases/:id/request-publish` | 발행 승인 대기. CI(`pipeline-bot`)가 요청하며, 이 시점에는 에지에 아무 이벤트도 발행되지 않음 |
| `PUBLISHED` | central-operator `POST /api/v1/releases/:id/approve-publish` | 담당자가 최종 승인. NATS 이벤트 발행 → 에지 배포 시작 |

**보안 점검 실행 방법**: Gitea UI → Actions 탭 → "Security Scan" → Run workflow  
입력값: `release_id` (Portal UUID), `image_ref` (Harbor 주소)

파이프라인: `.gitea/workflows/security-scan.yml`
- ① **Syft**: SBOM(SPDX JSON) 생성 → Harbor OCI referrer로 attach
- ② **Trivy**: CVE 스캔 → `PATCH /api/v1/releases/:id/cve-report` → `SCANNED`
- ③ **Critical CVE 차단**: Critical > 0 이면 파이프라인 실패, 이후 단계 진입 불가
- ④ **Cosign**: 내부 키로 이미지 서명 → `PATCH /api/v1/releases/:id/sign` → `SIGNED`
- ⑤ **발행 승인 요청**: `POST /api/v1/releases/:id/request-publish` → `PENDING_APPROVAL`

- **최종 발행**: `PENDING_APPROVAL` 상태에서만 가능. `central-operator` 역할 전용
- **긴급 패치**: `is_urgent=true` + `PUBLISHED` 상태 릴리스에 한해 에지 관리자 승인 없이 자동 배포
- **NATS 이벤트**: `approve-publish` 시 `releases.published.<id>` 이벤트 → edge-agent 승인 요청 자동 생성

### 승인 워크플로
엣지 노드별 배포 승인 요청을 생성하고 `PENDING → APPROVED → APPLIED` 흐름으로 처리합니다.  
승인 규칙(역할 체크, CVE gate)은 Portal API 핸들러 코드에서 직접 처리합니다.

승인 상태: `PENDING | APPROVED | REJECTED | DEFERRED | APPLIED | ROLLED_BACK | EXPIRED`

### 원격 세션
중앙 운영자가 엣지 노드에 원격으로 접속할 수 있는 세션을 최대 30분 TTL로 제어합니다.  
세션 생성 시 사유(reason)가 필수이며, `auditor` 역할은 세션 녹화 기록만 조회 가능합니다.

### 엣지 에이전트
각 엣지 노드에서 실행되는 경량 Go 데몬으로, NATS JetStream과 HTTP를 통해 중앙과 통신합니다.
- NATS subscribe: 릴리스 알림 및 승인 완료 이벤트 수신
- NATS publish: 주기적 하트비트 전송 (기본 10초)
- HTTP POST: 승인 요청 생성, 배포 진행률/결과 보고

### 인증 및 인가
- **인증**: Keycloak 24 OIDC — JWT Bearer 토큰
- **인가**: JWT `realm_access.roles` 클레임 기반 역할 체크 (Portal API 미들웨어 + 핸들러에서 처리)
  - `central-operator`: 릴리스 생성/발행, 원격 세션 생성 (승인은 불가 — 이중 안전장치)
  - `edge-admin`: **자신의 테넌트(현장)에 속한 에지의 최종 승인/거부만 가능**
  - `auditor`: 읽기 전용, 세션 녹화 조회
  - Keycloak JWT 커스텀 클레임: `tenant_id` (edge_nodes.tenant_id와 매핑)

### 실시간 에지 상태
Portal Web은 WebSocket(`/api/v1/ws/edges`)으로 에지 노드 상태를 실시간 수신합니다.  
edge-agent 하트비트 → NATS → Portal API → WebSocket broadcast → 브라우저 순으로 전달됩니다.

## 데이터 모델

```
edge_nodes                  releases                    approval_requests
  id (UUID)                   id (UUID)                   id (UUID)
  name                        package_name                release_id → releases
  region                      version                     edge_id → edge_nodes
  tenant_id                   artifact_digest             status
  status (UP/DOWN/DEGRADED)   image_ref                     PENDING/APPROVED/REJECTED
  last_heartbeat_at           sbom_uri                      DEFERRED/APPLIED
  agent_version               cve_report (JSONB)            ROLLED_BACK/EXPIRED
  k8s_version                 signature / signed_by       requested_by / decision_by
  capabilities (JSONB)        status                      idempotency_key
  labels (JSONB)                DRAFT/SCANNED/SIGNED      version (optimistic lock)
                                PENDING_APPROVAL          scheduled_at / deferred_until
                                PUBLISHED/DEPRECATED

deployment_records          remote_sessions
  id (UUID)                   id (UUID)
  approval_id                 edge_id → edge_nodes
  edge_id / release_id        operator_id
  phase                       reason (필수)
    DOWNLOADING/APPLYING      ttl_seconds (최대 1800)
    HEALTHCHECK/COMPLETED     status
    FAILED/ROLLED_BACK          PENDING_APPROVAL/ACTIVE
  progress_pct (0-100)          EXPIRED/TERMINATED
  error_code / error_message
```

## 기술 스택

- **백엔드**: Go 1.22, Gin, sqlx, NATS JetStream, Viper, Zap
- **프론트엔드**: React 18, TypeScript, Vite, Tailwind CSS, TanStack Query, Axios
- **DB**: PostgreSQL 16
- **인증/인가**: Keycloak 24 (OIDC), JWT realm_roles 기반 역할 체크
- **배포**: Docker Compose (로컬), Helm (운영)

## 로컬 개발 환경

### 사전 요구사항

- Docker & Docker Compose
- Go 1.22+
- Node.js 18+ / Yarn

### 인프라 실행

```bash
cd deploy/local
cp .env.example .env
docker compose up -d
```

| 서비스 | 주소 |
|---|---|
| PostgreSQL | `localhost:5432` |
| NATS | `localhost:4222` (모니터링: `localhost:8222`) |
| Keycloak | `http://localhost:8180` (admin / admin) |

로컬 Keycloak은 `edgedip` realm과 `portal-web` 클라이언트를 import합니다. 포털 로그인 테스트 계정은 `portal-admin / portal-admin`입니다.

### Portal API 실행

```bash
cd portal-api
make run
# → http://localhost:8080  (Public API, JWT 인증)
# → http://localhost:8081  (Agent API, mTLS)
```

### Portal Web 실행

```bash
cd portal-web
yarn install
yarn dev
# → http://localhost:5173
```

포털 로그인은 `http://localhost:5173/login`에서 Keycloak로 리다이렉트됩니다.

### Edge Agent 실행

```bash
export EDGE_ID=<uuid>
export EDGE_NAME=<노드명>
export NATS_URL=nats://localhost:4222

cd edge-agent
make run
```

#### Edge Agent 환경변수

| 변수 | 기본값 | 설명 |
|---|---|---|
| `EDGE_ID` | (필수) | 엣지 노드 UUID |
| `EDGE_NAME` | (필수) | 엣지 노드 이름 |
| `EDGE_REGION` | `default` | 리전 |
| `NATS_URL` | `nats://localhost:4222` | NATS 서버 주소 (DMZ: `tls://...`) |
| `NATS_CREDS` | — | NKey/JWT credentials 파일 (DMZ LeafNode 인증, 권장) |
| `NATS_TLS_CA` | — | NATS TLS CA 인증서 경로 (직접 mTLS 시) |
| `NATS_TLS_CERT` | — | NATS TLS 클라이언트 인증서 경로 |
| `NATS_TLS_KEY` | — | NATS TLS 클라이언트 키 경로 |
| `CENTRAL_API_URL` | `http://localhost:8081` | 중앙 Agent API 주소 |
| `HARBOR_URL` | `https://harbor.local` | Harbor 레지스트리 주소 |
| `HEARTBEAT_INTERVAL` | `10s` | 하트비트 전송 주기 |
| `AGENT_TLS_ENABLED` | `false` | mTLS 활성화 여부 |
| `AGENT_TLS_CA` | — | CA 인증서 경로 (TLS 활성 시 필수) |
| `AGENT_TLS_CERT` | — | 클라이언트 인증서 경로 (TLS 활성 시 필수) |
| `AGENT_TLS_KEY` | — | 클라이언트 키 경로 (TLS 활성 시 필수) |

## API 엔드포인트

### Public API (`:8080`, JWT 인증 필요)

| Method | Path | 설명 |
|--------|------|------|
| GET | `/health` | 헬스 체크 |
| POST/GET | `/api/v1/edges` | 에지 노드 등록/목록 |
| GET | `/api/v1/edges/:id` | 에지 노드 상세 |
| POST | `/api/v1/releases` | 릴리스 등록 |
| GET | `/api/v1/releases` | 릴리스 목록 |
| PATCH | `/api/v1/releases/:id/cve-report` | CVE 리포트 업데이트 (SCANNED) |
| PATCH | `/api/v1/releases/:id/sign` | 서명 정보 등록 (SIGNED) |
| POST | `/api/v1/releases/:id/request-publish` | 발행 승인 요청 (PENDING_APPROVAL) — pipeline-bot/central-operator |
| POST | `/api/v1/releases/:id/approve-publish` | 발행 최종 승인 (PUBLISHED + NATS 발행) — central-operator 전용 |
| POST/GET | `/api/v1/approvals` | 승인 요청 생성/목록 |
| POST | `/api/v1/approvals/:id/approve` | 승인 |
| POST | `/api/v1/approvals/:id/reject` | 거절 |
| POST | `/api/v1/approvals/:id/defer` | 연기 |
| GET | `/api/v1/approvals/:id/deployments` | 배포 진행 기록 조회 (폴링용) |
| GET | `/api/v1/ws/edges` | WebSocket — 실시간 에지 이벤트 |

### Agent API (`:8081`, mTLS)

| Method | Path | 설명 |
|--------|------|------|
| POST | `/agent/v1/approval-requests` | 승인 요청 자동 생성 |
| POST | `/agent/v1/download-progress` | 다운로드 진행률 보고 |
| POST | `/agent/v1/deployment-result` | 배포 결과 보고 |

## 데이터베이스 마이그레이션

마이그레이션 파일은 `portal-api/migrations/`에 위치하며, Docker Compose 실행 시 자동 적용됩니다.

```bash
# 수동 실행
docker compose exec postgres psql -U edgedip -d edgedip
```

## 디렉토리 구조

```
didimdol/
├── edge-agent/          # 엣지 노드 에이전트 (Go)
│   ├── cmd/agent/       # 진입점
│   └── internal/
│       ├── config/      # 환경변수 설정
│       ├── heartbeat/   # 하트비트 송신 (NATS publish)
│       ├── reporter/    # 배포 결과 보고 (HTTP POST)
│       ├── collector/   # 메트릭 수집
│       └── updater/     # 릴리스/승인 이벤트 수신 및 배포 처리
├── portal-api/          # 중앙 제어 API (Go)
│   ├── internal/
│   │   ├── handler/     # HTTP 핸들러 (edge, release, approval, session, agent, ws)
│   │   ├── service/     # NATS, Harbor 서비스
│   │   ├── repository/  # PostgreSQL 리포지토리
│   │   ├── domain/      # 도메인 모델
│   │   ├── middleware/  # 인증(JWT), 감사 로그
│   │   └── hub/         # WebSocket 브로드캐스트 허브
│   └── migrations/      # SQL 마이그레이션
├── portal-web/          # 관리자 웹 (React)
│   └── src/
├── update-operator/     # K8s 배포 오퍼레이터 (Go)
├── deploy/
│   ├── local/           # Docker Compose 로컬 환경
│   └── helm/            # Helm 배포 차트
└── pipelines/           # Gitea Actions CI 파이프라인
```
