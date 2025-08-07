package demo

import (
	"encoding/json"
	"fmt"
	"time"
)

// AccessLog represents a single access log entry (same as in server.go)
type AccessLog struct {
	Timestamp     time.Time `json:"timestamp"`
	Method        string    `json:"method"`
	Path          string    `json:"path"`
	Query         string    `json:"query"`
	RemoteAddr    string    `json:"remote_addr"`
	UserAgent     string    `json:"user_agent"`
	StatusCode    int       `json:"status_code"`
	ResponseTime  int64     `json:"response_time_ms"`
	ContentLength int64     `json:"content_length"`
	RequestID     string    `json:"request_id"`
	OAuthToken    string    `json:"oauth_token,omitempty"`
	Error         string    `json:"error,omitempty"`
}

func demo() {
	fmt.Println("Access Logging Demo")
	fmt.Println("===================")

	// Example 1: Basic HTTP request log
	fmt.Println("\n1. Basic HTTP Request Log:")
	basicLog := AccessLog{
		Timestamp:     time.Now(),
		Method:        "POST",
		Path:          "/mcp",
		Query:         "",
		RemoteAddr:    "192.168.1.100:54321",
		UserAgent:     "curl/7.68.0",
		StatusCode:    200,
		ResponseTime:  150,
		ContentLength: 1024,
		RequestID:     "req_1705311045123456789",
	}

	jsonData, _ := json.MarshalIndent(basicLog, "", "  ")
	fmt.Printf("ACCESS_LOG: %s\n", string(jsonData))

	// Example 2: OAuth request log
	fmt.Println("\n2. OAuth Request Log:")
	oauthLog := AccessLog{
		Timestamp:     time.Now(),
		Method:        "GET",
		Path:          "/.well-known/oauth-authorization-server",
		Query:         "",
		RemoteAddr:    "10.0.0.50:12345",
		UserAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
		StatusCode:    200,
		ResponseTime:  25,
		ContentLength: 512,
		RequestID:     "req_1705311045123456790",
	}

	jsonData, _ = json.MarshalIndent(oauthLog, "", "  ")
	fmt.Printf("ACCESS_LOG: %s\n", string(jsonData))

	// Example 3: Tool request with OAuth token
	fmt.Println("\n3. Tool Request with OAuth Token:")
	toolLog := AccessLog{
		Timestamp:     time.Now(),
		Method:        "POST",
		Path:          "/mcp",
		Query:         "",
		RemoteAddr:    "172.16.0.10:8080",
		UserAgent:     "Claude-Desktop/1.0",
		StatusCode:    200,
		ResponseTime:  300,
		ContentLength: 2048,
		RequestID:     "req_1705311045123456791",
		OAuthToken:    "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
	}

	jsonData, _ = json.MarshalIndent(toolLog, "", "  ")
	fmt.Printf("ACCESS_LOG: %s\n", string(jsonData))

	// Example 4: Error log
	fmt.Println("\n4. Error Log:")
	errorLog := AccessLog{
		Timestamp:     time.Now(),
		Method:        "POST",
		Path:          "/mcp",
		Query:         "",
		RemoteAddr:    "192.168.1.100:54321",
		UserAgent:     "curl/7.68.0",
		StatusCode:    401,
		ResponseTime:  50,
		ContentLength: 256,
		RequestID:     "req_1705311045123456792",
		Error:         "No bearer token provided",
	}

	jsonData, _ = json.MarshalIndent(errorLog, "", "  ")
	fmt.Printf("ACCESS_LOG: %s\n", string(jsonData))

	// Example 5: Tool-specific logs
	fmt.Println("\n5. Tool Request/Response Logs:")
	fmt.Println("TOOL_REQUEST: execute_query from 192.168.1.100:54321 - Query: SELECT * FROM system.runtime.queries")

	toolSuccessLog := map[string]interface{}{
		"timestamp":     time.Now(),
		"tool":          "execute_query",
		"args":          map[string]interface{}{"query": "SELECT * FROM system.runtime.queries"},
		"response_time": 150,
		"remote_addr":   "192.168.1.100:54321",
	}

	jsonData, _ = json.MarshalIndent(toolSuccessLog, "", "  ")
	fmt.Printf("TOOL_SUCCESS: %s\n", string(jsonData))

	fmt.Println("TOOL_RESPONSE: execute_query to 192.168.1.100:54321 - Results size: 2048 bytes")

	// Example 6: OAuth detailed logs
	fmt.Println("\n6. OAuth Detailed Logs:")
	fmt.Println("OAuth2: Authorization Server Metadata request from 10.0.0.50:12345 (User-Agent: Mozilla/5.0...)")
	fmt.Println("OAuth2: Authorization Server Metadata response sent to 10.0.0.50:12345 in 25ms")

	fmt.Println("\nAccess Logging Demo Complete!")
	fmt.Println("\nTo test with the actual server:")
	fmt.Println("1. Start the server: MCP_TRANSPORT=http ./main")
	fmt.Println("2. Run the test script: ./scripts/test_access_logs.sh")
	fmt.Println("3. Check the server logs for structured JSON access logs")
}

func main() {
	demo()
}
