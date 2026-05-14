# Portal API 보안 검토 결과

> 작성일: 2026-05-14  
> 대상 범위: `portal-api` 서버 초기화, 인증/감사 미들웨어, Agent API, WebSocket Hub, 릴리스 publish 흐름  
> 검증 명령: `go test ./...`, `go vet ./...`

---

## 검토 요약

`portal-api`는 현재 컴파일과 정적 검사 수준에서는 통과하지만, 인증 검증, Agent API 보호, 이벤트 발행, 감사 로그, 동시성 안정성 측면에서 운영 전 반드시 보완해야 할 문제가 확인되었다.

특히 JWT 서명 검증 부재와 Agent API mTLS 미적용은 외부 요청자가 상태 변경 API를 호출할 수 있는 직접적인 보안 위험으로 볼 수 있다. WebSocket Hub에는 런타임 race/panic 가능성이 있고, 릴리스 publish 이벤트 미발행은 README에 설명된 자동 승인 요청 흐름을 끊는다.

---

## Findings

| 심각도 | 항목 | 영향 |
|---|---|---|
| CRITICAL | JWT 서명 검증 부재 | 임의 Bearer 토큰으로 보호 API 접근 가능 |
| CRITICAL | Agent API mTLS 미적용 | 에지 상태/승인/배포 결과 API가 인증 없이 노출 |
| HIGH | WebSocket Hub 동시성 버그 | heartbeat 브로드캐스트 중 race 또는 panic 가능 |
| HIGH | 릴리스 publish NATS 이벤트 미발행 | `releases.published.<id>` 기반 자동 승인 요청 흐름 중단 |
| MEDIUM | 감사 로그 미들웨어 미구현 | 상태 변경 요청 추적 및 감사 증적 누락 |
| MEDIUM | TODO/stub API가 200 OK 반환 | 클라이언트가 미구현 기능을 성공으로 오인 |

---

## 상세 내용

### 1. JWT 서명 검증 부재 및 임의 Bearer 토큰 통과 가능성

- 심각도: CRITICAL
- 근거:
  - `portal-api/internal/middleware/auth.go:46`
  - `portal-api/internal/middleware/auth.go:49`
  - `portal-api/internal/middleware/auth.go:50`
- 내용:
  - `Auth()` 미들웨어는 Authorization 헤더가 `Bearer `로 시작하는지만 확인한다.
  - JWT 서명, issuer, audience, expiration, realm/client 검증이 없다.
  - `parseJWTSub()` 실패 시에도 `sub = "unknown"`으로 설정하고 요청을 계속 통과시킨다.
- 영향:
  - 공격자가 임의 문자열 또는 위조 JWT를 Bearer 토큰으로 전달해 `/api/v1/*` 보호 API에 접근할 수 있다.
- 권장 조치:
  - Keycloak JWKS 기반 JWT 서명 검증을 구현한다.
  - `iss`, `aud`, `exp`, `nbf`, `azp` 또는 client ID를 검증한다.
  - 파싱 또는 검증 실패 시 반드시 `401 Unauthorized`로 중단한다.
  - `DEV_MODE=true`는 로컬 환경에서만 허용되도록 환경 검증 또는 명시적 unsafe flag를 추가한다.

### 2. Agent 서버가 mTLS 없이 일반 HTTP로 노출됨

- 심각도: CRITICAL
- 근거:
  - `portal-api/cmd/server/main.go:156`
  - `portal-api/cmd/server/main.go:173`
  - `portal-api/cmd/server/main.go:184`
- 내용:
  - 코드 주석과 로그는 Agent 서버를 `mTLS-only`로 표현하지만 실제 실행은 `http.ListenAndServe(agentAddr, agentRouter)`이다.
  - TLS 설정, client certificate 검증, Agent 인증 미들웨어가 없다.
- 영향:
  - 인증되지 않은 요청자가 `/agent/v1/heartbeat`, `/agent/v1/approval-requests`, `/agent/v1/deployment-result`를 호출할 수 있다.
  - 에지 상태 위조, 승인 요청 스팸, 배포 결과 조작이 가능해진다.
- 권장 조치:
  - `http.Server`에 `TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert`를 설정한다.
  - 신뢰할 Agent CA를 로드하고 `ListenAndServeTLS` 또는 동등한 TLS 서버 구성을 사용한다.
  - 인증서 subject/SAN과 등록된 edge ID를 매핑해 요청의 `edge_id`와 일치 여부를 검증한다.
  - mTLS 도입 전까지는 Agent API를 내부 네트워크로 제한하거나 별도 shared secret/JWT 인증을 적용한다.

### 3. WebSocket Hub의 `RLock` 중 map 삭제 동시성 버그

- 심각도: HIGH
- 근거:
  - `portal-api/internal/hub/hub.go:56`
  - `portal-api/internal/hub/hub.go:62`
- 내용:
  - broadcast 처리 중 `h.mu.RLock()`을 잡은 상태로 `delete(h.clients, client)`를 수행한다.
  - 읽기 락 상태에서 map을 수정하기 때문에 동시성 안전성이 깨진다.
- 영향:
  - heartbeat 브로드캐스트 중 `fatal error: concurrent map iteration and map write` 또는 data race가 발생할 수 있다.
  - WebSocket 연결이 많은 환경에서 서버 안정성이 떨어진다.
- 권장 조치:
  - Hub 이벤트 루프 내부에서는 단일 goroutine 소유 모델을 사용하고 mutex를 제거하거나, 삭제가 필요한 client 목록을 모은 뒤 write lock으로 별도 처리한다.
  - `go test -race`로 WebSocket broadcast/register/unregister 시나리오를 검증한다.

### 4. 릴리스 publish 시 NATS 이벤트 미발행

- 심각도: HIGH
- 근거:
  - `portal-api/internal/handler/release.go:158`
  - `portal-api/internal/handler/release.go:195`
  - `portal-api/cmd/server/main.go:100`
- 내용:
  - README는 릴리스 발행 시 `releases.published.<id>` 이벤트가 발행되고 edge-agent가 이를 구독해 승인 요청을 자동 생성한다고 설명한다.
  - 실제 `PublishRelease`는 release 상태를 `PUBLISHED`로 저장한 뒤 응답만 반환한다.
  - `ReleaseHandler` 생성자에는 `NatsService`가 주입되지 않아 이벤트를 발행할 경로도 없다.
- 영향:
  - 릴리스 발행 후 자동 승인 요청 생성 흐름이 동작하지 않는다.
  - 운영자는 UI상 발행 성공을 봐도 에지 배포 워크플로가 진행되지 않을 수 있다.
- 권장 조치:
  - `ReleaseHandler`에 `NatsService`를 주입한다.
  - publish DB 저장 성공 후 `PublishReleaseNotification()`을 호출한다.
  - 이벤트 발행 실패 시 정책을 명확히 정한다. 예: 500 반환, outbox 저장 후 재시도, 또는 경고 상태 반환.
  - publish 이벤트 발행 성공/실패 테스트를 추가한다.

### 5. 감사 로그 미들웨어 미구현

- 심각도: MEDIUM
- 근거:
  - `portal-api/cmd/server/main.go:93`
  - `portal-api/internal/middleware/audit.go:9`
  - `portal-api/internal/middleware/audit.go:12`
- 내용:
  - `AuditLogger()`가 API router에 등록되어 있지만 실제로는 `c.Next()` 후 아무 데이터도 저장하지 않는다.
  - TODO 주석만 존재한다.
- 영향:
  - 승인, 릴리스 발행, 에지 등록 같은 상태 변경 작업에 대한 감사 증적이 남지 않는다.
  - 장애 조사, 책임 추적, 보안 감사 대응이 어렵다.
- 권장 조치:
  - `AuditRepository`를 주입하고 상태 변경 메서드/경로에 대해 actor, action, resource, outcome, status code, request ID를 저장한다.
  - 실패 요청도 기록한다.
  - 민감정보는 마스킹하거나 저장하지 않는다.

### 6. TODO/stub API가 200 OK를 반환함

- 심각도: MEDIUM
- 근거:
  - `portal-api/internal/handler/session.go:15`
  - `portal-api/internal/handler/approval.go:289`
  - `portal-api/internal/handler/edge.go:63`
  - `portal-api/internal/handler/edge.go:67`
  - `portal-api/internal/handler/edge.go:71`
- 내용:
  - 원격 세션, approval defer/events, edge heartbeats/commands/catalog 등 여러 미구현 API가 정상 `200 OK`를 반환한다.
  - 일부 응답에는 `"TODO"` 메시지가 포함되지만 HTTP 상태는 성공이다.
- 영향:
  - 프론트엔드 또는 외부 연동 시스템이 기능 성공으로 오인할 수 있다.
  - 운영 중 누락된 기능이 조용히 실패하는 형태가 된다.
- 권장 조치:
  - 미구현 엔드포인트는 `501 Not Implemented` 또는 명시적 feature flag 비활성 응답을 반환한다.
  - README/API 문서에 구현 상태를 반영한다.
  - 운영 노출 전에는 stub 라우트를 제거하거나 실제 구현으로 대체한다.

---

## 권장 수정 우선순위

1. JWT 검증 구현 및 실패 시 차단 처리
2. Agent API mTLS 또는 동등한 인증/인가 적용
3. WebSocket Hub 동시성 버그 수정 및 race 테스트 추가
4. 릴리스 publish NATS 이벤트 발행 구현
5. 감사 로그 저장 구현
6. TODO/stub API의 성공 응답 제거 또는 실제 구현

---

## 현재 검증 결과

| 명령 | 결과 | 비고 |
|---|---|---|
| `go test ./...` | PASS | 모든 패키지가 `[no test files]` 상태라 동작 검증 범위는 제한적 |
| `go vet ./...` | PASS | vet 경고 없음 |

---

## 비고

- 이번 문서는 보안 검토 결과 기록만 포함한다.
- 코드 수정, 마이그레이션, API 동작 변경은 수행하지 않았다.
- 발견된 항목 중 CRITICAL/HIGH는 운영 배포 전 우선 조치가 필요하다.
