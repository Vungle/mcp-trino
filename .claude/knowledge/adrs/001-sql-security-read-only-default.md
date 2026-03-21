# ADR 001: Read-only query enforcement as default security posture

## Status
Accepted

## Context
mcp-trino exposes a Trino distributed SQL engine to AI assistants via MCP tools. AI assistants generate SQL queries conversationally, creating a significant SQL injection risk if write operations are allowed. The `execute_query` tool accepts arbitrary SQL strings from untrusted input (the AI model's output), making it essential to restrict what queries can execute by default.

## Decision
All SQL queries are validated through `isReadOnlyQuery()` in `internal/trino/client.go` before execution. The function uses a whitelist approach: only queries starting with `SELECT`, `SHOW`, `DESCRIBE`, `EXPLAIN`, or `WITH` (CTEs) are allowed. Write operations (`INSERT`, `UPDATE`, `DELETE`, `CREATE`, `DROP`, `ALTER`, `TRUNCATE`, `MERGE`, `GRANT`, `REVOKE`, `COMMIT`, `ROLLBACK`, `CALL`, `EXECUTE`, `REFRESH`, `SET`, `RESET`) are blocked using word-boundary regex matching.

Three layers of sanitization protect against bypass:
1. **`sanitizeQueryForKeywordDetection()`** strips string literals (`'...'`), quoted identifiers (`"..."`), backtick identifiers, single-line comments (`--`), and multi-line comments (`/* */`) before keyword scanning, preventing false positives from embedded keywords.
2. **Semicolon blocking** prevents multi-statement injection (`SELECT 1; DROP TABLE x`).
3. **`isAllowedReadOnlyPattern()`** explicitly permits `SHOW CREATE TABLE/VIEW/SCHEMA` despite containing the `CREATE` keyword.

Write queries can be enabled via `TRINO_ALLOW_WRITE_QUERIES=true`, which logs a prominent warning at startup.

**Code references:**
- Security gate: `internal/trino/client.go` — `isReadOnlyQuery()`, `isAllowedReadOnlyPattern()`, `sanitizeQueryForKeywordDetection()`
- Config flag: `internal/config/config.go` — `AllowWriteQueries` field, `TRINO_ALLOW_WRITE_QUERIES` env var
- Enforcement point: `internal/trino/client.go` — `ExecuteQuery()` checks `!c.config.AllowWriteQueries && !isReadOnlyQuery(query)`
- Tests: `internal/trino/client_test.go` — `TestImprovedIsReadOnlyQuery` (33 test cases covering word boundaries, SHOW CREATE, injection attempts)

## Consequences
- **Positive:** Default-secure posture. AI-generated queries cannot modify or destroy data without explicit operator opt-in.
- **Positive:** Whitelist approach is safer than blacklist — unknown query types are rejected by default.
- **Positive:** String literal and comment stripping prevents embedding write keywords in quoted strings to bypass detection.
- **Negative:** The regex-based approach cannot handle all edge cases in Trino's full SQL grammar. A determined attacker with write access enabled could potentially craft bypass queries.
- **Negative:** `SHOW CREATE TABLE` required special-casing because it contains the `CREATE` keyword — each new read-only pattern containing write keywords needs explicit handling.

## Alternatives Considered
- **Trino-side RBAC:** Rely on Trino's built-in access control to restrict write operations. Rejected because it requires Trino cluster configuration changes and the MCP server should enforce its own security boundary regardless of backend permissions.
- **SQL parser (AST-based):** Use a proper SQL parser to validate query types. Rejected due to complexity — Trino SQL has many dialects and extensions. The regex approach covers the common cases while keeping the codebase simple.
- **Blacklist approach:** Block known write keywords only. Rejected because new SQL features or Trino extensions could introduce write operations not in the blacklist, making whitelist safer.
