# ADR-0002: PostgreSQL Infrastructure

Status: Accepted for local development

## Decision

AutoDataOps uses PostgreSQL 18.6 as its
initial transactional database.

## Development Configuration

- Docker Compose project: autodataops-dev
- Database: autodataops
- Bootstrap user: autodataops_bootstrap
- Host: 127.0.0.1
- Development port: 55432
- Authentication: SCRAM-SHA-256 for TCP clients
- Persistent Docker named volume
- Health checks enabled

## Data Persistence

PostgreSQL 18 requires the data volume to
be mounted at /var/lib/postgresql.

The volume is managed independently of
the database container lifecycle.

## Security

The database is published only on localhost.

Database credentials are generated locally
and stored in an ignored .env file.

The bootstrap user is privileged.

Application runtime services must NOT
use this bootstrap identity in production.

Separate database roles with minimum
required permissions will be introduced.

Docker administrators can inspect container
environment variables. Production deployments
must use an approved secret-management system.

## Future Requirements

- Versioned SQL migrations
- Incident persistence
- Transition history persistence
- Optimistic concurrency control
- Transactional outbox
- Tenant isolation
- Least-privilege database roles
- Backup and restore verification
- Database observability
- Connection pooling
- Production authentication and TLS
- Recovery testing

## Limitations

This configuration is for local development.

It must not be deployed to an exposed
production environment without additional
security and reliability controls.
