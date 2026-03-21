# ADR 003: Allowlist-based catalog/schema/table filtering for data governance

## Status
Accepted

## Context
Enterprise Trino clusters typically contain hundreds of catalogs, thousands of schemas, and millions of tables. Exposing all of these to an AI assistant creates two problems: (1) overwhelming the model with irrelevant metadata, degrading response quality, and (2) allowing access to sensitive data that should be restricted by organizational policy. Additionally, `SHOW CATALOGS/SCHEMAS/TABLES` queries on large Trino clusters can be slow, and filtering server-side avoids unnecessary data transfer.

## Decision
Three allowlist environment variables control what data the MCP server exposes:
- `TRINO_ALLOWED_CATALOGS`: comma-separated catalog names (e.g., `hive,postgresql`)
- `TRINO_ALLOWED_SCHEMAS`: comma-separated `catalog.schema` entries (e.g., `hive.analytics,hive.marts`)
- `TRINO_ALLOWED_TABLES`: comma-separated `catalog.schema.table` entries (e.g., `hive.analytics.users`)

When empty, no filtering is applied (all resources visible). When set, only matching entries are returned from list operations. All comparisons are case-insensitive via `strings.EqualFold()`.

Format validation happens at config load time: schemas must contain exactly 1 dot, tables exactly 2 dots. Malformed entries cause startup failure with a descriptive error message.

Filtering is applied post-query: Trino returns the full result set, and the client filters it in `filterCatalogs()`, `filterSchemas()`, and `filterTables()`. For `GetTableSchema`, the table parameter is resolved (supporting dotted notation like `catalog.schema.table`, `schema.table`, or just `table`) before the allowlist check.

**Code references:**
- Config parsing: `internal/config/config.go` — `parseAllowlist()`, `validateAllowlist()`, `logAllowlistConfiguration()`
- Filtering: `internal/trino/client.go` — `filterCatalogs()`, `filterSchemas()`, `filterTables()`
- Membership checks: `internal/trino/client.go` — `isCatalogAllowed()`, `isSchemaAllowed()`, `isTableAllowed()`
- Table resolution: `internal/trino/client.go` — `GetTableSchema()` resolves dotted table names before allowlist check
- Tests: `internal/trino/client_test.go` — `TestFilterCatalogs`, `TestFilterSchemas`, `TestFilterTables`, `TestIsCatalogAllowed`, `TestIsSchemaAllowed`, `TestIsTableAllowed`, `TestGetTableSchemaAllowlistLogic`
- Config tests: `internal/config/config_test.go` — `TestParseAllowlist`, `TestValidateAllowlist`, `TestNewTrinoConfigWithAllowlists`, `TestNewTrinoConfigMalformedAllowlist`
- Helm: `charts/mcp-trino/values.yaml` — `trino.allowlists.catalogs/schemas/tables`

## Consequences
- **Positive:** Operators can restrict AI assistant access to specific datasets without modifying Trino cluster permissions.
- **Positive:** Case-insensitive matching handles inconsistent casing between Trino connectors.
- **Positive:** Format validation at startup catches misconfigured allowlists before they silently filter everything.
- **Positive:** Post-query filtering is simple and reliable — no complex SQL rewriting needed.
- **Negative:** Post-query filtering means the full Trino query still executes — filtering does not reduce Trino-side load.
- **Negative:** Allowlists are static (env var based) — cannot be changed without restarting the server.
- **Negative:** Three separate allowlist levels can be confusing — a catalog allowlist does not automatically restrict schemas/tables within that catalog.

## Alternatives Considered
- **Trino-level access control:** Use Trino's built-in authorization (file-based, Ranger, OPA). Rejected because not all Trino deployments have fine-grained access control configured, and the MCP server should enforce its own boundaries.
- **Query rewriting:** Modify queries to add WHERE clauses or schema prefixes. Rejected due to complexity and risk of breaking complex queries.
- **Dynamic allowlists:** Store allowlists in a database or API. Rejected as over-engineering for the current use case — env vars align with the 12-factor app pattern used throughout the project.
