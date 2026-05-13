CREATE TYPE actor_type AS ENUM ('USER', 'AGENT', 'SYSTEM');
CREATE TYPE audit_outcome AS ENUM ('SUCCESS', 'FAILURE');

CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_id UUID,
    actor_type actor_type NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL DEFAULT '',
    resource_id TEXT NOT NULL DEFAULT '',
    outcome audit_outcome NOT NULL,
    request_id TEXT NOT NULL DEFAULT '',
    client_ip INET,
    metadata JSONB NOT NULL DEFAULT '{}',
    hash_prev BYTEA,
    hash_self BYTEA
);

CREATE INDEX idx_audit_logs_ts ON audit_logs(ts DESC);
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
