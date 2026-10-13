
-- ==========================================
-- AutoDataOps
-- Migration 000001
-- Core Enterprise Database Schema
-- ==========================================

CREATE SCHEMA core;

-- ------------------------------------------
-- Migration registry
-- ------------------------------------------

CREATE TABLE core.schema_migrations (

    version bigint PRIMARY KEY
        CHECK (version > 0),

    name text NOT NULL,

    checksum char(64) NOT NULL
        CHECK (checksum ~ '^[0-9a-f]{64}$'),

    applied_at timestamptz NOT NULL
        DEFAULT now()
);

-- ------------------------------------------
-- Organizations
-- ------------------------------------------

CREATE TABLE core.organizations (

    id uuid PRIMARY KEY
        DEFAULT gen_random_uuid(),

    slug text NOT NULL UNIQUE,

    display_name text NOT NULL,

    created_at timestamptz NOT NULL
        DEFAULT now(),

    CONSTRAINT org_slug_format
        CHECK (
            slug ~ '^[a-z][a-z0-9-]{2,62}$'
        ),

    CONSTRAINT org_name_size
        CHECK (
            length(btrim(display_name))
            BETWEEN 1 AND 200
        )
);

-- ------------------------------------------
-- Data pipelines
-- ------------------------------------------

CREATE TABLE core.pipelines (

    id uuid PRIMARY KEY
        DEFAULT gen_random_uuid(),

    organization_id uuid NOT NULL
        REFERENCES core.organizations(id)
        ON DELETE RESTRICT,

    source_type text NOT NULL,

    external_key text NOT NULL,

    display_name text NOT NULL,

    created_at timestamptz NOT NULL
        DEFAULT now(),

    updated_at timestamptz NOT NULL
        DEFAULT now(),

    CONSTRAINT pipeline_source_size
        CHECK (
            length(btrim(source_type))
            BETWEEN 1 AND 64
        ),

    CONSTRAINT pipeline_key_size
        CHECK (
            length(btrim(external_key))
            BETWEEN 1 AND 255
        ),

    CONSTRAINT pipeline_name_size
        CHECK (
            length(btrim(display_name))
            BETWEEN 1 AND 255
        ),

    CONSTRAINT pipelines_tenant_id_uk
        UNIQUE (organization_id, id),

    CONSTRAINT pipelines_external_key_uk
        UNIQUE (
            organization_id,
            source_type,
            external_key
        )
);

-- ------------------------------------------
-- Incidents
-- ------------------------------------------

CREATE TABLE core.incidents (

    id uuid PRIMARY KEY
        DEFAULT gen_random_uuid(),

    organization_id uuid NOT NULL
        REFERENCES core.organizations(id)
        ON DELETE RESTRICT,

    pipeline_id uuid NOT NULL,

    source text NOT NULL,

    summary text NOT NULL,

    severity text NOT NULL,

    status text NOT NULL
        DEFAULT 'detected',

    detected_at timestamptz NOT NULL,

    created_at timestamptz NOT NULL
        DEFAULT now(),

    updated_at timestamptz NOT NULL
        DEFAULT now(),

    version bigint NOT NULL
        DEFAULT 1,

    CONSTRAINT incident_source_size
        CHECK (
            length(btrim(source))
            BETWEEN 1 AND 128
        ),

    CONSTRAINT incident_summary_size
        CHECK (
            length(btrim(summary))
            BETWEEN 1 AND 512
        ),

    CONSTRAINT incident_severity_valid
        CHECK (
            severity IN (
                'low',
                'medium',
                'high',
                'critical'
            )
        ),

    CONSTRAINT incident_status_valid
        CHECK (
            status IN (
                'detected',
                'investigating',
                'mitigating',
                'resolved',
                'closed',
                'dismissed'
            )
        ),

    CONSTRAINT incident_version_positive
        CHECK (version >= 1),

    CONSTRAINT incident_dates_ordered
        CHECK (updated_at >= created_at),

    CONSTRAINT incidents_tenant_id_uk
        UNIQUE (organization_id, id),

    CONSTRAINT incident_pipeline_same_tenant_fk
        FOREIGN KEY (
            organization_id,
            pipeline_id
        )
        REFERENCES core.pipelines(
            organization_id,
            id
        )
        ON DELETE RESTRICT
);

-- ------------------------------------------
-- Verification evidence
-- ------------------------------------------

CREATE TABLE core.verification_evidence (

    id uuid PRIMARY KEY
        DEFAULT gen_random_uuid(),

    organization_id uuid NOT NULL,

    incident_id uuid NOT NULL,

    validator text NOT NULL,

    passed boolean NOT NULL,

    checked_at timestamptz NOT NULL,

    recorded_at timestamptz NOT NULL
        DEFAULT now(),

    details jsonb NOT NULL
        DEFAULT '{}'::jsonb,

    CONSTRAINT evidence_validator_size
        CHECK (
            length(btrim(validator))
            BETWEEN 1 AND 128
        ),

    CONSTRAINT evidence_details_object
        CHECK (
            jsonb_typeof(details) = 'object'
        ),

    CONSTRAINT evidence_tenant_incident_id_uk
        UNIQUE (
            organization_id,
            incident_id,
            id
        ),

    CONSTRAINT evidence_incident_same_tenant_fk
        FOREIGN KEY (
            organization_id,
            incident_id
        )
        REFERENCES core.incidents(
            organization_id,
            id
        )
        ON DELETE RESTRICT
);

-- ------------------------------------------
-- Incident history
-- ------------------------------------------

CREATE TABLE core.incident_transitions (

    id uuid PRIMARY KEY
        DEFAULT gen_random_uuid(),

    organization_id uuid NOT NULL,

    incident_id uuid NOT NULL,

    version_after bigint NOT NULL,

    from_status text NOT NULL,

    to_status text NOT NULL,

    reason text NOT NULL,

    verification_id uuid,

    occurred_at timestamptz NOT NULL,

    recorded_at timestamptz NOT NULL
        DEFAULT now(),

    CONSTRAINT transition_version_valid
        CHECK (version_after >= 2),

    CONSTRAINT transition_reason_size
        CHECK (
            length(btrim(reason))
            BETWEEN 1 AND 1024
        ),

    CONSTRAINT transition_from_valid
        CHECK (
            from_status IN (
                'detected',
                'investigating',
                'mitigating',
                'resolved',
                'closed',
                'dismissed'
            )
        ),

    CONSTRAINT transition_to_valid
        CHECK (
            to_status IN (
                'detected',
                'investigating',
                'mitigating',
                'resolved',
                'closed',
                'dismissed'
            )
        ),

    CONSTRAINT transition_changed
        CHECK (from_status <> to_status),

    CONSTRAINT transition_resolution_requires_evidence
        CHECK (
            (
                to_status = 'resolved'
                AND verification_id IS NOT NULL
            )
            OR
            (
                to_status <> 'resolved'
                AND verification_id IS NULL
            )
        ),

    CONSTRAINT transition_per_incident_version_uk
        UNIQUE (
            organization_id,
            incident_id,
            version_after
        ),

    CONSTRAINT transition_incident_same_tenant_fk
        FOREIGN KEY (
            organization_id,
            incident_id
        )
        REFERENCES core.incidents(
            organization_id,
            id
        )
        ON DELETE RESTRICT,

    CONSTRAINT transition_evidence_same_incident_fk
        FOREIGN KEY (
            organization_id,
            incident_id,
            verification_id
        )
        REFERENCES core.verification_evidence(
            organization_id,
            incident_id,
            id
        )
        ON DELETE RESTRICT
);

-- ------------------------------------------
-- Transactional outbox
-- ------------------------------------------

CREATE TABLE core.outbox_events (

    id uuid PRIMARY KEY
        DEFAULT gen_random_uuid(),

    organization_id uuid NOT NULL
        REFERENCES core.organizations(id)
        ON DELETE RESTRICT,

    aggregate_type text NOT NULL,

    aggregate_id uuid NOT NULL,

    event_type text NOT NULL,

    payload jsonb NOT NULL,

    occurred_at timestamptz NOT NULL
        DEFAULT now(),

    available_at timestamptz NOT NULL
        DEFAULT now(),

    published_at timestamptz,

    attempt_count integer NOT NULL
        DEFAULT 0,

    CONSTRAINT outbox_aggregate_size
        CHECK (
            length(btrim(aggregate_type))
            BETWEEN 1 AND 64
        ),

    CONSTRAINT outbox_event_type_size
        CHECK (
            length(btrim(event_type))
            BETWEEN 1 AND 128
        ),

    CONSTRAINT outbox_payload_object
        CHECK (
            jsonb_typeof(payload) = 'object'
        ),

    CONSTRAINT outbox_attempts_positive
        CHECK (attempt_count >= 0)
);

-- ------------------------------------------
-- Initial database indexes
-- ------------------------------------------

CREATE INDEX incidents_org_open_idx
ON core.incidents (
    organization_id,
    detected_at DESC
)
WHERE status NOT IN (
    'closed',
    'dismissed'
);

CREATE INDEX incident_transitions_by_incident_idx
ON core.incident_transitions (
    organization_id,
    incident_id,
    version_after DESC
);

CREATE INDEX verification_by_incident_idx
ON core.verification_evidence (
    organization_id,
    incident_id,
    checked_at DESC
);

CREATE INDEX outbox_unpublished_idx
ON core.outbox_events (
    available_at,
    occurred_at
)
WHERE published_at IS NULL;

