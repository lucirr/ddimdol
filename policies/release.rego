package edgedip.release

import rego.v1

default allow := false

# central-operator만 릴리즈 생성/발행 가능
allow if {
    input.role == "central-operator"
    input.action in {"create", "publish", "deprecate"}
}

# 모든 인증된 사용자는 릴리즈 조회 가능
allow if {
    input.role in {"central-operator", "edge-admin", "auditor"}
    input.action == "read"
}

# CVE gate: CRITICAL이 있으면 발행 불가
deny_publish if {
    input.action == "publish"
    input.cve_report.critical > 0
}

publish_allowed if {
    input.action == "publish"
    not deny_publish
    allow
}
