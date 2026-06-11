# 메시징 시스템 비교 — Edge DIP 선택 근거

## 배경

Edge DIP에서 중앙(Portal API)이 N개 Edge 노드에 릴리스/승인 이벤트를 전달해야 합니다.  
Edge 노드는 대부분 방화벽/DMZ 뒤에 위치하므로 **중앙 → Edge inbound 접속이 불가능**합니다.  
이 제약 조건을 전제로 Kafka, RabbitMQ, NATS JetStream을 비교했습니다.

## 요구사항

| 요구사항 | 설명 |
|----------|------|
| Outbound only | Edge가 중앙에 먼저 연결, 중앙이 Edge에 직접 접속하는 경로 없음 |
| 오프라인 내성 | Edge가 일시적으로 끊겼다 재접속해도 메시지 유실 없음 |
| Fan-out | 릴리스 1건 발행 시 구독 중인 모든 Edge가 동시 수신 |
| 낮은 운영 복잡도 | Edge 사이드 footprint 최소화, 인프라 추가 구성 최소화 |

## 비교

| 항목 | Kafka | RabbitMQ | NATS JetStream |
|------|-------|----------|----------------|
| **운영 무게** | 무거움 (KRaft/ZooKeeper + broker cluster) | 중간 | 가벼움 (단일 바이너리) |
| **DMZ outbound 터널** | ❌ 없음 — VPN/MirrorMaker 등 별도 구성 필요 | ❌ 없음 — 리버스 프록시 등 별도 구성 필요 | ✅ LeafNode 내장 |
| **오프라인 내성** | ✅ 로그 보관 | ✅ 큐 durable | ✅ JetStream persist |
| **Fan-out** | ✅ Consumer Group | ✅ Exchange/Binding | ✅ Subject 와일드카드 |
| **Edge 사이드 footprint** | 무거움 | 중간 | 매우 작음 |
| **생태계/레퍼런스** | 매우 풍부 | 풍부 | 상대적으로 적음 |

## 결정적 차이: NATS LeafNode

Kafka와 RabbitMQ는 DMZ outbound-only 환경을 네이티브로 지원하지 않습니다.  
중앙이 Edge에 메시지를 밀어 넣으려면 다음 중 하나가 필요합니다.

- Kafka: MirrorMaker 2 또는 별도 VPN 터널
- RabbitMQ: Federation plugin 또는 리버스 프록시

NATS는 **LeafNode가 프로토콜에 내장**되어 있어, Edge에서 `nats://central:4222`에 outbound 접속 하나만으로 터널이 완성됩니다. 코드/인프라 추가 없이 동일한 Subject/Stream 이름 그대로 사용할 수 있습니다.

```
Edge NATS (LeafNode) ──outbound──> 중앙 NATS (LeafNode Hub :7422)
     └── subscribe: releases.published.>
     └── subscribe: approvals.APPROVED.>
     └── publish:   edge.heartbeat.<edge-id>
```

## 각 시스템이 더 적합한 경우

**Kafka**
- 대용량 로그 스트리밍 (초당 수십만 건 이상)
- 강력한 순서 보장 및 장기 이벤트 소싱
- Kafka Connect, Kafka Streams 등 생태계 활용이 필요할 때
- Edge DIP 규모에는 과잉

**RabbitMQ**
- 복잡한 라우팅 (exchange type: direct/topic/fanout/headers)
- AMQP 호환이 필요한 레거시 연동
- DMZ 문제는 별도 해결이 필요

**NATS JetStream**
- DMZ/방화벽 뒤 Edge에 실시간 이벤트 전달
- 운영 복잡도를 낮추고 싶은 소~중규모
- 메시지 볼륨이 크지 않은 제어 채널 용도

## 결론

Edge DIP의 핵심 제약(DMZ outbound-only, N개 Edge fan-out, 오프라인 내성)을 가장 단순하게 충족하는 선택이 NATS JetStream입니다.

**트레이드오프**: Kafka/RabbitMQ 대비 생태계와 운영 레퍼런스가 적습니다. Edge 수가 수백 개를 넘거나 메시지 볼륨이 대폭 증가하는 시점에 재검토가 필요합니다.
