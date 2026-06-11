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
  Portal API  <──subscribe─ EDGE_EVENTS 스트림
  (하트비트 수신)             │   edge.heartbeat.<edge-id>
                            │        │
                            │        └─> DB 업데이트 + WebSocket broadcast
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
