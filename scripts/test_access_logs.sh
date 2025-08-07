#!/bin/bash

# Test script to demonstrate access logging functionality
# This script makes various requests to the MCP server to generate access logs

set -e

echo "Testing Access Logs for MCP Trino Server"
echo "========================================"

# Configuration
SERVER_URL="http://localhost:8080"
MCP_ENDPOINT="$SERVER_URL/mcp"
STATUS_ENDPOINT="$SERVER_URL/"

echo "1. Testing status endpoint..."
curl -s -w "Status: %{http_code}, Time: %{time_total}s\n" "$STATUS_ENDPOINT"

echo -e "\n2. Testing OAuth metadata endpoints..."
curl -s -w "Status: %{http_code}, Time: %{time_total}s\n" "$SERVER_URL/.well-known/oauth-authorization-server"
curl -s -w "Status: %{http_code}, Time: %{time_total}s\n" "$SERVER_URL/.well-known/oauth-metadata"

echo -e "\n3. Testing MCP endpoint without authentication..."
curl -s -w "Status: %{http_code}, Time: %{time_total}s\n" -X POST "$MCP_ENDPOINT" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

echo -e "\n4. Testing MCP endpoint with invalid authentication..."
curl -s -w "Status: %{http_code}, Time: %{time_total}s\n" -X POST "$MCP_ENDPOINT" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer invalid-token" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

echo -e "\n5. Testing with different User-Agent..."
curl -s -w "Status: %{http_code}, Time: %{time_total}s\n" \
  -H "User-Agent: TestClient/1.0" \
  "$STATUS_ENDPOINT"

echo -e "\n6. Testing with query parameters..."
curl -s -w "Status: %{http_code}, Time: %{time_total}s\n" \
  "$STATUS_ENDPOINT?test=1&debug=true"

echo -e "\n7. Testing OPTIONS request (CORS preflight)..."
curl -s -w "Status: %{http_code}, Time: %{time_total}s\n" \
  -X OPTIONS "$MCP_ENDPOINT" \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type,Authorization"

echo -e "\n8. Testing large request body..."
curl -s -w "Status: %{http_code}, Time: %{time_total}s\n" -X POST "$MCP_ENDPOINT" \
  -H "Content-Type: application/json" \
  -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"execute_query\",\"arguments\":{\"query\":\"$(printf 'SELECT %s FROM system.runtime.queries' "$(seq -s ',' 1 100 | sed 's/,/ as col_/g')")\"}}}"

echo -e "\nAccess log testing completed!"
echo "Check the server logs to see the detailed access log entries."
echo "Each request should generate structured JSON logs with:"
echo "- Timestamp"
echo "- HTTP method and path"
echo "- Remote address"
echo "- User agent"
echo "- Status code"
echo "- Response time"
echo "- Content length"
echo "- Request ID"
echo "- OAuth token (if present)" 