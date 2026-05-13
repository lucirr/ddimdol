package edgedip.session_test

import data.edgedip.session

test_central_operator_can_create_session if {
    session.allow with input as {
        "role": "central-operator",
        "action": "create",
        "ttl_seconds": 600,
        "reason": "maintenance window"
    }
}

test_central_operator_can_activate_session if {
    session.allow with input as {
        "role": "central-operator",
        "action": "activate",
        "ttl_seconds": 900,
        "reason": "emergency patch"
    }
}

test_central_operator_can_terminate_session if {
    session.allow with input as {
        "role": "central-operator",
        "action": "terminate",
        "ttl_seconds": 300,
        "reason": "done"
    }
}

test_edge_admin_cannot_create_session if {
    not session.allow with input as {
        "role": "edge-admin",
        "action": "create",
        "ttl_seconds": 600,
        "reason": "maintenance"
    }
}

test_auditor_can_read_recording if {
    session.allow with input as {
        "role": "auditor",
        "action": "read_recording"
    }
}

test_auditor_cannot_create_session if {
    not session.allow with input as {
        "role": "auditor",
        "action": "create",
        "ttl_seconds": 600,
        "reason": "audit check"
    }
}

test_valid_ttl if {
    session.valid_ttl with input as {
        "ttl_seconds": 1800
    }
}

test_ttl_exceeds_max if {
    not session.valid_ttl with input as {
        "ttl_seconds": 1801
    }
}

test_ttl_zero_invalid if {
    not session.valid_ttl with input as {
        "ttl_seconds": 0
    }
}
