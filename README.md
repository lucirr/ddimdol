# Edge DIP — 엣지 배포 관리 플랫폼

엣지 노드에 대한 소프트웨어 배포를 중앙에서 안전하게 관리하는 플랫폼입니다.  
릴리스 승인, 배포 자동화, 원격 세션 제어, OPA 기반 정책 적용을 통합 제공합니다.

## 시스템 아키텍처

```
=================================================================
                   중앙 (Central Control Plane)
=================================================================

  [개발자] --> Gitea --> Gitea Actions
                            |
                            v
            +---------------+--------------+
            |     보안점검 파이프라인        |
            |  Syft(SBOM) -> Trivy/Grype   |
            |   -> Cosign Sign -> Harbor   |
            +---------------+--------------+
                            |
                            v
                      Harbor Registry
                            |
                            v
                  +---------+----------+
                  |   ArgoCD (중앙)     |  --> Helm Chart Repo (Gitea)
                  |   ApplicationSet   |
                  +---------+----------+
                            |
        +-------------------+-------------------+
        |                                       |
        v                                       v
  +-----------+                          +-----------+
  | 통합 포털  |  <-- Keycloak (OIDC) --> | OPA 정책  |
  | React 18  |                          | (Rego)   |
  +-----+-----+                          +-----------+
        |
        v
  +-----------+    +-----------+
  | Portal    |--->| PostgreSQL|
  | API       |    | 16        |
  | Go/Gin    |    +-----------+
  +-----+-----+
        |
        | releases.published.>   (릴리스 노티)
        | approvals.APPROVED.*   (승인 완료)
        | edge.heartbeat.<id>    (하트비트 수신)
        v
  +-----------+
  | NATS      |  <==== outbound only (에지 → 중앙) ====
  | JetStream |
  +-----+-----+
        |
========|================================================
        |   에지 (Edge Plane, N개 사이트)
========|================================================
        |
        v
  +-----------+    +-------------------+
  | Edge      |    | K3s/RKE2          |
  | Agent     |--->| + Update Operator |
  | (Go)      |    | (CatalogPackage   |
  |           |    |  CRD reconcile)   |
  +-----+-----+    +--------+----------+
        |                   |
        | 승인 요청 자동 생성  | kubectl apply
        | POST /agent/v1/   | CatalogPackage
        | approval-requests |
        v                   v
  +-----------+    +-----------+
  | Central   |    | Local     |
  | Portal API|    | Harbor    |
  +-----------+    | Mirror    |
                   | (image    |
                   |  pull)    |
                   +-----------+
```

**핵심 통신 패턴**:
- 에지는 항상 **outbound 클라이언트** — NATS subscribe, Harbor pull, HTTP POST
- 중앙은 에지에 직접 inbound 접속 불가
- NATS JetStream 스트림:
  - `RELEASES` 스트림 → `releases.published.>` (릴리스 알림)
  - `APPROVALS` 스트림 → `approvals.APPROVED.*` (승인 완료)
  - `EDGE_EVENTS` 스트림 ← `edge.heartbeat.<edge-id>` (하트비트)

**승인 요청 생성 경로 (두 가지)**:
1. **UI 수동 생성** — 관리자가 Portal Web에서 릴리스 + 에지 노드 선택 후 `POST /api/v1/approvals` 직접 호출
2. **edge-agent 자동 생성** — `releases.published.>` 수신 시 `POST /agent/v1/approval-requests` 자동 호출

**배포 흐름**: 릴리스 발행 → (UI 또는 agent가) 승인 요청 생성 → 관리자 승인 → `approvals.APPROVED.*` 이벤트 → edge-agent가 Harbor pull → `CatalogPackage` CRD apply → Update Operator reconcile

## 컴포넌트

| 디렉토리 | 언어/프레임워크 | 역할 |
|---|---|---|
| `portal-api/` | Go 1.22, Gin, sqlx | 중앙 제어 REST API 서버 |
| `portal-web/` | React 18, TypeScript, Vite | 관리자 웹 포털 |
| `edge-agent/` | Go 1.22, NATS JetStream | 엣지 노드 에이전트 |
| `update-operator/` | Go 1.22 | K8s 기반 배포 오퍼레이터 |
| `policies/` | OPA Rego | 승인·세션 정책 |
| `deploy/local/` | Docker Compose | 로컬 개발 인프라 |

## 핵심 기능

### 릴리스 관리
소프트웨어 패키지를 `DRAFT → SCANNED → SIGNED → PUBLISHED` 단계로 관리합니다.  
각 릴리스는 아티팩트 다이제스트, SBOM URI, CVE 리포트, 서명 정보를 포함합니다.

- **등록**: Portal UI에서 패키지명/버전/이미지 정보 입력 → `DRAFT` 상태로 생성
- **발행**: `SCANNED` 또는 `SIGNED` 상태의 릴리스에 한해 UI에서 발행 버튼 클릭 → `PUBLISHED` 전환 (Critical CVE가 있으면 발행 거부)
- **NATS 이벤트**: 발행 시 `releases.published.<id>` 이벤트 발행 → edge-agent가 수신하여 승인 요청 자동 생성

### 승인 워크플로
엣지 노드별 배포 승인 요청을 생성하고 `PENDING → APPROVED → APPLIED` 흐름으로 처리합니다.  
OPA 정책(`policies/approval.rego`, `policies/release.rego`)으로 승인 규칙을 코드로 관리합니다.

### 원격 세션
중앙 운영자가 엣지 노드에 원격으로 접속할 수 있는 세션을 최대 30분 TTL로 제어합니다.  
세션 생성 시 사유(reason)가 필수이며, `auditor` 역할은 세션 녹화 기록만 조회 가능합니다.

### 엣지 에이전트
각 엣지 노드에서 실행되는 경량 Go 데몬으로, NATS JetStream을 통해 중앙과 통신합니다.
- 주기적 하트비트 전송 (기본 10초)
- 메트릭 수집 및 보고
- 배포 명령 수신 및 자동 업데이트

## 데이터 모델

```
edge_nodes        릴리스 정보        승인 요청
  id (UUID)         id (UUID)         id (UUID)
  name              package_name      release_id → releases
  region            version           edge_id → edge_nodes
  tenant_id         artifact_digest   status (PENDING/APPROVED/...)
  status            status (DRAFT/…)  requested_by / decision_by
  last_heartbeat_at signed_by

remote_sessions
  id (UUID)
  edge_id → edge_nodes
  operator_id
  reason (필수)
  ttl_seconds (최대 1800)
  status (PENDING_APPROVAL/ACTIVE/EXPIRED/TERMINATED)
```

## 기술 스택

- **백엔드**: Go 1.22, Gin, sqlx, NATS JetStream, Viper, Zap
- **프론트엔드**: React 18, TypeScript, Vite, Tailwind CSS, TanStack Query, Axios
- **DB**: PostgreSQL 16
- **인증**: Keycloak 24 (OIDC)
- **정책 엔진**: OPA (Open Policy Agent) Rego
- **배포**: ArgoCD (ApplicationSet), Docker Compose (로컬)

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

### Portal API 실행

```bash
cd portal-api
make run
# → http://localhost:8080
```

### Portal Web 실행

```bash
cd portal-web
yarn install
yarn dev
# → http://localhost:5173
```

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
| `NATS_URL` | `nats://localhost:4222` | NATS 서버 주소 |
| `CENTRAL_API_URL` | `http://localhost:8080` | 중앙 API 주소 |
| `HARBOR_URL` | `https://harbor.local` | Harbor 레지스트리 주소 |
| `HEARTBEAT_INTERVAL` | `10s` | 하트비트 전송 주기 |

## 정책 (OPA)

`policies/` 디렉토리에 Rego 파일로 정책을 정의합니다.

- `session.rego` — `central-operator`만 세션 생성/활성화/종료 가능, TTL 최대 1800초
- `approval.rego` — 배포 승인 규칙
- `release.rego` — 릴리스 게시 규칙

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
│       ├── heartbeat/   # 하트비트 송신
│       ├── reporter/    # 메트릭 보고
│       ├── collector/   # 메트릭 수집
│       └── updater/     # 배포 업데이트 처리
├── portal-api/          # 중앙 제어 API (Go)
│   ├── internal/
│   │   └── middleware/  # 감사 로그 등
│   └── migrations/      # SQL 마이그레이션
├── portal-web/          # 관리자 웹 (React)
│   └── src/
├── update-operator/     # K8s 배포 오퍼레이터 (Go)
├── policies/            # OPA Rego 정책
└── deploy/
    ├── local/           # Docker Compose 로컬 환경
    └── argocd/          # ArgoCD 배포 매니페스트
```
