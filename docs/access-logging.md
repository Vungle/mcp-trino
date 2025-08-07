# Access Logging

The MCP Trino Server includes comprehensive access logging functionality that provides detailed information about all HTTP requests and MCP tool operations.

## Overview

Access logging is implemented at multiple levels:

1. **HTTP Request Logging**: All HTTP requests are logged with detailed information
2. **OAuth Request Logging**: OAuth-related endpoints have enhanced logging
3. **MCP Tool Logging**: Individual tool requests are logged with timing and results

## HTTP Access Logs

Every HTTP request generates a structured JSON log entry with the following fields:

```json
{
  "timestamp": "2024-01-15T10:30:45.123Z",
  "method": "POST",
  "path": "/mcp",
  "query": "",
  "remote_addr": "192.168.1.100:54321",
  "user_agent": "curl/7.68.0",
  "status_code": 200,
  "response_time_ms": 150,
  "content_length": 1024,
  "request_id": "req_1705311045123456789",
  "oauth_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### Fields Explained

- **timestamp**: ISO 8601 timestamp of when the request started
- **method**: HTTP method (GET, POST, etc.)
- **path**: Request path
- **query**: Query parameters (if any)
- **remote_addr**: Client IP address and port
- **user_agent**: Client's User-Agent header
- **status_code**: HTTP response status code
- **response_time_ms**: Response time in milliseconds
- **content_length**: Size of response body in bytes
- **request_id**: Unique identifier for the request
- **oauth_token**: First 10 characters of OAuth token (if present)

## OAuth Access Logs

OAuth endpoints have additional detailed logging:

### Metadata Endpoints
- Request timing and response times
- Client information (IP, User-Agent)
- Configuration status (enabled/disabled)
- Error details with client context

### Registration Endpoints
- Client registration requests with parameters
- Redirect URI configuration
- Response timing

### Authorization Endpoints
- Authorization flow tracking
- Token validation attempts
- Error conditions with context

## MCP Tool Logs

Each MCP tool request generates detailed logs:

### Tool Request Logs
```
TOOL_REQUEST: execute_query from 192.168.1.100:54321 - Query: SELECT * FROM system.runtime.queries
```

### Tool Success Logs
```
TOOL_SUCCESS: {"timestamp":"2024-01-15T10:30:45.123Z","tool":"execute_query","args":{"query":"SELECT * FROM system.runtime.queries"},"response_time":150,"remote_addr":"192.168.1.100:54321"}
```

### Tool Error Logs
```
TOOL_ERROR: {"timestamp":"2024-01-15T10:30:45.123Z","tool":"execute_query","args":{"query":"INVALID SQL"},"response_time":50,"remote_addr":"192.168.1.100:54321","error":"query execution failed: syntax error"}
```

### Tool Response Logs
```
TOOL_RESPONSE: execute_query to 192.168.1.100:54321 - Results size: 2048 bytes
```

## Log Format

All access logs use structured JSON format for easy parsing and analysis:

- **HTTP Access Logs**: Prefixed with `ACCESS_LOG:`
- **OAuth Logs**: Prefixed with `OAuth2:`
- **Tool Logs**: Prefixed with `TOOL_REQUEST:`, `TOOL_SUCCESS:`, `TOOL_ERROR:`, or `TOOL_RESPONSE:`

## Configuration

Access logging is enabled by default and cannot be disabled. The logging level follows the standard Go `log` package configuration.

### Environment Variables

No additional configuration is required for access logging. The following environment variables affect logging behavior:

- `MCP_PORT`: Port for HTTP server (default: 8080)
- `MCP_HOST`: Host for HTTP server (default: localhost)

## Testing Access Logs

Use the provided test script to generate sample access logs:

```bash
# Make the script executable
chmod +x scripts/test_access_logs.sh

# Run the test script
./scripts/test_access_logs.sh
```

This script will:
1. Test status endpoints
2. Test OAuth metadata endpoints
3. Test MCP endpoints with and without authentication
4. Test different User-Agent strings
5. Test query parameters
6. Test CORS preflight requests
7. Test large request bodies

## Log Analysis

### Parsing JSON Logs

You can parse the structured logs using tools like `jq`:

```bash
# Extract all access logs
grep "ACCESS_LOG:" server.log | jq -r .

# Extract tool requests
grep "TOOL_REQUEST:" server.log

# Extract errors
grep "TOOL_ERROR:" server.log | jq -r .

# Calculate average response time
grep "ACCESS_LOG:" server.log | jq -r '.response_time_ms' | awk '{sum+=$1} END {print "Average:", sum/NR}'
```

### Monitoring Patterns

Common patterns to monitor:

1. **High Response Times**: Look for `response_time_ms` > 1000
2. **Authentication Failures**: Look for 401 status codes
3. **Tool Errors**: Monitor `TOOL_ERROR` entries
4. **Large Requests**: Monitor `content_length` for unusually large responses

## Security Considerations

- OAuth tokens are truncated to first 10 characters in logs
- No sensitive data is logged in plain text
- Request IDs help correlate related log entries
- Remote addresses are logged for security monitoring

## Performance Impact

Access logging has minimal performance impact:
- JSON marshaling is efficient
- Log entries are written asynchronously
- No blocking operations in the request path
- Memory usage is constant per request

## Troubleshooting

### Missing Logs
- Ensure the server is running in HTTP mode (`MCP_TRANSPORT=http`)
- Check that the log output is not being redirected
- Verify that the logging middleware is properly applied

### High Response Times
- Monitor `response_time_ms` in access logs
- Check for database connection issues
- Look for large response sizes in `content_length`

### Authentication Issues
- Check OAuth token format in logs
- Monitor 401 status codes
- Verify OAuth configuration 