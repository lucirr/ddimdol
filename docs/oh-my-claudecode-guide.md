# oh-my-claudecode (OMC) 사용 가이드

> GitHub: https://github.com/Yeachan-Heo/oh-my-claudecode  
> 버전: v4.13.7 | 라이선스: MIT

---

## 목차

1. [개요](#개요)
2. [설치](#설치)
3. [오케스트레이션 모드](#오케스트레이션-모드)
4. [매직 키워드](#매직-키워드)
5. [설정](#설정)
6. [MCP 도구](#mcp-도구)
7. [훅 시스템](#훅-시스템)
8. [기능](#기능)
9. [문제 해결](#문제-해결)
10. [실습](#실습)
11. [oh-my-claudecode vs oh-my-openagent 비교](#oh-my-claudecode-vs-oh-my-openagent-비교)
12. [참고 링크](#참고-링크)

---

## 개요

**oh-my-claudecode (OMC)** 는 Claude Code를 위한 멀티 에이전트 오케스트레이션 시스템으로 복잡한 코딩 작업을 여러 전문 AI 에이전트가 협업하여 병렬로 처리하며 수행합니다.

---

## 설치

### 요구사항

- Claude Code CLI
- Claude Max/Pro 구독 또는 Anthropic API 키
- Node.js ≥ 20.0.0
- tmux (Team 모드, rate-limit 감지에 필요)
- (선택) Gemini CLI, Codex CLI (크로스 검증용)

### 설치 방법

Claude Code 세션 안에서 순서대로 실행합니다.

```bash
# Marketplace에 저장소 등록
/plugin marketplace add https://github.com/Yeachan-Heo/oh-my-claudecode

# 플러그인 설치 (스코프 선택 필요)
/plugin install oh-my-claudecode

# 플러그인 리로드 (설치 직후 반드시 실행)
/reload-plugins

# OMC 초기화 (Claude Code 세션 내에서)
/omc-setup

# Claude Code를 완전히 종료 후 다시 실행
```

> `omc-setup`은 환경에 맞는 설정을 자동으로 구성해줍니다.

#### 플러그인 설치 스코프 선택

`/plugin install` 실행 시 적용 범위를 선택하는 프롬프트가 나타납니다.

| 스코프 | 저장 위치 | 적용 범위 | 추천 상황 |
|--------|-----------|-----------|-----------|
| **User** | `~/.claude/` | 모든 프로젝트 | OMC를 항상 사용하고 싶을 때 |
| **Project** | `./.claude/` | 현재 프로젝트만 | 팀 프로젝트, 버전 관리에 포함할 때 (권장) |
| **Local** | `./.claude/local/` | 현재 프로젝트만 (git 제외) | 개인 설정, `.gitignore`에 포함할 때 |

설치가 완료되면 세션 시작 시 OMC가 자동으로 활성화됩니다.

**설치 확인**

```bash
# CLI 정상 동작 확인
omc --version
```

설치 후 CLI 명령어: `oh-my-claudecode`, `omc`, `omc-cli`

---

## 오케스트레이션 모드

### 1. Team 모드

단계별 파이프라인으로 작업을 처리하는 기본 모드입니다.

```
team-plan → team-prd → team-exec → team-verify → team-fix
```

| 단계 | 역할 |
|------|------|
| `team-plan` | 요구사항 분석 및 계획 수립 |
| `team-prd` | PRD(제품 요구사항 문서) 작성 |
| `team-exec` | 코드 구현 |
| `team-verify` | 검증 및 테스트 |
| `team-fix` | 이슈 수정 |

**명령어 형식:**

```
/team N:provider "작업 설명"
```

- `N` — 병렬로 실행할 워커 에이전트 수
- `provider` — 사용할 에이전트 종류

| Provider | 용도 |
|----------|------|
| `executor` | 일반 구현 작업 |
| `codex` | 코드 리뷰, 아키텍처 검증 |
| `gemini` | UI/UX 설계, 대용량 컨텍스트 작업 |
| `claude` | 일반 Claude 작업 |

**사용 예시:**

```bash
# 3개 에이전트로 TypeScript 오류 수정
/team 3:executor "fix all TypeScript errors"

# 2개 Codex 에이전트로 보안 리뷰
/team 2:codex "review auth module for security issues"

# 2개 Gemini 에이전트로 UI 리디자인
/team 2:gemini "redesign UI components for accessibility"
```

### 2. Autopilot 모드

5단계 자율 실행 파이프라인입니다.

```
확장 → 계획 → 실행 → QA → 검증
```

**사용법:**
```
autopilot: 사용자 대시보드 페이지를 만들어줘
```

### 3. Ralph 모드

검증 루프가 포함된 지속 실행 모드입니다. 반복적인 작업이나 장시간 작업에 적합합니다.

```
ralph: API 엔드포인트 전체 리팩토링
```

### 4. Ultrawork 모드

최대 병렬 처리 모드입니다. 독립적인 작업이 많을 때 사용합니다.

```
ultrawork: 모든 컴포넌트에 단위 테스트 추가
```

### 5. Deep Interview 모드

소크라테스식 요구사항 명확화 모드입니다. 아이디어가 막연할 때 먼저 사용하세요.

```
deep-interview: 새 프로젝트 시작
```

---

## 매직 키워드

파워 유저를 위한 선택적 단축키. 자연어도 잘 작동합니다.

| 키워드 | 효과 | 예시 |
|--------|------|------|
| `team` | 표준 Team 오케스트레이션 | `/team 3:executor "fix all TypeScript errors"` |
| `omc team` | tmux CLI 워커 (codex/gemini/claude) | `omc team 2:codex "security review"` |
| `ccg` | 트라이-모델 Codex+Gemini 오케스트레이션 | `/ccg review this PR` |
| `autopilot` | 완전 자율 실행 | `autopilot: build a todo app` |
| `ralph` | 지속 모드 | `ralph: refactor auth` |
| `ulw` | 최대 병렬화 | `ulw fix all errors` |
| `plan` | 계획 인터뷰 | `plan the API` |
| `ralplan` | 반복적 계획 합의 | `ralplan this feature` |
| `deep-interview` | 소크라테스식 요구사항 명확화 | `deep-interview "vague idea"` |

---

### 모델 라우팅 전략

```
Haiku   → 단순 작업, 빠른 응답 필요 시
Sonnet  → 일반 개발 작업 (기본값)
Opus    → 복잡한 아키텍처 결정, 심층 분석
```

---

---

## 설정

### 프로젝트 설정 (`.omc/config.json`)

```json
{
  "defaultMode": "team",
  "models": {
    "fast": "claude-haiku-4-5",
    "default": "claude-sonnet-4-6",
    "powerful": "claude-opus-4-7"
  },
  "notifications": {
    "telegram": {
      "token": "YOUR_BOT_TOKEN",
      "chatId": "YOUR_CHAT_ID"
    }
  }
}
```

### 전역 설정 (`~/.omc/config.json`)

모든 프로젝트에 적용되는 기본 설정입니다.

---

## MCP 도구

OMC는 다음 MCP 도구를 내장합니다.

| 도구 | 설명 |
|------|------|
| State Management | 세션 상태 관리 |
| Notepad | 컨텍스트 유지 메모 |
| Project Memory | 장기 프로젝트 기억 |
| LSP (12종) | 코드 인텔리전스 |
| AST Grep | 구조적 코드 검색 |
| Python REPL | 데이터 분석 |
| Session Search | 이전 세션 검색 |

---

## 훅 시스템

20개 훅이 Claude Code 라이프사이클 이벤트를 가로챕니다.

| 카테고리 | 훅 예시 |
|----------|---------|
| Core | `PreToolUse`, `PostToolUse` |
| Context Management | 컨텍스트 압축 전 보존 |
| Quality/Verification | 자동 코드 품질 검사 |
| Lifecycle | 세션 시작/종료 처리 |

훅 비활성화:
```bash
export OMC_DISABLE_HOOKS=true
```

---

## 기능

### 새 기능 개발

```
# 1. 요구사항이 명확하지 않을 때 → Deep Interview 먼저
/deep-interview 사용자 알림 시스템을 만들고 싶어

# 2. 요구사항이 명확할 때 → Team 모드 바로 실행
team: 이메일 + 푸시 알림을 지원하는 알림 시스템 구현

# 3. 독립 작업이 많을 때 → Ultrawork
ultrawork: 모든 API 엔드포인트에 swagger 문서 추가
```

### 장시간 작업

```
# Ralph 모드로 검증 루프 포함 실행
ralph: 레거시 코드베이스 전체를 TypeScript로 마이그레이션
```

### 빠른 단순 작업

```
# Autopilot으로 간단히
autopilot: README 업데이트
```

---

## 문제 해결

### Rate Limit 발생 시

OMC가 자동으로 감지하고 재시도합니다. tmux가 설치되어 있어야 합니다.

```bash
# tmux 설치 (macOS)
brew install tmux
```

### 에이전트가 멈췄을 때

```
/cancel
```

### 세션 재개

```
/resume
```

### 진단

```bash
/omc --diagnostics
```

---

## 실습

Claude Code에 oh-my-claudecode를 설치하고 실행해보는 간단한 예제입니다.

### 설치

터미널에서 실행.

```bash
mkdir omc-practice && cd omc-practice
git init
claude  # Claude Code 실행

/plugin marketplace add https://github.com/Yeachan-Heo/oh-my-claudecode

/plugin install oh-my-claudecode

/reload-plugins

/omc-setup

/exit
# Claude Code를 완전히 종료 후 다시 실행
```

### 실행

Claude 터미널 또는 VS Code 에서 실행.

#### step 1: Plan — 요구사항 명확할때

즉시 구현 계획 문서 작성 

```bash
plan: 간단한 Todo list UI를 React로 만들어줘 
```

#### * step 1: Deep Interview — 아이디어 막연할 때 요구사항 명확화

대화 후 스펙 확정,
plan 파일로 저장도 가능,
이어서 autopilot, team 자동 실행도 가능

```bash
deep-interview: 간단한 Todo list UI를 React로 만들어줘 
```

#### step 2: Autopilot — 전체 뼈대 자동 생성

탐색 → 계획 → 구현 → QA → 검증 단계 자동 실행

plan 없이 바로 실행도 가능 (autopilot: 간단한 Todo list UI를 React로 만들어줘)

```bash
autopilot: plan 결과 내용 또는 plan 파일  붙혀넣기
```

#### step 3: Team — 추가 기능 병렬 구현

plan → prd → exec → verify → fix 순서로 파이프라인 실행

```bash
# 워커 에이전트 2개가 병렬로 파일을 나눠 작성
# :executor = 구현 전문 에이전트
/team 2:executor "Todo 완료 항목 필터링 기능 추가"
```

#### step 4: Ultrawork — 독립 작업 일괄 처리

순서 없이 동시에 처리해도 되는 작업들 처리

team과 달리 파이프라인 없음 — 그냥 최대한 동시에 처리

```bash
ulw: 모든 컴포넌트 파일에 Jest 단위 테스트 작성
```

#### step 5: Ralph — 품질 기준 달성까지 반복

완료 조건이 있고 자동 반복이 필요한 마무리 작업
완료 조건이 측정 가능할수록 효과적 (빌드 성공, 커버리지 %, 에러 0개)
ultrawork 병렬 실행을 내부적으로 포함하므로 ulw + ralph 같이 쓸 필요 없음

```bash
# 테스트 작성 → test -cover 실행 → 80% 미달 → 추가 작성 → 재검증 → 80% 달성 시 자동 종료
ralph: 테스트 커버리지 80% 이상으로 만들어줘
```

#### 전체 흐름

```
[아이디어]
      ↓
plan 또는 deep-interview   → 요구사항 구체화
      ↓
autopilot        → 전체 뼈대 자동 생성
      ↓
team             → 추가 기능 병렬 구현
      ↓
ulw              → 독립 작업 일괄 처리
      ↓
ralph            → 품질 기준 달성까지 반복
      ↓
[배포 가능한 프로젝트]
```

---

## oh-my-claudecode vs oh-my-openagent 비교

oh-my-claudecode(OMC) 는 Claude Code CLI 위에서 슬래시 커맨드로 멀티에이전트 개발 워크플로우를 자동화하는 오케스트레이션인 반면, oh-my-openagent 는 OpenAI Agents SDK 기반으로 Python 코드로 에이전트 간 핸드오프와 도구 호출을 정의하는 범용 에이전트 앱 구축 프레임워크입니다.

### 한눈에 비교

| 항목 | oh-my-claudecode (OMC) | oh-my-openagent (omo) |
|------|------------------------|------------------------|
| **주요 플랫폼** | Claude Code 전용 | OpenCode 기반, Claude Code 호환 |
| **GitHub 스타** | ~33.5k | ~57.3k |
| **버전** | v4.13.7 | v4.0.0 |
| **지원 모델** | Claude (Haiku/Sonnet/Opus) 중심 | 멀티 제공사 (Claude, GPT-5.5, Kimi K2.6, GLM-5.1) |
| **팀 모드** | 최대 병렬 워커 N개 | 최대 8명 병렬, tmux 실시간 시각화 |
| **에이전트 수** | 19개 전문 에이전트 | 6개 핵심 에이전트 (역할 특화) |
| **철학** | Claude 생태계 최적화 | 특정 플랫폼 락인 탈피, 멀티 모델 |

### 에이전트 구조 비교

**oh-my-claudecode** — 19개 에이전트를 4개 레인으로 구성
```
Build/Analysis → Review → Domain → Coordination
```

**oh-my-openagent** — 6개 핵심 에이전트, 역할 명확히 분리
| 에이전트 | 역할 | 기본 모델 |
|----------|------|-----------|
| Sisyphus | 메인 오케스트레이터 | Claude Opus 4.7 / Kimi K2.6 |
| Hephaestus | 자율 구현 워커 | GPT-5.5 |
| Prometheus | 전략 계획 | — |
| Oracle | 아키텍처/디버깅 전문가 | — |
| Librarian | 문서/코드 검색 전문가 | — |
| Explore | 빠른 코드베이스 탐색 | — |

### 주요 기능 차이

**oh-my-openagent만의 특징:**
- **IntentGate**: 명령 실행 전 사용자 의도 분석
- **Todo Enforcer**: 에이전트가 유휴 상태가 되는 것을 방지
- **Ralph Loop (`/ulw-loop`)**: 작업 100% 완료까지 자기 참조 반복
- **`/init-deep`**: 토큰 효율화를 위한 계층적 AGENTS.md 자동 생성
- **멀티 제공사 모델 라우팅**: 작업 카테고리에 따라 Claude/GPT/Kimi/GLM 자동 선택

**oh-my-claudecode만의 특징:**
- **Claude Code 네이티브 최적화**: 훅, 스킬, MCP 완전 통합
- **Notepad Wisdom**: 세션 간 학습/결정/이슈 맥락 유지
- **ccg 모드**: 트라이-모델 Codex+Gemini 오케스트레이션
- **스킬 학습 시스템**: 반복 패턴을 스킬로 저장·재사용
- **Deep Interview**: 소크라테스식 요구사항 명확화 전용 모드

### Claude 요금제별 호환성

> **중요:** oh-my-openagent는 API 키 기반으로 모델을 호출합니다.  
> Claude Team/Pro 구독은 API 키를 제공하지 않으므로, **oh-my-openagent에서 Claude 모델을 사용할 수 없습니다.**

| 요금제 | oh-my-claudecode | oh-my-openagent (Claude 모델) |
|--------|-----------------|-------------------------------|
| Claude Pro ($20/월) | 사용 가능 | 사용 불가 (API 키 없음) |
| Claude Team ($30/월) | 사용 가능 | 사용 불가 (API 키 없음) |
| Anthropic API (종량제) | 사용 가능 | 사용 가능 |
| GPT-5.5, Kimi, GLM 등 | 해당 없음 | 사용 가능 (별도 구독) |

Claude Team/Pro 구독자라면 oh-my-openagent에서 Claude 대신 GPT-5.5, Kimi K2.6, GLM-5.1 등 타 제공사 모델을 조합해서 사용해야 합니다.

### 어떤 걸 선택할까?

| 상황 | 추천 |
|------|------|
| Claude Code를 주로 사용하고 Claude 모델에 집중하고 싶다 | **oh-my-claudecode** |
| Claude Team/Pro 구독만 있고 API 키가 없다 | **oh-my-claudecode** |
| Anthropic API 키가 있고 여러 LLM을 혼용하고 싶다 | **oh-my-openagent** |
| OpenCode 환경을 사용한다 | **oh-my-openagent** |
| Claude 생태계(스킬, 훅, MCP) 최대 활용이 목표다 | **oh-my-claudecode** |
| 팀 단위로 Claude Code를 도입하려 한다 | **oh-my-claudecode** |

---

## 참고 링크

- [oh-my-claudecode GitHub](https://github.com/Yeachan-Heo/oh-my-claudecode)
- [oh-my-openagent GitHub](https://github.com/code-yeongyu/oh-my-openagent)
