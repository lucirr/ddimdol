# 중앙 관제 시스템 스캐폴딩 결과

> 작성일: 2026-05-13  
> 기반 문서: [edgedip_impl_plan_20260512.md](./edgedip_impl_plan_20260512.md)

---

## 생성된 구조

```
didimdol/
├── portal-api/                  # Go 1.22 + Gin 백엔드
│   ├── cmd/server/main.go       # 듀얼 포트 서버 (8080 API / 8081 mTLS Agent)
│   ├── internal/
│   │   ├── config/config.go     # Viper 환경변수 설정
│   │   ├── domain/              # 엔티티 (EdgeNode, Release, Approval, Deployment, RemoteSession, AuditLog)
│   │   ├── handler/             # API 핸들러 stub (22개 엔드포인트)
│   │   ├── middleware/          # JWT 인증, 감사 로그 미들웨어
│   │   ├── repository/          # Repository 인터페이스 + Postgres stub
│   │   └── service/             # 승인 상태머신, 에지 관리, NATS 발행/구독
│   ├── migrations/              # SQL 마이그레이션 001~006
│   ├── go.mod
│   └── Makefile
├── portal-web/                  # React 18 + TypeScript 프론트엔드
│   ├── src/
│   │   ├── types/               # EdgeNode, Release, Approval, AuditLog 타입
│   │   ├── hooks/               # TanStack Query hooks (useEdges, useApprovals, useReleases)
│   │   ├── components/          # Badge, Card, Button UI + Sidebar/Header 레이아웃
│   │   └── pages/               # Dashboard, Edges, Approvals, Releases 페이지
│   ├── package.json
│   └── vite.config.ts
└── deploy/local/                # 로컬 개발 인프라
    ├── docker-compose.yml       # PostgreSQL 16, Redis 7, NATS 2.10, Keycloak 24
    ├── .env.example
    ├── init-db.sql              # keycloak DB 자동 생성
    └── Makefile
```

---

## 기술 스택

| 영역 | 기술 |
|------|------|
| 백엔드 | Go 1.22 + Gin |
| DB | PostgreSQL 16 |
| 캐시 | Redis 7 |
| 메시징 | NATS 2.10 JetStream |
| 인증 | Keycloak 24 (OIDC) |
| 프론트엔드 | React 18 + TypeScript + Vite + TanStack Query + Tailwind |

---

## API 엔드포인트 (stub 구현 완료)

### Public API (port 8080)
- `GET /health`
- `GET/POST /api/v1/edges`, `GET /api/v1/edges/:id`
- `GET/POST /api/v1/releases`, `POST /api/v1/releases/:id/publish`
- `GET/POST /api/v1/approvals`, `POST /api/v1/approvals/:id/{approve,reject,defer}`
- `GET/POST /api/v1/remote-sessions`, `POST /api/v1/remote-sessions/:id/{activate,terminate}`
- `GET /api/v1/audit-logs`

### Agent API (port 8081, mTLS 전용)
- `POST /agent/v1/heartbeat`
- `POST /agent/v1/download-progress`
- `POST /agent/v1/deployment-result`

---

## 빌드 검증

- Go: `go build ./...` → PASS
- Frontend: `yarn build` (tsc + vite) → PASS, 0 TypeScript errors

---

## 다음 단계

1. **Repository 구현**: Postgres stub → 실제 sqlx 쿼리
2. **NATS 연동**: nats.go service → JetStream publish/subscribe 실제 구현
3. **Keycloak OIDC**: middleware/auth.go JWT 검증 실제 구현
4. **에지 에이전트 PoC**: heartbeat 수신 → EdgeNode status 갱신 흐름 검증

---

## 로컬 실행 방법

```bash
# 인프라 시작
cd deploy/local && make up

# 백엔드 실행
cd portal-api && cp ../deploy/local/.env.example .env && make run

# 프론트엔드 실행
cd portal-web && yarn dev
# → http://localhost:5173
```
