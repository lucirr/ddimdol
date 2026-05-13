CREATE TYPE approval_status AS ENUM (
    'PENDING', 'APPROVED', 'REJECTED', 'DEFERRED', 'APPLIED', 'ROLLED_BACK', 'EXPIRED'
);

CREATE TABLE approval_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    release_id UUID NOT NULL REFERENCES releases(id),
    edge_id UUID NOT NULL REFERENCES edge_nodes(id),
    requested_by UUID NOT NULL,
    status approval_status NOT NULL DEFAULT 'PENDING',
    decision_by UUID,
    decision_reason TEXT NOT NULL DEFAULT '',
    scheduled_at TIMESTAMPTZ,
    deferred_until TIMESTAMPTZ,
    idempotency_key TEXT NOT NULL UNIQUE,
    version INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_approvals_edge_status ON approval_requests(edge_id, status);
CREATE INDEX idx_approvals_release ON approval_requests(release_id);
