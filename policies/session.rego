package edgedip.session

import rego.v1

default allow := false

# central-operator만 원격 세션 생성 가능
allow if {
    input.role == "central-operator"
    input.action in {"create", "activate", "terminate"}
}

# TTL 검증: 최대 30분
valid_ttl if {
    input.ttl_seconds <= 1800
    input.ttl_seconds > 0
}

# 세션 생성 시 reason 필수
valid_session if {
    count(input.reason) > 0
    valid_ttl
}

# auditor는 세션 기록만 조회 가능
allow if {
    input.role == "auditor"
    input.action == "read_recording"
}
