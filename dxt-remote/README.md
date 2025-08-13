# Trino MCP Remote Extension

This Desktop Extension (DXT) provides easy one-click access to remote Trino MCP servers for SQL analytics and data exploration using OAuth authentication.

## What This Extension Provides

- **SQL Query Execution**: Run SELECT, SHOW, DESCRIBE, and EXPLAIN queries against Trino databases
- **Schema Discovery**: Browse catalogs, schemas, and tables
- **Table Inspection**: Get detailed table structure and column information
- **Remote Access**: Connect to OAuth-enabled Trino MCP servers without local setup
- **One-Click Install**: Simple DXT installation process with automatic configuration

## Available Tools

1. `execute_query` - Execute SQL queries with security restrictions (read-only by default)
2. `list_catalogs` - Discover available data catalogs
3. `list_schemas` - List schemas within catalogs  
4. `list_tables` - List tables within schemas
5. `get_table_schema` - Retrieve table structure and column details

## How It Works

This DXT uses [mcp-remote](https://github.com/geelen/mcp-remote) to proxy MCP requests to your remote Trino server. The extension:

1. **Handles OAuth Authentication**: Automatically manages the OAuth 2.0 flow with your Trino server
2. **Proxies Requests**: Routes MCP tool calls through mcp-remote to your server
3. **Manages Tokens**: Stores and refreshes authentication tokens securely
4. **Provides Debug Logging**: Optional debug output for troubleshooting

## Installation

### Option 1: Download Pre-built DXT
1. Download a pre-built `.dxt` file for your environment
2. Double-click to install in Claude Desktop, or use your MCP client's extension installer
3. When prompted, configure your Trino MCP Server URL
4. The extension handles OAuth authentication automatically

### Option 2: Build Custom DXT
If you need a custom server URL, build your own DXT:

```bash
# From the mcp-trino repository root
make dxt URL=https://your-trino-server.com/mcp NAME=your-custom-name

# The DXT will be created in dxt-remote/your-custom-name.dxt
```

## Configuration

During installation, you'll be prompted to configure:

- **Trino MCP Server URL**: The HTTPS endpoint of your Trino MCP server (e.g., `https://your-server.com/mcp`)

The default URL can be customized when building the DXT extension.

## Authentication

This DXT extension automatically handles OAuth 2.0 authentication through mcp-remote. When you first use the extension:

1. **Initial Connection**: The extension will detect that authentication is required
2. **Browser OAuth Flow**: Your browser will open to complete the OAuth login
3. **Token Storage**: Authentication tokens are securely stored by mcp-remote
4. **Automatic Renewal**: Tokens are automatically refreshed as needed

No manual token configuration is required - the OAuth flow is handled automatically.

## Security Features

- **OAuth 2.0 Authentication**: Secure authentication flow with your identity provider
- **Read-Only Operations**: Queries are restricted to SELECT, SHOW, DESCRIBE, EXPLAIN by default
- **HTTPS Encryption**: All communications are encrypted in transit
- **Secure Token Storage**: Authentication tokens are stored securely by mcp-remote
- **No Local Caching**: No query results or sensitive data stored locally

## Troubleshooting

### Connection Issues

If you encounter connection problems:

1. **Network Connectivity**: Verify you can reach your Trino server URL in a browser
2. **HTTPS Requirements**: Ensure your server uses HTTPS (required for OAuth)
3. **Firewall Settings**: Check that HTTPS traffic (port 443) is allowed
4. **Server Status**: Verify your Trino MCP server is running and accessible

### Authentication Issues

If OAuth authentication fails:

1. **Browser Issues**: Clear browser cache and cookies for your server domain
2. **Token Expiry**: Delete stored tokens at `~/.mcp-auth/` to force re-authentication
3. **Provider Configuration**: Verify OAuth provider settings on your server
4. **Client Registration**: Ensure mcp-remote is registered as a valid OAuth client

### Debug Information

For detailed troubleshooting:

1. **Debug Logs**: The extension runs with `--debug` flag enabled by default
2. **Log Location**: Check `~/.mcp-auth/` directory for detailed logs
3. **Server Logs**: Check your Trino MCP server logs for authentication errors
4. **Network Inspection**: Use browser developer tools to inspect network requests

### Getting Help

- **Repository Issues**: [GitHub Issues](https://github.com/tuannvm/mcp-trino/issues)
- **MCP Documentation**: [Model Context Protocol Docs](https://modelcontextprotocol.io/)
- **mcp-remote Issues**: [mcp-remote GitHub](https://github.com/geelen/mcp-remote/issues)

## Building Your Own DXT

To create custom DXT extensions for your organization:

```bash
# Clone the repository
git clone https://github.com/tuannvm/mcp-trino.git
cd mcp-trino

# Build with your server URL
make dxt URL=https://your-company.com/mcp NAME=company-trino-remote

# Distribute the generated .dxt file
ls dxt-remote/company-trino-remote.dxt
```

The generated DXT can be distributed to your users for one-click installation.

## Technical Details

- **Transport**: HTTP with StreamableHTTP protocol via mcp-remote
- **Authentication**: OAuth 2.0 with PKCE (Proof Key for Code Exchange)
- **Security**: TLS encryption, secure token storage, read-only query restrictions
- **Compatibility**: Works with Claude Desktop, Cursor, Windsurf, and other MCP clients
- **Dependencies**: Automatically downloads mcp-remote via npx (no manual installation needed)