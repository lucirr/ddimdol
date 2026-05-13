CREATE TYPE release_status AS ENUM ('DRAFT', 'SCANNED', 'SIGNED', 'PUBLISHED', 'DEPRECATED');

CREATE TABLE releases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    package_name TEXT NOT NULL,
    version TEXT NOT NULL,
    artifact_digest TEXT NOT NULL,
    sbom_uri TEXT NOT NULL DEFAULT '',
    cve_report JSONB NOT NULL DEFAULT '{}',
    signature TEXT NOT NULL DEFAULT '',
    signed_by TEXT NOT NULL DEFAULT '',
    status release_status NOT NULL DEFAULT 'DRAFT',
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(package_name, version)
);
