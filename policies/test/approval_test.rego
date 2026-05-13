package edgedip.approval_test

import data.edgedip.approval

test_central_operator_can_approve if {
    approval.allow with input as {
        "role": "central-operator",
        "action": "approve",
        "edge_id": "edge-001",
        "user_edge_id": "edge-001"
    }
}

test_edge_admin_can_approve_own_edge if {
    approval.allow with input as {
        "role": "edge-admin",
        "action": "approve",
        "edge_id": "edge-001",
        "user_edge_id": "edge-001"
    }
}

test_edge_admin_cannot_approve_other_edge if {
    not approval.allow with input as {
        "role": "edge-admin",
        "action": "approve",
        "edge_id": "edge-002",
        "user_edge_id": "edge-001"
    }
}

test_auditor_can_only_read if {
    approval.allow with input as {
        "role": "auditor",
        "action": "read",
        "edge_id": "edge-001",
        "user_edge_id": "edge-001"
    }
}

test_auditor_cannot_approve if {
    not approval.allow with input as {
        "role": "auditor",
        "action": "approve",
        "edge_id": "edge-001",
        "user_edge_id": "edge-001"
    }
}
