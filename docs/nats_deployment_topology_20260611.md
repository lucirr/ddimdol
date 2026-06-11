# NATS 배포 토폴로지 비교 — DMZ 환경

## 배경

중앙망과 에지망 사이에 DMZ 서버가 별도로 존재하는 환경에서  
NATS를 어디에 배치할지, 또는 HAProxy로 대체할 수 있는지 비교합니다.

모든 구성에서 **에지→중앙 outbound-only** 원칙을 유지해야 합니다.

---

## 구성 A: 중앙 + DMZ + 에지 모두 NATS

```
[중앙망]                  [DMZ]                   [에지망]
중앙 NATS            DMZ NATS                에지 NATS
(LeafNode Hub)  <──  (LeafNode)  <──────────  (LeafNode)
     :7422              :7422                    :7422
                                                   ↑
                                              Edge Agent
                                              (localhost)
```

### 동작 방식

- 에지 NATS가 DMZ NATS에 outbound 연결
- DMZ NATS가 중앙 NATS에 outbound 연결
- 두 구간 모두 inbound 포트 오픈 불필요
- Edge Agent는 로컬 NATS(`localhost:4222`)에만 접속

### 방화벽 오픈

| 출발 | 도착 | 포트 | 방향 |
|------|------|------|------|
| 에지망 | DMZ | :4222 또는 :7422 | outbound |
| DMZ | 중앙망 | :7422 | outbound |

### 장점

- DMZ에서 연결 수 집약 (에지 100개 → 중앙 NATS 연결은 DMZ NATS 1개)
- DMZ NATS에서 메시지 필터링/검사 가능
- 에지 내부 프로세스(Update Operator 등)가 로컬 NATS를 공유 가능
- 중앙 NATS 장애 시 DMZ NATS가 일정 시간 버퍼링

### 단점

- NATS 서버를 3곳에 설치/운영
- 에지 노드마다 NATS 서버 프로세스 추가 (리소스 사용 증가)
- 운영 복잡도 가장 높음

---

## 구성 B: 중앙 NATS + DMZ HAProxy + 에지 Agent만 (현재 구현)

```
[중앙망]          [DMZ]                  [에지망]
중앙 NATS   <──  HAProxy               Edge Agent
                (TCP passthrough)  <──  (직접 연결)
                    :4222
```

### 동작 방식

- HAProxy가 TCP 모드로 NATS 포트를 단순 포워딩
- Edge Agent가 HAProxy를 통해 중앙 NATS에 직접 접속
- NATS LeafNode 프로토콜은 Agent ↔ 중앙 NATS 사이에서 직접 처리
- HAProxy는 NATS 프로토콜을 이해하지 않음 — 순수 TCP 터널

### 방화벽 오픈

| 출발 | 도착 | 포트 | 방향 |
|------|------|------|------|
| 에지망 | DMZ | :4222 | outbound |
| DMZ | 중앙망 | :4222 | outbound |

### HAProxy 설정 예시

```
frontend nats_front
    bind *:4222
    mode tcp
    default_backend nats_back

backend nats_back
    mode tcp
    server central-nats central-nats.internal:4222 check
```

### 장점

- 에지에 NATS 서버 설치 불필요 — Agent만으로 동작
- 인프라 구성 단순 (HAProxy는 이미 DMZ 표준 구성요소)
- 운영 복잡도 가장 낮음

### 단점

- 에지 수가 많아지면 중앙 NATS에 직접 연결 수가 선형 증가
- DMZ에서 메시지 레벨 필터링 불가 (TCP passthrough이므로)
- 에지 내부 프로세스 간 NATS 공유 불가

---

## 구성 C: 중앙 NATS + DMZ HAProxy + 에지 NATS

```
[중앙망]          [DMZ]                  [에지망]
중앙 NATS   <──  HAProxy           에지 NATS (LeafNode)
                (TCP passthrough)  <──  ↑
                    :4222          Edge Agent (localhost)
```

### 동작 방식

- 에지 NATS가 HAProxy를 통해 중앙 NATS에 LeafNode 연결
- Edge Agent는 로컬 NATS에만 접속
- DMZ는 여전히 단순 TCP passthrough

### 장점

- 에지 내부 프로세스 간 NATS 공유 가능
- 에지 NATS가 로컬 버퍼 역할 (중앙 연결 끊겨도 에지 내부 통신 유지)
- DMZ는 HAProxy만으로 단순하게 유지

### 단점

- 에지 노드마다 NATS 서버 프로세스 추가
- 구성 A보다는 단순하지만 구성 B보다는 복잡

---

## 비교 요약

| 항목 | A (NATS 3곳) | B (HAProxy + Agent만) | C (HAProxy + 에지 NATS) |
|------|:---:|:---:|:---:|
| 에지 설치 요소 | Agent + NATS | Agent만 | Agent + NATS |
| DMZ 설치 요소 | NATS | HAProxy | HAProxy |
| 운영 복잡도 | 높음 | 낮음 | 중간 |
| 에지 수 확장성 | ✅ DMZ에서 집약 | △ 중앙 연결 수 증가 | △ 중앙 연결 수 증가 |
| DMZ 메시지 필터링 | ✅ | ❌ | ❌ |
| 에지 내부 프로세스 공유 | ✅ | ❌ | ✅ |
| 에지 오프라인 내성 | ✅ (에지 NATS 버퍼) | ✅ (JetStream) | ✅ (JetStream + 로컬 버퍼) |

---

## 권장 선택 기준

| 상황 | 권장 구성 |
|------|----------|
| 에지 수 적고 운영 단순화 우선 | **B** (현재 구현) |
| 에지 내부 프로세스 간 통신 필요 | **C** |
| 에지 수 많고 (100+) DMZ에서 집약 필요 | **A** |
| DMZ에서 메시지 레벨 보안 검사 필요 | **A** |

현재 Edge DIP 구현은 **구성 B**입니다.  
에지 수 증가 또는 에지 내부 멀티 프로세스 요건이 생기는 시점에 C 또는 A로 전환을 검토합니다.
