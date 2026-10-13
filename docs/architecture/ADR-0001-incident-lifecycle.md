# ADR-0001: Incident Lifecycle

Status: Accepted for initial implementation

## Context

AutoDataOps requires consistent incident lifecycle
management across collectors, APIs, and recovery services.

## Decision

An Incident is a domain aggregate with:

- A randomly generated UUID v4
- Validated severity
- A controlled lifecycle
- UTC timestamps
- A history of status transitions
- Explicit resolution verification references

## Lifecycle

Detected -> Investigating
Detected -> Dismissed

Investigating -> Mitigating
Investigating -> Dismissed

Mitigating -> Investigating
Mitigating -> Resolved

Resolved -> Investigating
Resolved -> Closed

Closed and Dismissed are terminal.

## Safety Requirements

A transition to Resolved requires a positive
verification attestation.

The domain currently checks the attestation's
structure, success flag, and timing.

This alone does NOT authenticate the evidence.

A trusted application service must verify the
evidence against persistent validation records
before permitting the transition.

All state changes will eventually be stored
transactionally in PostgreSQL.

Authorization, optimistic concurrency control,
tenant isolation, and durable audit history
must be implemented before production use.

## Future Requirements

- Durable incident repository
- Organization and tenant identifiers
- Incident correlation and deduplication
- Database-backed verification evidence
- Optimistic concurrency
- RBAC and authorization
- Reliable event publishing
- Immutable or tamper-evident audit storage
- OpenTelemetry and OpenLineage integration

## Scope

This ADR covers the initial domain model only.

No production readiness is claimed.
