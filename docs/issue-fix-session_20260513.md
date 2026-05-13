# Edge DIP 이슈 감사 및 수정 세션
**날짜:** 2026-05-13

---

## 주요 Prompt (사용자 요청 원문)

1. "아직 미완된 작업이 있나요?" (이전 세션에서 이어짐)
2. `/oh-my-claudecode:team 3:executor 현재 구현된 것까지 이슈가 될 만한 것 찾아줘`
3. `ultrawork 이슈 모두 수정해줘`
4. `portal-api 재시작해줘`
5. `vs-code 에서 portal-web 에 에러가 있는것 같은데`
6. `tsconfig.json` (baseUrl 경고 관련)
7. `redis 는 왜 있는 거지?`
8. `deep-interview 다시 검토해보고, 필요없으면 제거해`
9. `앞으로 세션이 종료될때마다 현재 세션의 prompt 와 응답을 docs 폴더에 "축약어-날짜.md" 로 기록해줘`
10. `이번 세션요약 저장할때는 꼭 prompt 를 넣어줘`
11. `이번세션 요약 저장 해줘`

---

## 세션 개요

이전 세션에서 구현한 Edge DIP 포털 시스템(portal-api, portal-web, CI/인프라)에 대해 3개 에이전트로 이슈 감사를 수행하고, ultrawork로 발견된 이슈를 병렬 수정한 세션.

---

## 이슈 감사 결과 (3 에이전트 병렬 실행)

### portal-api — CRITICAL 4건 / HIGH 14건 / MEDIUM 6건 / LOW 8건

| 심각도 | 이슈 |
|--------|------|
| CRITICAL | JWT 서명 검증 없음 |
| CRITICAL | DEV_MODE=true 로 전체 인증 우회 가능 |
| CRITICAL | RequestedBy UUID 하드코딩 |
| CRITICAL | Agent 라우터 mTLS/인증 없음 |
| HIGH | 이중멱등성 오류 묵살 (silent discard) |
| HIGH | 버전 충돌 시 500 반환 (409여야 함) |
| HIGH | CVE 체크 float64만 검사 (int 타입 우회 가능) |
| HIGH | 배포 상태 업데이트 오류 묵살 |
| HIGH | edgeName URL 미이스케이프 |

### portal-web — CRITICAL 4건 / HIGH 7건 / MEDIUM 9건 / LOW 7건

| 심각도 | 이슈 |
|--------|------|
| CRITICAL | 모든 뮤테이션 try/catch 없음 |
| CRITICAL | WebSocket Vite 프록시 우회, Bearer 토큰 미전송 |
| CRITICAL | DeploymentRecord 타입과 실제 사용 타입 불일치 |
| HIGH | 모든 페이지 isError 상태 미렌더링 |
| HIGH | ErrorBoundary 없음 |
| HIGH | WebSocket 언마운트 후 메모리 누수 |
| HIGH | useDeployments 클라이언트 필터 후 total 오표시 |

### CI/인프라 — CRITICAL 4건 / HIGH 6건 / MEDIUM 7건 / LOW 5건

| 심각도 | 이슈 |
|--------|------|
| CRITICAL | COSIGN 개인키 디스크 파일로 기록 |
| CRITICAL | NATS 연결 인증/TLS 없음 |
| CRITICAL | YAML 인젝션 (ImageRef 미검증) |
| HIGH | Trivy Docker 소켓 마운트 |
| HIGH | release-gate CVE gate summary 항상 PASSED |
| HIGH | hostPID: true 불필요한 호스트 PID 공유 |

---

## 수정 완료 목록 (ultrawork 병렬 수정)

### portal-api (10개)
- `middleware/auth.go` — DEV_MODE 경고 로그 + JWT sub 클레임 추출
- `handler/approval.go` — RequestedBy 실제 요청자로 교체, 멱등성 오류 503 반환, 버전 충돌 409, CVE int/float64 모두 처리
- `handler/agent.go` — Version 1→0, 배포 상태 오류 로깅
- `handler/release.go` — PublishRelease 201→200, DB 오류 404 vs 500 구분
- `service/harbor.go` — edgeName URL 인코딩

### portal-web (7개)
- `ApprovalsPage/ReleasesPage/EdgesPage` — 뮤테이션 try/catch + 에러 UI
- 모든 페이지 — isError 상태 렌더링
- `useEdgeWebSocket.ts` — 언마운트 후 재연결 메모리 누수 수정
- `DeploymentPage.tsx` — 상태 타입 안전성 + 날짜 null 처리
- `useDeployments.ts` — pagination total 필터 후 건수로 수정
- `pagination.tsx` — aria-label 접근성 + 총 건수 항상 표시
- `ErrorBoundary.tsx` + `App.tsx` — ErrorBoundary 추가

### CI/인프라 (10개)
- `release-gate.yml` — COSIGN env var 방식, Trivy 소켓 제거, gate summary 수정, curl --fail-with-body, release_id null 검증
- `portal-api-ci.yml`, `edge-agent-ci.yml` — COSIGN env var 방식
- `updater.go` — YAML 인젝션 방어, Nak on failure, sanitizeName 8→16자
- `catalogpackage_controller.go` — CRITICAL TODO 주석
- `daemonset.yaml` — hostPID: true → false
- `update-operator/config/rbac/` — RBAC 매니페스트 신규 생성

---

## 기타 수정

### portal-api 재시작
- 환경변수: `DATABASE_URL`, `NATS_URL`, `DEV_MODE=true`
- 바이너리: `/tmp/portal-api`, PID: 63160
- 헬스체크: `{"status":"ok"}` 확인

### tsconfig.json baseUrl 제거
- `moduleResolution: "bundler"` 모드에서 `baseUrl`이 VS Code 경고 유발
- `paths`만으로 `@/*` 별칭 동작 — `baseUrl` 제거

### Redis 완전 제거
- 코드 전체에서 실제 사용처 없음 확인
- 제거 파일: `config/config.go`, `configmap.yaml`, `values.yaml`, `docker-compose.yml`

---

## 빌드 검증

| 프로젝트 | 명령 | 결과 |
|---------|------|------|
| portal-api | `GOARCH=arm64 GOOS=darwin go build ./...` | ✅ |
| portal-web | `yarn tsc --noEmit` | ✅ |
| edge-agent | `GOARCH=arm64 GOOS=darwin go build ./...` | ✅ |
| update-operator | `GOARCH=arm64 GOOS=darwin go build ./...` | ✅ |

---

## 워크플로 메모

- ultrawork 모드 사용 → 3개 에이전트 병렬 수정 → 빌드 검증
- 세션 요약은 수동 요청 방식으로 저장 (`docs/축약어-날짜.md`)
- 저장 시 사용자 prompt 원문 반드시 포함

---

# Session 2: 2026-05-13 (오후)

## 주요 Prompt (사용자 요청 원문)

1. "소스 코드 기반으로 README.md 한글로 작성해줘"
2. "@docs/edgedip_impl_plan_20260512.md 의 시스템아키텍처 내용을 현 코드에 맞게 고쳐서 README.md 에 넣어줘"
3. "승인요청을 중앙에서 ui 에서 등록하는 게 아니고, edge agent 에서 하는 건가?"
4. "ui 에 승인요청 등록하는 화면이 있는데?"
5. "릴리즈 publish 는 어디에서 하는 거지?"
6. "ui 에서 publish 할 수 있게 추가하고, README 에도 반영해줘"
7. "이번 세션 저장해줘"

---

## 세션 개요

소스 코드 전체를 분석하여 README.md를 한글로 작성하고, 구현 계획서 아키텍처를 실제 코드 기반으로 수정했으며, 릴리스 publish UI 기능을 추가한 세션.

---

## 작업 내용

### README.md 신규 작성
- 시스템 아키텍처 다이어그램 (ASCII)
- 컴포넌트 설명 표
- 핵심 기능 (릴리스 관리, 승인 워크플로, 원격 세션, 엣지 에이전트)
- 데이터 모델 요약
- 기술 스택, 로컬 개발 환경, OPA 정책, 디렉토리 구조

### 아키텍처 다이어그램 수정 (계획서 → 실제 코드 기반)

| 항목 | 계획서 | 실제 코드 |
|---|---|---|
| 통신 방식 | mTLS + gRPC | NATS JetStream outbound only |
| 캐시 | Redis + NATS | NATS만 사용 |
| NATS 스트림 | 추상적 언급 | `RELEASES`, `APPROVALS`, `EDGE_EVENTS` 명시 |
| 배포 실행 | Helm upgrade | Harbor pull + CatalogPackage CRD apply |
| 모니터링 | Thanos/Grafana | 미구현 (제외) |

### 승인 요청 생성 경로 명확화
- **UI 수동 생성**: ApprovalsPage에서 릴리스 + 에지 선택 → `POST /api/v1/approvals`
- **edge-agent 자동 생성**: `releases.published.>` 수신 시 → `POST /agent/v1/approval-requests`
- 두 경로 모두 README에 명시

### 릴리스 publish UI 추가
- `portal-web/src/hooks/useReleases.ts` — `usePublishRelease` 훅 추가 (`POST /releases/:id/publish`)
- `portal-web/src/pages/ReleasesPage.tsx` — 발행 버튼 추가 (SCANNED/SIGNED 상태만 표시, 에러 인라인 표시)
- `yarn tsc --noEmit` 통과 확인

---

## 발견된 미구현 사항 (Blocker)

- **portal-api `PublishRelease` 핸들러가 NATS 이벤트를 발행하지 않음**
  - `handler/release.go`는 DB 상태만 변경, `natsSvc` 미주입
  - publish 후 edge-agent가 이벤트를 수신하지 못하는 구조적 결함

### 다음 세션 작업 (Exact Next Step)
`portal-api/internal/handler/release.go`의 `PublishRelease`에 NATS 이벤트 발행 추가:
1. `ReleaseHandler` 구조체에 `nats *service.NatsService` 필드 추가
2. `NewReleaseHandler` 생성자 수정
3. `main.go`에서 `natsSvc` 주입
4. `PublishRelease` 핸들러 끝에 `natsSvc.PublishReleaseNotification()` 호출 추가

---

## 수정된 파일

| 파일 | 상태 | 비고 |
|---|---|---|
| `README.md` | ✅ 완료 | 소스 코드 기반 한글 README |
| `portal-web/src/hooks/useReleases.ts` | ✅ 완료 | `usePublishRelease` 훅 추가 |
| `portal-web/src/pages/ReleasesPage.tsx` | ✅ 완료 | publish 버튼 추가 |
