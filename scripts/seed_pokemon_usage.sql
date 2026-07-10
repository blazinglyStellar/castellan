-- Seed usage data for Pokemon provider
-- Backdated over 7 days (Jul 2–9, 2026)
-- Idempotent: ON CONFLICT (request_id) DO NOTHING
--
-- Run: psql "postgresql://postgres:1234@localhost:5432/castellan?sslmode=disable" -f scripts/seed_pokemon_usage.sql

DO $$
DECLARE
    v_provider_id     UUID := 'ed546883-6b49-479f-802a-9db743f6e9bf';
    v_endpoint_id     UUID := '06ec1ef5-723f-4f43-9bd8-910eb397f8f9';
    v_consumer_id     UUID := '67e39047-d85d-4669-a4bc-32a130fd7e8d';
    v_cost            NUMERIC(20,10) := 0.001;
    v_balance_after   NUMERIC(20,10);
    v_usage_id        UUID;

    -- Data arrays: 30 rows
    day_offsets   INT[]  := ARRAY[0,0,0,  1,1,1,1,1,  2,2,2,2,2,  3,3,3,  4,4,4,4,4,  5,5,5,5,  6,6,6,  7,7];
    hours         INT[]  := ARRAY[8,10,14, 9,11,13,15,18, 7,10,12,14,17, 8,10,12, 9,11,13,15,17, 8,10,12,14, 9,11,13, 7,9];
    minutes       INT[]  := ARRAY[15,42,3, 0,30,15,45,20, 30,0,15,50,5, 30,15,45, 0,20,40,10,30, 0,0,0,30, 30,15,0, 45,30];
    statuses      TEXT[] := ARRAY['completed','completed','completed', 'completed','completed','refunded','completed','completed', 'completed','completed','completed','completed','failed', 'completed','completed','completed', 'completed','completed','refunded','completed','completed', 'completed','completed','completed','failed', 'completed','completed','completed', 'completed','refunded'];
    codes         INT[]  := ARRAY[200,200,200, 200,200,502,200,200, 200,200,200,200,500, 200,200,200, 200,200,504,200,200, 200,200,200,503, 200,200,200, 200,408];
    latencies     INT[]  := ARRAY[120,95,180, 75,110,1200,90,65, 85,130,55,190,850, 100,140,70, 115,85,3000,95,150, 105,80,160,950, 110,45,135, 90,500];
    resp_sizes    INT[]  := ARRAY[1024,2048,512, 4096,2048,0,1024,8192, 2048,4096,1024,512,0, 2048,1024,4096, 2048,1024,0,8192,2048, 1024,4096,512,0, 2048,1024,4096, 2048,256];
BEGIN
    RAISE NOTICE 'Consumer user: %', v_consumer_id;

    FOR i IN 1 .. 30 LOOP
        v_usage_id := NULL;

        INSERT INTO usage_events (
            id, consumer_id, provider_id, endpoint_id,
            request_cost, currency, status_code, latency_ms, response_size,
            request_id, status, created_at
        ) VALUES (
            gen_random_uuid(),
            v_consumer_id,
            v_provider_id,
            v_endpoint_id,
            v_cost,
            'XLM',
            codes[i],
            latencies[i],
            resp_sizes[i],
            'pokemon-seed-' || i || '-' || gen_random_uuid()::TEXT,
            statuses[i]::usage_status,
            '2026-07-09 00:00:00+00'::TIMESTAMPTZ - (day_offsets[i] || ' days')::INTERVAL
                + (hours[i] || ' hours')::INTERVAL
                + (minutes[i] || ' minutes')::INTERVAL
        )
        ON CONFLICT (request_id) DO NOTHING
        RETURNING id INTO v_usage_id;

        IF v_usage_id IS NULL THEN
            CONTINUE;
        END IF;

        -- Ledger entry for completed/refunded events
        IF statuses[i] IN ('completed', 'reserved', 'refunded') THEN
            IF statuses[i] = 'refunded' THEN
                UPDATE users SET balance = balance + v_cost, account_updated_at = now()
                WHERE id = v_consumer_id
                RETURNING balance INTO v_balance_after;
            ELSE
                UPDATE users SET balance = balance - v_cost, account_updated_at = now()
                WHERE id = v_consumer_id
                RETURNING balance INTO v_balance_after;
            END IF;

            INSERT INTO ledger_entries (
                user_id, entry_type, amount, balance_after,
                currency, reference_id, reference_type, status, description, created_at
            ) VALUES (
                v_consumer_id,
                (CASE WHEN statuses[i] = 'refunded' THEN 'refund' ELSE 'deduction' END)::entry_type,
                CASE WHEN statuses[i] = 'refunded' THEN v_cost ELSE -v_cost END,
                v_balance_after,
                'XLM',
                v_usage_id,
                'usage_event',
                CASE WHEN statuses[i] = 'refunded' THEN 'cancelled' ELSE 'completed' END::ledger_status,
                CASE WHEN statuses[i] = 'refunded' THEN 'Refund for Pokemon API call' ELSE 'Pokemon API call: /v1/greet/pikachu' END,
                '2026-07-09 00:00:00+00'::TIMESTAMPTZ - (day_offsets[i] || ' days')::INTERVAL
                    + (hours[i] || ' hours')::INTERVAL
                    + (minutes[i] || ' minutes')::INTERVAL
            );
        END IF;
    END LOOP;

    RAISE NOTICE 'Done. Seeded 30 usage events for Pokemon provider.';
END $$;
