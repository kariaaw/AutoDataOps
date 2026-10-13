
BEGIN;

DO $test$

DECLARE

    org_a uuid := gen_random_uuid();

    org_b uuid := gen_random_uuid();

    pipe_a uuid := gen_random_uuid();

    incident_a uuid := gen_random_uuid();

    proof_a uuid := gen_random_uuid();

BEGIN

    -- Create two organizations.

    INSERT INTO core.organizations(
        id,
        slug,
        display_name
    )
    VALUES
        (
            org_a,
            'smoke-tenant-alpha',
            'Tenant A'
        ),
        (
            org_b,
            'smoke-tenant-bravo',
            'Tenant B'
        );

    -- Create a pipeline for organization A.

    INSERT INTO core.pipelines(
        id,
        organization_id,
        source_type,
        external_key,
        display_name
    )
    VALUES (
        pipe_a,
        org_a,
        'airflow',
        'billing.close',
        'Billing Daily Close'
    );

    -- Create a valid incident.

    INSERT INTO core.incidents(
        id,
        organization_id,
        pipeline_id,
        source,
        summary,
        severity,
        detected_at
    )
    VALUES (
        incident_a,
        org_a,
        pipe_a,
        'airflow',
        'Test failure',
        'high',
        now()
    );

    -- Register successful verification evidence.

    INSERT INTO core.verification_evidence(
        id,
        organization_id,
        incident_id,
        validator,
        passed,
        checked_at
    )
    VALUES (
        proof_a,
        org_a,
        incident_a,
        'test-validator',
        true,
        now()
    );

    -- Valid transition 1.

    INSERT INTO core.incident_transitions(
        organization_id,
        incident_id,
        version_after,
        from_status,
        to_status,
        reason,
        occurred_at
    )
    VALUES (
        org_a,
        incident_a,
        2,
        'detected',
        'investigating',
        'investigating test event',
        now()
    );

    -- Valid transition 2.

    INSERT INTO core.incident_transitions(
        organization_id,
        incident_id,
        version_after,
        from_status,
        to_status,
        reason,
        occurred_at
    )
    VALUES (
        org_a,
        incident_a,
        3,
        'investigating',
        'mitigating',
        'mitigation started',
        now()
    );

    -- Valid transition 3.

    INSERT INTO core.incident_transitions(
        organization_id,
        incident_id,
        version_after,
        from_status,
        to_status,
        reason,
        verification_id,
        occurred_at
    )
    VALUES (
        org_a,
        incident_a,
        4,
        'mitigating',
        'resolved',
        'validated test event',
        proof_a,
        now()
    );

    UPDATE core.incidents

    SET
        status = 'resolved',
        version = 4,
        updated_at = now()

    WHERE
        id = incident_a
        AND organization_id = org_a;

    -- Create an outbox event.

    INSERT INTO core.outbox_events(
        organization_id,
        aggregate_type,
        aggregate_id,
        event_type,
        payload
    )
    VALUES (
        org_a,
        'incident',
        incident_a,
        'incident.detected',
        '{"test":true}'::jsonb
    );

    -- Verify transition count.

    IF (
        SELECT count(*)
        FROM core.incident_transitions
        WHERE incident_id = incident_a
    ) <> 3

    THEN

        RAISE EXCEPTION
            'Expected incident transitions missing';

    END IF;

    -- Test tenant relationship protection.

    BEGIN

        INSERT INTO core.incidents(
            organization_id,
            pipeline_id,
            source,
            summary,
            severity,
            detected_at
        )

        VALUES (
            org_b,
            pipe_a,
            'airflow',
            'Illegal tenant crossing',
            'high',
            now()
        );

        RAISE EXCEPTION
            'Cross-tenant incident unexpectedly accepted';

    EXCEPTION

        WHEN foreign_key_violation THEN

            RAISE NOTICE
                'PASS: Cross-tenant relation rejected';

    END;

    -- Test resolution evidence constraint.

    BEGIN

        INSERT INTO core.incident_transitions(
            organization_id,
            incident_id,
            version_after,
            from_status,
            to_status,
            reason,
            occurred_at
        )

        VALUES (
            org_a,
            incident_a,
            5,
            'mitigating',
            'resolved',
            'No verification evidence',
            now()
        );

        RAISE EXCEPTION
            'Resolution without evidence accepted';

    EXCEPTION

        WHEN check_violation THEN

            RAISE NOTICE
                'PASS: Resolution without evidence rejected';

    END;

    -- Test severity validation.

    BEGIN

        INSERT INTO core.incidents(
            organization_id,
            pipeline_id,
            source,
            summary,
            severity,
            detected_at
        )

        VALUES (
            org_a,
            pipe_a,
            'airflow',
            'Invalid severity',
            'urgent',
            now()
        );

        RAISE EXCEPTION
            'Invalid severity unexpectedly accepted';

    EXCEPTION

        WHEN check_violation THEN

            RAISE NOTICE
                'PASS: Invalid severity rejected';

    END;

END;

$test$;

-- Remove every synthetic test record.

ROLLBACK;

