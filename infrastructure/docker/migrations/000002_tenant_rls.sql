-- AutoDataOps: tenant row-level isolation.
-- Tenant identity must be established by trusted application code.
-- No runtime privileges are granted by this migration.

DO $migration$
DECLARE
    item record;
BEGIN
    FOR item IN
        SELECT * FROM (VALUES
            ('organizations', 'id'),
            ('pipelines', 'organization_id'),
            ('incidents', 'organization_id'),
            ('incident_transitions', 'organization_id'),
            ('verification_evidence', 'organization_id'),
            ('outbox_events', 'organization_id')
        ) AS t(table_name, tenant_key)
    LOOP
        EXECUTE format(
            'ALTER TABLE core.%I ENABLE ROW LEVEL SECURITY',
            item.table_name
        );

        EXECUTE format(
            'ALTER TABLE core.%I FORCE ROW LEVEL SECURITY',
            item.table_name
        );

        EXECUTE format(
            'CREATE POLICY tenant_scope ON core.%I FOR ALL TO PUBLIC '
            || 'USING (%I = NULLIF(current_setting(%L, true), %L)::uuid) '
            || 'WITH CHECK (%I = NULLIF(current_setting(%L, true), %L)::uuid)',
            item.table_name, item.tenant_key, 'app.organization_id', '',
            item.tenant_key, 'app.organization_id', ''
        );
    END LOOP;
END
$migration$;
