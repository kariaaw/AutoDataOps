
# ADR-0003: AutoDataOps Core Database Schema

Status: Accepted for initial local development.

## Database

PostgreSQL 18.6

## Schema

core

## Tables

1. organizations
2. pipelines
3. incidents
4. incident_transitions
5. verification_evidence
6. outbox_events
7. schema_migrations

## Migration Policy

Migrations are versioned.

The migration runner verifies SHA-256 checksums.

Applied migrations must never be edited.

New database changes require a new migration.

## Transaction Safety

The initial migration is applied in one transaction.

The smoke tests execute in a transaction
that is rolled back after verification.

## Tenant Relationships

Composite foreign keys prevent incidents
from referencing pipelines owned by
different organizations.

This does not yet provide complete tenant
isolation or authorization.

## Incident Lifecycle

The Incident Domain defines valid state changes.

Database constraints provide additional
structural protection.

The database does not yet independently
enforce the complete state machine.

## Recovery Evidence

Resolution transitions require an evidence
reference belonging to the same incident.

Evidence authenticity and successful validation
must be enforced by trusted application services.

## Security Limitations

Row-Level Security is not yet configured.

Dedicated runtime database roles are not yet created.

The bootstrap database user is privileged.

The application must never use the privileged
bootstrap identity in production.

## Future Engineering Requirements

- Least-privilege database roles
- PostgreSQL Row-Level Security
- Tenant isolation tests
- Optimistic concurrency control
- Atomic incident state transitions
- Transactional outbox publishing
- Append-only audit enforcement
- Trusted verification evidence
- Backup and restore tests
- Database performance benchmarks
- Automated migration validation

## Production Readiness

This database schema is an initial foundation.

Production readiness is not claimed.

