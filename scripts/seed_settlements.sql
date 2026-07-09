-- Seed settlement batches + entries for all 4 providers (pokemon + 3 Go-seeded)
-- Spans Jul 2–9, 2026. 4 batches: 3 completed, 1 pending.
-- Includes provider-side ledger entries (settlement payouts to provider account).
-- Idempotent: skips if settlement_batches already has > 3 rows (Go seed base).
--
-- Prerequisites:
--   1. Go seed has been run (creates seed-provider@castellan.local users, providers, accounts)
--   2. scripts/seed_pokemon_usage.sql has been run (creates pokemon provider + usage)
--
-- Run: psql "postgresql://postgres:1234@localhost:5432/castellan?sslmode=disable" -f scripts/seed_settlements.sql

DO $$
DECLARE
    v_owner_id          UUID;
    v_pokemon_id        UUID := 'ed546883-6b49-479f-802a-9db743f6e9bf';
    v_weather_id        UUID;
    v_ai_id             UUID;
    v_blockchain_id     UUID;
    v_owner_account_id  UUID;
    v_batch_count       INT;
    v_batch_id          UUID;
    v_balance_after     NUMERIC(20,10);

    -- 4 batches
    batch_statuses    TEXT[]  := ARRAY['completed', 'completed', 'completed', 'pending'];
    batch_totals      TEXT[]  := ARRAY['1.21', '0.84', '1.36', '0.54'];
    batch_tx_hashes   TEXT[]  := ARRAY[
        'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1',
        'b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2',
        'c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3',
        NULL
    ];
    batch_days_ago    INT[]   := ARRAY[6, 4, 2, 0];

    -- Per-provider amounts per batch: [batch][provider]
    -- Providers: 1=pokemon, 2=weather-api, 3=ai-inference, 4=blockchain-node
    prov_amounts      TEXT[][];
    prov_wallets      TEXT[]  := ARRAY[
        'GCVXL5SEEDPOKEMONWALLETADDRESSPLACEHOLDER12345',
        'GB7YQ4SEEDWEATHERWALLETADDRESSPLACEHOLDER78901',
        'GA4N6BSEEDAIWALLETADDRESSPLACEHOLDER23456789012',
        'GDR8A0SEEDBLOCKCHAINWALLETADDRESSPLACEHOLDER67890'
    ];
    prov_ids          UUID[];

    v_provider_id     UUID;
    v_wallet          TEXT;
    v_idx             INT;
    v_prov_idx        INT;
BEGIN
    -- Look up Go-seeded entities
    SELECT id INTO v_owner_id FROM users WHERE email = 'seed-provider@castellan.local';
    IF v_owner_id IS NULL THEN
        RAISE EXCEPTION 'seed-provider user not found. Run the Go seed first.';
    END IF;

    SELECT id INTO v_weather_id    FROM providers WHERE owner_id = v_owner_id AND name = 'weather-api';
    SELECT id INTO v_ai_id         FROM providers WHERE owner_id = v_owner_id AND name = 'ai-inference';
    SELECT id INTO v_blockchain_id FROM providers WHERE owner_id = v_owner_id AND name = 'blockchain-node';

    IF v_weather_id IS NULL OR v_ai_id IS NULL OR v_blockchain_id IS NULL THEN
        RAISE EXCEPTION 'Go-seeded providers not found. Run the Go seed first.';
    END IF;

    SELECT id INTO v_owner_account_id FROM accounts WHERE owner_id = v_owner_id;
    IF v_owner_account_id IS NULL THEN
        RAISE EXCEPTION 'Provider account not found. Run the Go seed first.';
    END IF;

    -- Idempotency: skip if we already have more than the Go seed's 3 batches
    SELECT COUNT(*) INTO v_batch_count FROM settlement_batches;
    IF v_batch_count > 3 THEN
        RAISE NOTICE 'Settlement batches already exist (%). Skipping seed.', v_batch_count;
        RETURN;
    END IF;

    prov_ids := ARRAY[v_pokemon_id, v_weather_id, v_ai_id, v_blockchain_id];

    prov_amounts := ARRAY[
        ARRAY['0.007', '0.45', '0.50', '0.25'],
        ARRAY['0.007', '0.35', '0.30', '0.18'],
        ARRAY['0.007', '0.50', '0.55', '0.30'],
        ARRAY['0.004', '0.20', '0.22', '0.12']
    ];

    FOR v_idx IN 1 .. 4 LOOP
        INSERT INTO settlement_batches (
            status, total_amount, currency, entry_count, tx_hash, completed_at
        ) VALUES (
            batch_statuses[v_idx]::batch_status,
            batch_totals[v_idx]::NUMERIC(20,10),
            'XLM',
            4,
            NULLIF(batch_tx_hashes[v_idx], ''),
            CASE WHEN batch_statuses[v_idx] = 'completed' THEN
                '2026-07-09 00:00:00+00'::TIMESTAMPTZ - (batch_days_ago[v_idx] || ' days')::INTERVAL
            ELSE NULL END
        )
        RETURNING id INTO v_batch_id;

        RAISE NOTICE 'Batch %: % (%, % XLM)',
            v_idx, batch_statuses[v_idx],
            COALESCE(substring(batch_tx_hashes[v_idx] FROM 1 FOR 12), '<pending>'),
            batch_totals[v_idx];

        FOR v_prov_idx IN 1 .. 4 LOOP
            v_provider_id := prov_ids[v_prov_idx];
            v_wallet      := prov_wallets[v_prov_idx];

            INSERT INTO settlement_entries (
                batch_id, provider_id, amount, currency, wallet_address, status
            ) VALUES (
                v_batch_id,
                v_provider_id,
                prov_amounts[v_idx][v_prov_idx]::NUMERIC(20,10),
                'XLM',
                v_wallet,
                batch_statuses[v_idx]::settlement_entry_status
            );
        END LOOP;

        -- Credit provider owner account for completed batches
        IF batch_statuses[v_idx] = 'completed' THEN
            UPDATE accounts
            SET balance = balance + batch_totals[v_idx]::NUMERIC(20,10),
                updated_at = now()
            WHERE id = v_owner_account_id
            RETURNING balance INTO v_balance_after;

            INSERT INTO ledger_entries (
                account_id, entry_type, amount, balance_after,
                currency, reference_id, reference_type, status, description, created_at
            ) VALUES (
                v_owner_account_id,
                'settlement'::entry_type,
                batch_totals[v_idx]::NUMERIC(20,10),
                v_balance_after,
                'XLM',
                v_batch_id,
                'settlement_batch',
                'completed'::ledger_status,
                'Settlement payout batch #' || v_idx,
                '2026-07-09 00:00:00+00'::TIMESTAMPTZ - (batch_days_ago[v_idx] || ' days')::INTERVAL
            );

            RAISE NOTICE '  Account % credited +%s XLM (balance: %)',
                substring(v_owner_account_id::TEXT, 1, 8),
                batch_totals[v_idx],
                v_balance_after;
        END IF;
    END LOOP;

    RAISE NOTICE 'Done. Seeded 4 settlement batches (3 completed + 1 pending).';
END $$;
