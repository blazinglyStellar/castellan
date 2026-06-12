# Endpoint Registration — Interface Plan

## Flow

```
OpenAPI Spec (openapi.yaml)
         │
         ▼
   CLI Tool ──POST──► Castellan API ──INSERT──► api_endpoints (status: draft)
         │                  │
         │                  ▼
         │           Dashboard loads endpoints
         │                  │
         ▼                  ▼
   User sets prices    User toggles public
   & enables public    & sets prices
         │                  │
         └──────────────────┘
                     │
                     ▼
          api_endpoints (status: active)
          Gateway starts routing & billing
```

---

## 1. Backend API Endpoints

### Providers

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/providers` | Register a new provider |
| `GET` | `/api/v1/providers` | List own providers |
| `GET` | `/api/v1/providers/{id}` | Get provider details |
| `PUT` | `/api/v1/providers/{id}` | Update provider (name, base_url) |
| `DELETE` | `/api/v1/providers/{id}` | Deactivate provider |

### Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/providers/{id}/endpoints` | List endpoints for a provider |
| `PUT` | `/api/v1/providers/{id}/endpoints/{eid}` | Update endpoint (price, status) |
| `POST` | `/api/v1/providers/{id}/endpoints/bulk` | Bulk-import from CLI (status = draft) |
| `PATCH` | `/api/v1/providers/{id}/endpoints/{eid}/status` | Toggle active/inactive/draft |

`POST /bulk` accepts:
```json
{
  "endpoints": [
    { "route": "/search", "method": "GET" },
    { "route": "/search", "method": "POST" },
    { "route": "/images/{id}", "method": "GET" }
  ],
  "auto_publish": false   // if true, sets status=active with default price
}
```

### Endpoint status model

```
draft ──► active ◄──► inactive
```

- **draft**: Imported from CLI, not yet priced or public. Excluded from gateway routing.
- **active**: Priced and public. Gateway routes and bills requests.
- **inactive**: Was active but temporarily disabled by the provider. No routing.

---

## 2. CLI Tool (`cmd/cli/`)

### Command structure

```
castellan import [--provider-id <id> | --provider-name <name>] [flags] ./openapi.yaml

castellan login         # authenticate, store API key in ~/.castellan/config
castellan providers     # list providers
castellan endpoints     # list endpoints for a provider
```

### Flags for `import`

| Flag | Default | Description |
|---|---|---|
| `--provider-id` | — | Target provider ID (mutually exclusive with --provider-name) |
| `--provider-name` | — | Create/lookup provider by name |
| `--dry-run` | false | Print parsed endpoints without uploading |
| `--publish` | false | Auto-publish with a default price (0.0001 XLM) |
| `--default-price` | 0.0001 | Default price when --publish is set |

### OpenAPI parsing

Supports **OpenAPI 3.0 and 3.1** (JSON and YAML). Extracts:

| OpenAPI field | → | `api_endpoints` column |
|---|---|---|
| `paths.{path}` | → | `route` |
| `paths.{path}.{method}` | → | `method` |
| `paths.{path}.{method}.operationId` | → | `label` (stored in description) |
| `paths.{path}.{method}.summary` | → | `description` |
| `paths.{path}.{method}.parameters` | → | Stored as JSON metadata for future use |

**Skip rules:**
- Ignore parameters-only paths (no operation)
- Ignore paths with unsupported methods (CONNECT, TRACE etc.)
- Warn on duplicate routes within the same spec

### Authentication

- `castellan login` prompts for API key, stores it in `~/.castellan/config`
- All API calls include `Authorization: Bearer ca_xxxxx`
- The API key belongs to the user's account; providers are created under that user

### Implementation outline

```go
// cmd/cli/main.go — cobra root command
// cmd/cli/import.go — "import" subcommand, OpenAPI parsing + upload
// cmd/cli/login.go  — "login" subcommand, saves API key to config file
// internal/cli/     — shared CLI helpers
// internal/openapi/ — OpenAPI spec parser
//   ├── parser.go       — loads YAML/JSON, extracts paths
//   ├── parser_test.go  — test with sample specs
//   └── types.go        — OpenAPI 3.x struct subset
```

### OpenAPI parse example

Given this spec:
```yaml
openapi: 3.0.0
paths:
  /weather:
    get:
      summary: Get current weather
      operationId: getWeather
  /weather/forecast:
    get:
      summary: Get weather forecast
```

CLI outputs:
```
Parsed 2 endpoints from openapi.yaml
  POST /api/v1/providers/p_abc/endpoints/bulk

Dry-run: 2 endpoints would be uploaded
  GET  /weather
  GET  /weather/forecast
```

---

## 3. Dashboard (Next.js)

### Pages

| Route | Page | Description |
|---|---|---|
| `/login` | Login | API key login (stores token, redirects to dashboard) |
| `/providers` | Provider list | List all providers, button to add new |
| `/providers/new` | New provider | Form: name, base URL |
| `/providers/{id}` | Provider detail | Tabs: Endpoints, Analytics, Settings |
| `/providers/{id}/endpoints` | Endpoint manager | Table of all endpoints |
| `/providers/{id}/endpoints/{eid}` | Edit endpoint | Set price, toggle status |

### Endpoint manager table

| Route | Method | Price | Status | Actions |
|---|---|---|---|---|
| /weather | GET | 0.0001 XLM | Active ✓ | Edit | Disable |
| /weather/forecast | GET | _not set_ | Draft | Edit | Publish |
| /search | POST | _not set_ | Draft | Edit | Publish |
| /images/{id} | GET | 0.0005 XLM | Active ✓ | Edit | Disable |
| /users | DELETE | — | Draft | _needs method config_ |

- Draft rows are visually distinct (greyed out, badge)
- Click row to expand and set price + toggle public

### Stack

| Layer | Choice | Reason |
|---|---|---|
| Framework | Next.js 14+ (App Router) | Per PRD, SSR for dashboard pages |
| UI | shadcn/ui + Tailwind | Per PRD |
| API client | React Query / SWR | Caching, refetching endpoint lists |
| Auth | API key in header, stored in cookie/localStorage | Matches the gateway auth model |

---

## 4. Data model impact

The existing `api_endpoints` table needs one addition:

```sql
ALTER TABLE api_endpoints ADD COLUMN description TEXT;
ALTER TABLE api_endpoints ADD COLUMN openapi_operation_id TEXT;
```

The status enum expands:

```sql
ALTER TYPE endpoint_status ADD VALUE 'draft' BEFORE 'active';
```

---

## 5. Implementation order

| Step | What | Depends on |
|---|---|---|
| 1 | Backend: provider CRUD endpoints | — |
| 2 | Backend: endpoint CRUD + bulk import | Step 1 |
| 3 | Backend: auth middleware (API key check) | — |
| 4 | CLI: `login` command (save API key) | Step 3 |
| 5 | CLI: `import` command (OpenAPI parse + upload) | Steps 2, 4 |
| 6 | Dashboard: scaffold Next.js + shadcn/ui | — |
| 7 | Dashboard: login page | Step 3 |
| 8 | Dashboard: provider list + new provider forms | Step 1 |
| 9 | Dashboard: endpoint manager table | Step 2 |
| 10 | Dashboard: price editing + publish toggles | Step 2 |

---

## 6. Open questions

1. **Auto-create provider from CLI?** — `--provider-name "My API"` creates a provider on the fly if it doesn't exist, or is `--provider-id` required?
2. **How does the CLI get an API key?** — Does the user generate one from the Dashboard, or does `castellan login` also handle registration (create user + key)?
3. **Public vs private endpoints** — "public" means the endpoint is enabled for routing, or does "public" mean it appears in a future marketplace/discovery feature?
