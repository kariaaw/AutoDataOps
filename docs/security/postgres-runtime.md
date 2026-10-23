# PostgreSQL Runtime Security

Status: Development / Not Production Ready

## Database Roles

- autodataops_bootstrap: administrative and migration operations.
- autodataops_runtime: non-login permission role.
- autodataops_service: restricted application login.

The application must never use the bootstrap superuser.

## Tenant Isolation

Migration 000002 enables PostgreSQL Row-Level Security.

Current policies rely on the session variable app.organization_id.

This variable can be changed by a SQL session.
Therefore, the current policies are not an independent
security boundary against arbitrary SQL execution.

Before production deployment:

- Authenticate and authorize every request.
- Derive tenant identity from a trusted principal.
- Establish tenant context inside controlled transactions.
- Prevent arbitrary SQL execution through application accounts.
- Validate cross-tenant isolation with adversarial tests.

## Runtime Restrictions

The initial runtime permissions only allow reading
core.organizations under RLS.

Additional permissions must be reviewed individually.

## Development Environment

Local PostgreSQL connections may use sslmode=disable
only on trusted loopback connections.

Production requires appropriate transport security,
secret management, access controls and security testing.

## Deployment Status

The current PostgreSQL runtime integration is intended
for development and controlled testing only.
