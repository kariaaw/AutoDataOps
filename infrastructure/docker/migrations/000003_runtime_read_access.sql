-- AutoDataOps: initial least-privilege runtime access.
-- Requires autodataops_runtime to be provisioned.
-- Privileges are database-local.
-- Tenant isolation is enforced by migration 000002.

GRANT USAGE ON SCHEMA core TO autodataops_runtime;

GRANT SELECT ON TABLE core.organizations
TO autodataops_runtime;
