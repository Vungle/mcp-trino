# ADR 002: OAuth 2.1 authentication via reusable oauth-mcp-proxy library

## Status
Accepted

## Context
MCP servers exposed over HTTP need authentication to prevent unauthorized access to Trino data. The MCP specification defines OAuth 2.1 as the standard authentication mechanism. Rather than implementing OAuth directly in mcp-trino, the team needed a reusable solution that could be shared across multiple MCP servers.

## Decision
OAuth 2.1 is provided by the external `github.com/Vungle/oauth-mcp-proxy` library (v1.0.3), integrated as a Go dependency rather than a sidecar proxy. The library handles token issuance, validation, OIDC discovery, and RFC 8414 metadata endpoints.

Two operational modes are supported:
- **Native mode** (`OAUTH_MODE=native`): Client-driven OIDC flow. The MCP client handles the OAuth dance directly with the identity provider. The server validates tokens using OIDC discovery.
- **Proxy mode** (`OAUTH_MODE=proxy`): Server-driven flow. The oauth-mcp-proxy library acts as an authorization server, managing the full OAuth flow including authorization endpoints, token exchange, and callback handling.

Four identity providers are supported: `hmac` (symmetric JWT signing for development), `okta`, `google`, and `azure` (all using OIDC discovery).

`OAUTH_ENABLED` is the single boolean source of truth — when `false`, all OAuth configuration is ignored and endpoints are unauthenticated.

**Code references:**
- Integration point: `internal/mcp/server.go` — `createMCPServer()` creates `oauth.NewServer()` and applies `oauthServer.Middleware()` via `WithToolHandlerMiddleware()`
- Config mapping: `internal/mcp/server.go` — `trinoConfigToOAuthConfig()` maps Trino config to `oauth.Config`
- HTTP registration: `internal/mcp/server.go` — `ServeHTTP()` calls `s.oauthServer.RegisterHandlers(mux)` for well-known endpoints
- Bearer token enforcement: `internal/mcp/server.go` — `createMCPHandler()` checks `Authorization: Bearer` header
- User extraction: `internal/trino/client.go` — `getQueryUsername()` uses `oauth.GetUserFromContext()` for Trino query tagging
- Config: `internal/config/config.go` — OAuth fields (`OAuthEnabled`, `OAuthMode`, `OAuthProvider`, `JWTSecret`, OIDC fields)
- Tests: `internal/config/oauth_test.go` — `TestOAuthModeConfiguration`, `TestOAuthAllowedRedirectsConfiguration`, `TestOAuthProxyModeValidation`

## Consequences
- **Positive:** Reusable library means OAuth implementation is maintained once and shared across MCP servers. Bug fixes and security patches propagate to all consumers.
- **Positive:** Multi-provider support (HMAC for dev, Okta/Google/Azure for production) covers common enterprise identity setups.
- **Positive:** Server-wide middleware application ensures all tools are protected without per-tool configuration.
- **Negative:** Dependency on external library means OAuth behavior changes require updating the library version. Config validation is partially delegated to the library.
- **Negative:** `OAUTH_REDIRECT_URI` was deprecated in favor of `OAUTH_ALLOWED_REDIRECT_URIS` — backward compatibility code adds maintenance burden.

## Alternatives Considered
- **Sidecar proxy:** Run oauth-mcp-proxy as a separate container/process in front of mcp-trino. Rejected because it adds deployment complexity and latency for every request.
- **Built-in OAuth:** Implement OAuth directly in mcp-trino. Rejected because it would duplicate effort across multiple MCP servers and the OAuth specification is complex to implement correctly.
- **API key authentication:** Simple shared secret. Rejected because it does not meet MCP specification requirements for OAuth 2.1 and lacks user identity propagation.
