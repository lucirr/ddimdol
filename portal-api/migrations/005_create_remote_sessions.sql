CREATE TYPE session_status AS ENUM (
    'PENDING_APPROVAL', 'ACTIVE', 'EXPIRED', 'TERMINATED'
);

CREATE TABLE remote_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    edge_id UUID NOT NULL REFERENCES edge_nodes(id),
    operator_id UUID NOT NULL,
    reason TEXT NOT NULL,
    ticket_ref TEXT NOT NULL DEFAULT '',
    status session_status NOT NULL DEFAULT 'PENDING_APPROVAL',
    approved_by UUID,
    token_jti TEXT UNIQUE,
    ttl_seconds INT NOT NULL DEFAULT 1800 CHECK (ttl_seconds <= 1800),
    whitelist_entries JSONB NOT NULL DEFAULT '[]',
    recording_uri TEXT NOT NULL DEFAULT '',
    activated_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    terminated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
