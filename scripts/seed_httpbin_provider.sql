WITH ins_provider AS (
    INSERT INTO providers (owner_id, name, base_url, description, status)
    SELECT id, 'httpbin', 'https://httpbin.org', 'E2E test downstream', 'active'
    FROM users WHERE email = 'seed-provider@castellan.local'
    ON CONFLICT (name) DO NOTHING RETURNING id
)
INSERT INTO api_endpoints (provider_id, route, method, price_amount, rate_limit, description)
SELECT (SELECT id FROM ins_provider), *
FROM (VALUES
    ('/post',       'POST',  0.05, 30, 'Echo POST'),
    ('/get',        'GET',   0.01, 60, 'Echo GET'),
    ('/status/200', 'GET',   0.01, 60, '200 OK test'),
    ('/status/500', 'GET',   0.01, 60, '500 failure test')
) AS t(route, method, price_amount, rate_limit, description)
WHERE EXISTS (SELECT 1 FROM ins_provider)
ON CONFLICT (provider_id, route, method) DO NOTHING;