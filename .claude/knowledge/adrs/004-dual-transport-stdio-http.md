# ADR 004: Dual transport support — STDIO for desktop, HTTP StreamableHTTP for remote

## Status
Accepted

## Context
MCP clients operate in two distinct environments: desktop applications (Claude Desktop, VS Code extensions) that communicate via subprocess STDIO, and remote/server deployments where clients connect over HTTP. The MCP specification supports both transport types. mcp-trino needed to support both without duplicating the tool implementation layer.

## Decision
Transport is selected at startup via `MCP_TRANSPORT` environment variable (`stdio` or `http`, default: `stdio`). Both transports share the same `MCPServer` instance and tool registrations.

**STDIO transport:** Uses `mcpserver.ServeStdio()` from mcp-go library. No HTTP server is started. Best for desktop MCP clients that spawn the server as a subprocess.

**HTTP transport:** Uses `mcpserver.NewStreamableHTTPServer()` from mcp-go library with `WithEndpointPath("/mcp")`. The modern StreamableHTTP endpoint is served at `/mcp`. For backward compatibility, the same handler is also mounted at `/sse` (legacy SSE-based transport).

Additional HTTP endpoints:
- `/status` — health check returning `{"status":"ok","version":"..."}` (used by Kubernetes probes)
- `/.well-known/oauth-authorization-server` — OAuth metadata (when OAuth enabled, registered by oauth-mcp-proxy)
- `/oauth/callback` — OAuth callback (when OAuth enabled)

HTTPS is supported via `HTTPS_CERT_FILE` and `HTTPS_KEY_FILE` environment variables. Graceful shutdown with 30-second timeout handles SIGINT/SIGTERM signals.

**Code references:**
- Transport selection: `cmd/main.go` — `switch transport` block selects `ServeStdio()` or `ServeHTTP()`
- STDIO: `internal/mcp/server.go` — `ServeStdio()` delegates to `mcpserver.ServeStdio()`
- HTTP setup: `internal/mcp/server.go` — `ServeHTTP()` configures StreamableHTTP, mux, CORS, OAuth handlers
- Dual endpoints: `internal/mcp/server.go` — `mux.HandleFunc("/mcp", mcpHandler)` and `mux.HandleFunc("/sse", mcpHandler)`
- Graceful shutdown: `internal/mcp/server.go` — `handleSignals()` with 30s `context.WithTimeout`
- Docker Compose: `docker-compose.yml` — `MCP_TRANSPORT=http` for containerized deployment
- Helm: `charts/mcp-trino/values.yaml` — `mcpServer.transport: "http"` for Kubernetes

## Consequences
- **Positive:** Single binary serves both desktop and server deployments without code changes.
- **Positive:** Tool implementations are transport-agnostic — registered once on the shared MCPServer.
- **Positive:** `/sse` backward compatibility means existing clients using the legacy SSE transport continue to work.
- **Positive:** HTTPS support with graceful shutdown makes the server production-ready.
- **Negative:** STDIO mode cannot serve multiple concurrent clients — it is inherently single-session.
- **Negative:** The `/sse` endpoint serves StreamableHTTP, not actual SSE — this is a naming compatibility shim, not true SSE transport.

## Alternatives Considered
- **HTTP only:** Remove STDIO support and require all clients to use HTTP. Rejected because desktop MCP clients (Claude Desktop) expect subprocess STDIO communication.
- **Separate binaries:** Build two binaries, one for each transport. Rejected because it doubles the build and release surface area for no benefit.
- **Auto-detection:** Detect whether stdin is a TTY to choose transport. Rejected because it is fragile and some CI/deployment environments have unexpected TTY states.
