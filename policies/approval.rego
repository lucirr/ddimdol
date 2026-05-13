package edgedip.approval

import rego.v1

# 기본 거부
default allow := false

# central-operator 역할은 모든 에지에 대해 승인 가능
allow if {
    input.role == "central-operator"
    input.action in {"approve", "reject", "defer"}
}

# edge-admin은 자신의 에지에 대해서만 승인 가능
allow if {
    input.role == "edge-admin"
    input.action in {"approve", "reject", "defer"}
    input.edge_id == input.user_edge_id
}

# auditor는 읽기만 가능
allow if {
    input.role == "auditor"
    input.action == "read"
}

# 승인 요청 생성은 edge-admin, central-operator 모두 가능
allow if {
    input.role in {"edge-admin", "central-operator"}
    input.action == "create"
}
