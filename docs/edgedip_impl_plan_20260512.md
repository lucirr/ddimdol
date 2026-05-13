# Edge DIP 통합 포털 세부 구현 계획

> 작성일: 2026-05-12  
> 담당: 김영창  
> 기반 문서: [김영창_작업계획.md](./김영창_작업계획.md)

---

## 1. 기술 스택 확정

### 1.1 중앙 관제 시스템 (Central Control Plane)

| 영역 | 기술 선택 | 선택 이유 |
|------|----------|----------|
| 백엔드 API | **Go 1.22 + Gin/Echo** | 폐쇄망 단일 바이너리 배포 용이, 낮은 메모리 풋프린트, k8s 생태계 친화 |
| 인증/인가 | **Keycloak 24 + OIDC + OPA** | 카탈로그 통합 인증 요건 충족, RBAC/ABAC 모두 지원 |
| 메인 DB | **PostgreSQL 16** | 트랜잭션 일관성 (승인 워크플로우 필수), JSONB로 메타데이터 유연성 |
| 캐시/큐 | **Redis 7 + NATS JetStream** | NATS는 폐쇄망 환경에서 Kafka보다 경량, 에지-중앙 메시징 적합 |
| 프론트엔드 | **React 18 + TypeScript + Vite + TanStack Query + shadcn/ui** | 폐쇄망 정적 자산 배포 용이 |
| 메트릭 | **Prometheus + Thanos + Grafana 10** | 에지 다수 클러스터 → Thanos 글로벌 뷰, Push Gateway로 폐쇄망 적응 |
| 로그/감사 | **Loki + OpenTelemetry Collector** | Elastic 대비 경량, 폐쇄망 운영 부담 감소 |

### 1.2 배포 파이프라인 (Release Pipeline)

| 영역 | 기술 선택 | 선택 이유 |
|------|----------|----------|
| Git 저장소 | **Gitea 1.21** | 폐쇄망 자체 호스팅, GitHub Actions 호환 (Gitea Actions) |
| 컨테이너 레지스트리 | **Harbor 2.10** | 서명(Cosign/Notary v2), 취약점 스캔(Trivy 내장), 복제 기능 |
| GitOps | **ArgoCD 2.10** | ApplicationSet으로 다수 에지 클러스터 관리, 폐쇄망 표준 |
| 패키지 매니저 | **Helm 3.14 + Kustomize** | 차트 버전 관리 + 에지별 오버레이 |
| SBOM 생성 | **Syft** | 컨테이너 + 바이너리 모두 지원 |
| 취약점 스캔 | **Trivy + Grype (이중 검증)** | DB 오프라인 미러링 지원 |
| 서명 | **Cosign (키 기반)** | 폐쇄망에서 OIDC keyless 사용 불가, KMS 키 기반 서명 |
| 단방향 전송 | **binup + rsync over SSH** | 단방향 터널링 |

### 1.3 에지 노드 (Edge Plane)

| 영역 | 기술 선택 | 선택 이유 |
|------|----------|----------|
| 에지 에이전트 | **Go 1.22 (정적 바이너리)** | systemd 데몬, 의존성 최소화 |
| 오케스트레이션 | **K3s 1.29 / RKE2** | 에지 환경 경량 k8s |
| 업데이트 Operator | **Operator SDK (Go)** | CRD 기반 카탈로그 자동 업데이트, 롤백 기본 제공 |
| 로컬 메트릭 | **node-exporter + cAdvisor** | 노드 up/down + 상세 자원 |
| 통신 | **mTLS + gRPC (에지→중앙 단방향 outbound)** | 폐쇄망 inbound 차단 환경 대응 |

### 1.4 JIT 원격 접속 (Phase 4)

| 영역 | 기술 선택 |
|------|----------|
| 세션 게이트웨이 | **Apache Guacamole (HTML5 RDP/SSH/VNC)** |
| 토큰 발급 | **단기 JWT (TTL ≤ 30분, JTI 회수 가능)** |
| 화이트리스트 | **Cilium NetworkPolicy + 에지 firewalld 동적 룰** |
| 세션 녹화 | **Guacamole 세션 레코딩 + S3-호환 MinIO** |

---

## 2. 시스템 아키텍처

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
                  +---------+----------+
                            |
        +-------------------+-------------------+
        |                                       |
        v                                       v
  +-----------+                          +-----------+
  | 통합 포털  |  <-- Keycloak (OIDC) --> | 관제 콘솔  |
  | (React)   |                          | (Grafana) |
  +-----+-----+                          +-----+-----+
        |                                       ^
        v                                       |
  +-----------+    +-----------+          +-----+-----+
  | Portal    |--->| Postgres  |          | Thanos    |
  | API (Go)  |    | Redis     |          | Receiver  |
  | Gin       |    | NATS      |          +-----+-----+
  +-----+-----+    +-----------+                ^
        |                                       |
        | (승인 알림 / 명령 발행)                  | (Prometheus remote_write)
        v                                       |
  +-----------+                                 |
  | NATS      |  <==== 단방향 outbound ====      |
  | JetStream |                                 |
  +-----+-----+                                 |
        |                                       |
========|=======================================|================
        |   에지 (Edge Plane, N개 사이트)         |
========|=======================================|================
        |                                       |
        v                                       |
  +-----------+    +--------------+    +--------+-------+
  | Edge      |--->| K3s/RKE2     |--->| node-exporter  |
  | Agent     |    | + Update     |    | cAdvisor       |
  | (Go)      |    |   Operator   |    | Prometheus     |
  +-----+-----+    +------+-------+    +----------------+
        |                 |
        | (binup pull)    | (Catalog CRD reconcile)
        v                 v
  +-----------+    +-----------+
  | Local     |    | 로컬 카탈로그|
  | Harbor    |    | (Helm)    |
  | Mirror    |    +-----------+
  +-----------+
```

**핵심 통신 패턴**:
- 에지는 항상 outbound 클라이언트 (NATS subscribe, Harbor pull, Prometheus remote_write)
- 중앙은 직접 에지에 inbound 접속 불가 → JIT 원격 접속 시에만 에지가 reverse tunnel 개설

---

## 3. Phase별 세부 스프린트 분해

### Phase 1: 중앙 바이너리 릴리즈 파이프라인 (2026-06 ~ 07, 4 스프린트)

**Sprint 1.1 (2026-06-01 ~ 06-14): 기반 구축**
- Gitea + Harbor 폐쇄망 설치 및 HA 구성
- Cosign KMS 키 생성, 신뢰 체인 정의
- 사내 CA 발급 및 mTLS 인증서 부트스트랩

**Sprint 1.2 (2026-06-15 ~ 06-28): 보안점검 파이프라인**
- Syft SBOM 자동 생성 워크플로우 (Gitea Actions)
- Trivy/Grype 이중 스캔 + CVE 임계값 게이트 (CRITICAL=0, HIGH≤3)
- Cosign 서명 및 Harbor push

**Sprint 1.3 (2026-07-01 ~ 07-14): binup 단방향 전송**
- binup 파이프라인 통합 (외부망 → 폐쇄망)
- 매니페스트 무결성 검증(SHA-256 + 서명 검증)
- ArgoCD ApplicationSet 템플릿 작성

**Sprint 1.4 (2026-07-15 ~ 07-31): 에지 노티 메커니즘**
- NATS JetStream `releases.*` 스트림 설계
- 릴리즈 이벤트 publish API (`POST /releases`)
- 에지 에이전트 PoC subscribe 동작 확인
- **Phase 1 게이트**: 1개 테스트 에지에 자동 릴리즈 노티 도달

---

### Phase 2: 관리자 승인 기반 업데이트 (2026-08 ~ 10, 6 스프린트)

**Sprint 2.1 (2026-08-01 ~ 08-14): 데이터 모델 & 인증**
- Keycloak Realm/Client/Role 설계 (`edge-admin`, `central-operator`, `auditor`)
- Postgres 스키마 마이그레이션 (ApprovalRequest, EdgeNode, AuditLog)
- OPA 정책 작성

**Sprint 2.2 (2026-08-15 ~ 08-31): 승인 워크플로우 API**
- Portal API: 승인 요청 CRUD, 상태 머신
  - `PENDING → APPROVED/REJECTED/DEFERRED → APPLIED → ROLLED_BACK`
- NATS 알림 발행 (에지 관리자 대상)
- 멱등성 키 처리, 낙관적 락

**Sprint 2.3 (2026-09-01 ~ 09-14): 승인 UI**
- React 알람 센터 (실시간 SSE/WebSocket)
- 승인 상세 화면 (변경 사항 diff, SBOM 표시, CVE 리포트)
- 다중 승인자 정책 UI

**Sprint 2.4 (2026-09-15 ~ 09-30): Agent Pull 다운로드**
- 에지 에이전트 승인 확인 → Harbor pull
- 디스크 공간 사전 점검, 부분 다운로드 재개
- 다운로드 진행률 보고 (NATS reply)

**Sprint 2.5 (2026-10-01 ~ 10-14): 카탈로그 자동 업데이트 Operator**
- CRD: `CatalogPackage`, `CatalogRelease`
- Reconcile 로직 (Helm upgrade with `--atomic --wait`)
- 자동 롤백 트리거 (헬스체크 실패 시 helm rollback)

**Sprint 2.6 (2026-10-15 ~ 10-31): 통합 테스트**
- E2E 시나리오 5종 (정상 승인/거부/연기/롤백/네트워크 단절 복구)
- **Phase 2 게이트**: 3개 에지에서 승인→배포→롤백 성공

---

### Phase 3: 에지-중앙 관제 연동 (2026-11 ~ 12, 4 스프린트)

**Sprint 3.1 (2026-11-01 ~ 11-14): 상태 수집 에이전트**
- node-exporter + cAdvisor 사이드카
- 에지 에이전트 health beacon (10초 주기, NATS `edge.heartbeat.<edge-id>`)
- 자원 임계치 로컬 alert evaluation

**Sprint 3.2 (2026-11-15 ~ 11-30): Push 메트릭 파이프라인**
- 에지 Prometheus → Thanos Receiver (remote_write)
- Thanos Query 글로벌 뷰
- 라벨 표준화 (`edge_id`, `region`, `tenant`)

**Sprint 3.3 (2026-12-01 ~ 12-14): 관제 콘솔**
- 통합 포털 내 에지 상태 대시보드 (React + Grafana 임베드)
- 에지 토폴로지 뷰 (up/down 색상, 자원 게이지)
- 알람 룰 (3 heartbeat miss = down)

**Sprint 3.4 (2026-12-15 ~ 12-31): 안정화**
- 50개 에지 부하 테스트
- **Phase 3 게이트**: 30초 이내 에지 장애 감지율 ≥ 99%

---

### Phase 4: JIT 원격 접속 (2027-01 ~ 02, 공동: 송원빈, 4 스프린트)

**Sprint 4.1 (2027-01-01 ~ 01-14): 인터페이스 정의**
- 책임 분담: 송원빈(에지 reverse tunnel daemon) / 김영창(토큰 발급 + 게이트웨이 + 감사)
- API 계약서 작성 (OpenAPI 3.1)

**Sprint 4.2 (2027-01-15 ~ 01-31): 토큰 발급 & 화이트리스트**
- RemoteSession API (생성, 활성화, 종료)
- 시간 제한 JWT 발급 (max TTL 30분, 사유 기록)
- 에지 화이트리스트 동적 적용 (NetworkPolicy patch)

**Sprint 4.3 (2027-02-01 ~ 02-14): Guacamole 게이트웨이**
- Guacamole + Postgres 백엔드 구성
- SSO 연동 (Keycloak)
- 세션 녹화 → MinIO

**Sprint 4.4 (2027-02-15 ~ 02-28): 감사 & 종료**
- AuditLog 모든 키스트로크/명령 캡처
- 자동 세션 만료 + 화이트리스트 회수
- **Phase 4 게이트**: 보안팀 침투 테스트 통과

---

### Phase 5: 성능 시험 (2027-05 ~ 06, 2 스프린트)

**Sprint 5.1 (2027-05-01 ~ 05-31): KPI 검증 환경**
- 50회 연속 배포 자동화 스크립트
- 실패 분류 체계 (네트워크/디스크/서명/롤백)
- 카오스 엔지니어링 (Litmus) 시나리오

**Sprint 5.2 (2027-06-01 ~ 06-30): 최종 검증**
- 50회 × 3 환경 = 150 배포 시도, 성공률 측정
- **Phase 5 게이트**: 99% 이상 (148/150) 달성

---

## 4. API 설계 (주요 엔드포인트)

### 릴리즈 관리
```
POST   /api/v1/releases                       # 새 릴리즈 등록
GET    /api/v1/releases?status=&package=
GET    /api/v1/releases/{id}                  # 상세 (SBOM, CVE 포함)
POST   /api/v1/releases/{id}/publish          # 에지 노티 발행
```

### 승인 워크플로우
```
POST   /api/v1/approvals                      # 승인 요청 생성 (idempotency-key)
GET    /api/v1/approvals?edge_id=&status=
GET    /api/v1/approvals/{id}
POST   /api/v1/approvals/{id}/approve         # body: {reason, schedule_at?}
POST   /api/v1/approvals/{id}/reject          # body: {reason}
POST   /api/v1/approvals/{id}/defer           # body: {until, reason}
GET    /api/v1/approvals/{id}/events          # 상태 전이 이력
```

### 에지 노드 / 관제
```
GET    /api/v1/edges
GET    /api/v1/edges/{id}
GET    /api/v1/edges/{id}/heartbeats?since=
POST   /api/v1/edges/{id}/commands
GET    /api/v1/edges/{id}/catalog
```

### 원격 세션 (JIT)
```
POST   /api/v1/remote-sessions
GET    /api/v1/remote-sessions?edge_id=
POST   /api/v1/remote-sessions/{id}/activate
POST   /api/v1/remote-sessions/{id}/terminate
GET    /api/v1/remote-sessions/{id}/recording
```

### 감사 로그
```
GET    /api/v1/audit-logs?actor=&resource=&from=&to=
GET    /api/v1/audit-logs/export
```

### 에지 에이전트 콜백 (mTLS 전용)
```
POST   /agent/v1/heartbeat                    # 10초 주기
POST   /agent/v1/download-progress
POST   /agent/v1/deployment-result
```

---

## 5. 데이터 모델

### EdgeNode
```sql
id: UUID (PK)
name: TEXT UNIQUE
region: TEXT
tenant_id: UUID
status: ENUM(UP, DOWN, DEGRADED, UNKNOWN)
last_heartbeat_at: TIMESTAMPTZ
agent_version: TEXT
k8s_version: TEXT
capabilities: JSONB
labels: JSONB
public_key: TEXT        -- mTLS 클라이언트 인증서 SPKI
created_at, updated_at
```

### Release
```sql
id: UUID (PK)
package_name: TEXT
version: SEMVER
artifact_digest: TEXT   -- sha256:...
sbom_uri: TEXT
cve_report: JSONB       -- {critical:0, high:2, medium:7}
signature: TEXT
signed_by: TEXT
status: ENUM(DRAFT, SCANNED, SIGNED, PUBLISHED, DEPRECATED)
published_at: TIMESTAMPTZ
UNIQUE(package_name, version)
```

### ApprovalRequest
```sql
id: UUID (PK)
release_id: UUID FK -> Release
edge_id: UUID FK -> EdgeNode
requested_by: UUID FK -> User
status: ENUM(PENDING, APPROVED, REJECTED, DEFERRED, APPLIED, ROLLED_BACK, EXPIRED)
decision_by: UUID NULLABLE
decision_reason: TEXT
scheduled_at: TIMESTAMPTZ NULLABLE
deferred_until: TIMESTAMPTZ NULLABLE
idempotency_key: TEXT UNIQUE
version: INT            -- 낙관적 락
created_at, updated_at
INDEX(edge_id, status), INDEX(release_id)
```

### DeploymentRecord
```sql
id: UUID (PK)
approval_id: UUID FK -> ApprovalRequest
edge_id: UUID FK
release_id: UUID FK
phase: ENUM(DOWNLOADING, APPLYING, HEALTHCHECK, COMPLETED, FAILED, ROLLED_BACK)
progress_pct: SMALLINT
error_code: TEXT
started_at, finished_at
```

### RemoteSession
```sql
id: UUID (PK)
edge_id: UUID FK
operator_id: UUID FK -> User
reason: TEXT NOT NULL
ticket_ref: TEXT
status: ENUM(PENDING_APPROVAL, ACTIVE, EXPIRED, TERMINATED)
approved_by: UUID
token_jti: TEXT UNIQUE
ttl_seconds: INT CHECK (ttl_seconds <= 1800)
whitelist_entries: JSONB
recording_uri: TEXT
activated_at, expires_at, terminated_at
```

### AuditLog (append-only)
```sql
id: BIGSERIAL (PK)
ts: TIMESTAMPTZ NOT NULL
actor_id: UUID
actor_type: ENUM(USER, AGENT, SYSTEM)
action: TEXT
resource_type: TEXT
resource_id: TEXT
outcome: ENUM(SUCCESS, FAILURE)
request_id: TEXT
client_ip: INET
metadata: JSONB
hash_prev: BYTEA        -- 해시 체인 (변조 방지)
hash_self: BYTEA
INDEX(ts), INDEX(actor_id), INDEX(resource_type, resource_id)
```

### Heartbeat (TimescaleDB hypertable 권장)
```sql
edge_id: UUID
ts: TIMESTAMPTZ
cpu_pct, mem_pct, disk_pct: NUMERIC
node_count, ready_node_count: INT
extra: JSONB
PRIMARY KEY (edge_id, ts)
```

---

## 6. 의존성 및 리스크

### Phase 간 의존성
```
Phase 1 (릴리즈 파이프라인)
   └─> Phase 2 (승인 워크플로우, Release 엔티티 필요)
            └─> Phase 3 (관제, EdgeNode 모델 공유)
                     └─> Phase 4 (JIT, 관제 콘솔 에지 선택 UI 재사용)
                              └─> Phase 5 (전 Phase 통합 KPI 검증)
```

**병렬화 기회**: Phase 3 메트릭 파이프라인은 Sprint 2.5 시점부터 별도 트랙으로 병행 가능

### 외부 팀 의존성

| 의존 | 대상 | 시점 | 리스크 | 완화 |
|------|------|------|--------|------|
| 송원빈 (Phase 4) | 에지 측 reverse tunnel daemon | 2027-01 | 인터페이스 변경 | Sprint 4.1에서 OpenAPI 계약 잠금, mock 서버 제공 |
| 신동우 (인증 인가) | Keycloak Realm 설계 합의 | 2026-08 | 역할 모델 미합의 시 Phase 2 지연 | Sprint 2.1 시작 전 RBAC 매트릭스 사전 합의 |
| 보안팀 | CVE 임계값 정책 | 2026-06 | 임계값 과도 시 릴리즈 차단 | Phase 1 시작 전 SLA 문서화 |
| 인프라팀 | 폐쇄망 binup 채널 | 2026-06 | 외부망 전송 대역 부족 | 사전 대역 확보 요청, 증분 전송 |

### 기술 리스크

| 리스크 | 영향 | 확률 | 완화 |
|--------|------|------|------|
| NATS HA 운영 미경험 | 메시징 단절 시 승인 누락 | 중 | Redis Streams 백업 채널, 장애 훈련 |
| 50개 에지 동시 다운로드 시 Harbor 대역 포화 | 배포 성공률 하락 | 중 | 에지 로컬 Harbor 미러, 다운로드 윈도우 분산 |
| Cosign 키 유출 | 전체 신뢰 체인 붕괴 | 낮·고영향 | HSM/KMS 저장, 키 회전 절차 |
| 배포 성공률 99% 미달 | KPI 실패 | 중 | Phase 2부터 실패 분류 수집 |

---

## 7. Phase별 검증 기준 (Definition of Done)

### Phase 1
- [ ] push → SBOM/CVE/서명/Harbor push 완전 자동화
- [ ] CRITICAL CVE 1건이라도 발견 시 파이프라인 fail
- [ ] binup 무결성 검증 100%
- [ ] 릴리즈 노티 도달 시간 < 30초
- [ ] 단위 테스트 커버리지 ≥ 80%

### Phase 2
- [ ] 승인 상태 머신 모든 전이 통합 테스트 통과
- [ ] 동시 승인 요청 100건 처리 시 데드락/유실 0건
- [ ] 자동 롤백 헬스체크 실패 30초 이내 발동
- [ ] 모든 승인/거부 액션 AuditLog + 해시 체인 검증
- [ ] 3개 에지 E2E 시나리오 5종 통과

### Phase 3
- [ ] Heartbeat 누락 30초 이내 DOWN 전이, 정확도 ≥ 99%
- [ ] Thanos 글로벌 쿼리 50개 에지 14일 데이터 5초 이내 응답
- [ ] 알람 false positive rate ≤ 1%

### Phase 4
- [ ] 토큰 TTL 초과 시 자동 세션 종료 + 화이트리스트 회수
- [ ] 모든 세션 100% 녹화 + 1년 보관
- [ ] 보안팀 침투 테스트 통과
- [ ] 송원빈 모듈과 통합 E2E 시나리오 통과

### Phase 5
- [ ] 50회 × 3 환경 = 성공률 ≥ 99% (148/150)
- [ ] 실패 케이스 전체 근본 원인 분석 보고서
- [ ] 카오스 시나리오 복구율 ≥ 95%
- [ ] KPI 측정 재현 스크립트 공개

---

## 8. 예상 디렉토리 구조

```
didimdol/
├── portal-api/          # Go 백엔드
├── portal-web/          # React 프론트엔드
├── edge-agent/          # 에지 Go 에이전트
├── update-operator/     # k8s Operator
├── deploy/
│   ├── helm/            # Helm 차트
│   └── argocd/          # ApplicationSet
├── pipelines/
│   └── gitea-actions/   # CI 워크플로우
└── docs/
    └── adr/             # 아키텍처 결정 기록
```

---

## 9. 착수 전 확인 사항

1. 기술 스택 변경 여부 (NATS vs Kafka, Keycloak vs 자체 IAM)
2. 신동우님과 RBAC 매트릭스 사전 합의 회의 일정
3. 송원빈님과 Phase 4 인터페이스 계약 책임자 확정
4. binup 단방향 채널 SLA (전송 주기/대역) 확인
5. 99% KPI 측정 범위 정의 (다운로드 완료 vs 헬스체크 통과까지)
