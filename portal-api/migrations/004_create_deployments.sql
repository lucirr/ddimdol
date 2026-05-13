CREATE TYPE deployment_phase AS ENUM (
    'DOWNLOADING', 'APPLYING', 'HEALTHCHECK', 'COMPLETED', 'FAILED', 'ROLLED_BACK'
);

CREATE TABLE deployment_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    approval_id UUID NOT NULL REFERENCES approval_requests(id),
    edge_id UUID NOT NULL REFERENCES edge_nodes(id),
    release_id UUID NOT NULL REFERENCES releases(id),
    phase deployment_phase NOT NULL DEFAULT 'DOWNLOADING',
    progress_pct SMALLINT NOT NULL DEFAULT 0 CHECK (progress_pct BETWEEN 0 AND 100),
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);
