CREATE TYPE edge_status AS ENUM ('UP', 'DOWN', 'DEGRADED', 'UNKNOWN');

CREATE TABLE edge_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    region TEXT NOT NULL,
    tenant_id UUID NOT NULL,
    status edge_status NOT NULL DEFAULT 'UNKNOWN',
    last_heartbeat_at TIMESTAMPTZ,
    agent_version TEXT NOT NULL DEFAULT '',
    k8s_version TEXT NOT NULL DEFAULT '',
    capabilities JSONB NOT NULL DEFAULT '{}',
    labels JSONB NOT NULL DEFAULT '{}',
    public_key TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_edge_nodes_status ON edge_nodes(status);
CREATE INDEX idx_edge_nodes_tenant_id ON edge_nodes(tenant_id);
