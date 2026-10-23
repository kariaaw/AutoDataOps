#!/usr/bin/env bash
set -euo pipefail

CONTAINER="${AUTODATAOPS_PG_CONTAINER:-autodataops-dev-postgres-1}"

docker exec -i "$CONTAINER" \
  psql -U autodataops_bootstrap -d postgres \
  -X -v ON_ERROR_STOP=1 <<'SQL'

DO $bootstrap$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_roles
        WHERE rolname = 'autodataops_runtime'
    ) THEN
        CREATE ROLE autodataops_runtime
        NOLOGIN NOSUPERUSER NOCREATEDB
        NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_roles
        WHERE rolname = 'autodataops_service'
    ) THEN
        CREATE ROLE autodataops_service
        NOLOGIN NOSUPERUSER NOCREATEDB
        NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_roles
        WHERE rolname = 'autodataops_runtime'
          AND NOT rolcanlogin
          AND NOT rolsuper
          AND NOT rolcreatedb
          AND NOT rolcreaterole
          AND NOT rolreplication
          AND NOT rolbypassrls
          AND NOT rolinherit
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_roles
        WHERE rolname = 'autodataops_service'
          AND NOT rolsuper
          AND NOT rolcreatedb
          AND NOT rolcreaterole
          AND NOT rolreplication
          AND NOT rolbypassrls
          AND rolinherit
    ) THEN
        RAISE EXCEPTION 'Unsafe AutoDataOps role attributes';
    END IF;
END
$bootstrap$;

GRANT autodataops_runtime TO autodataops_service
WITH INHERIT TRUE, SET FALSE;
SQL
