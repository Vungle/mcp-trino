package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/mcp-trino/internal/trino"
)

// TrinoHandlers contains all handlers for Trino-related tools
type TrinoHandlers struct {
	TrinoClient *trino.Client
}

// NewTrinoHandlers creates a new set of Trino handlers
func NewTrinoHandlers(client *trino.Client) *TrinoHandlers {
	return &TrinoHandlers{
		TrinoClient: client,
	}
}

// logToolRequest logs detailed information about a tool request
func logToolRequest(toolName string, args map[string]interface{}, startTime time.Time, err error, remoteAddr string) {
	responseTime := time.Since(startTime).Milliseconds()

	logEntry := map[string]interface{}{
		"timestamp":     startTime,
		"tool":          toolName,
		"args":          args,
		"response_time": responseTime,
		"remote_addr":   remoteAddr,
	}

	if err != nil {
		logEntry["error"] = err.Error()
		log.Printf("TOOL_ERROR: %s", logEntry)
	} else {
		log.Printf("TOOL_SUCCESS: %s", logEntry)
	}
}

// ExecuteQuery handles query execution
func (h *TrinoHandlers) ExecuteQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	// Extract remote address from context if available
	remoteAddr := "unknown"
	if req, ok := ctx.Value("http_request").(*http.Request); ok {
		remoteAddr = req.RemoteAddr
	}

	// Type assert Arguments to map[string]interface{}
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		mcpErr := fmt.Errorf("invalid arguments format")
		logToolRequest("execute_query", nil, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	// Extract the query parameter
	query, ok := args["query"].(string)
	if !ok {
		mcpErr := fmt.Errorf("query parameter must be a string")
		logToolRequest("execute_query", args, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	log.Printf("TOOL_REQUEST: execute_query from %s - Query: %s", remoteAddr, query)

	// Execute the query - SQL injection protection is handled within the client
	results, err := h.TrinoClient.ExecuteQuery(query)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		mcpErr := fmt.Errorf("query execution failed: %w", err)
		logToolRequest("execute_query", args, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	// Convert results to JSON string for display
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		mcpErr := fmt.Errorf("failed to marshal results to JSON: %w", err)
		logToolRequest("execute_query", args, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	logToolRequest("execute_query", args, startTime, nil, remoteAddr)
	log.Printf("TOOL_RESPONSE: execute_query to %s - Results size: %d bytes", remoteAddr, len(jsonData))

	// Return the results as formatted JSON text
	return mcp.NewToolResultText(string(jsonData)), nil
}

// ListCatalogs handles catalog listing
func (h *TrinoHandlers) ListCatalogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	// Extract remote address from context if available
	remoteAddr := "unknown"
	if req, ok := ctx.Value("http_request").(*http.Request); ok {
		remoteAddr = req.RemoteAddr
	}

	log.Printf("TOOL_REQUEST: list_catalogs from %s", remoteAddr)

	catalogs, err := h.TrinoClient.ListCatalogs()
	if err != nil {
		log.Printf("Error listing catalogs: %v", err)
		mcpErr := fmt.Errorf("failed to list catalogs: %w", err)
		logToolRequest("list_catalogs", nil, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	// Convert catalogs to JSON string for display
	jsonData, err := json.MarshalIndent(catalogs, "", "  ")
	if err != nil {
		mcpErr := fmt.Errorf("failed to marshal catalogs to JSON: %w", err)
		logToolRequest("list_catalogs", nil, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	logToolRequest("list_catalogs", nil, startTime, nil, remoteAddr)
	log.Printf("TOOL_RESPONSE: list_catalogs to %s - Found %d catalogs", remoteAddr, len(catalogs))

	return mcp.NewToolResultText(string(jsonData)), nil
}

// ListSchemas handles schema listing
func (h *TrinoHandlers) ListSchemas(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	// Extract remote address from context if available
	remoteAddr := "unknown"
	if req, ok := ctx.Value("http_request").(*http.Request); ok {
		remoteAddr = req.RemoteAddr
	}

	// Type assert Arguments to map[string]interface{}
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		mcpErr := fmt.Errorf("invalid arguments format")
		logToolRequest("list_schemas", nil, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	// Extract catalog parameter (optional)
	var catalog string
	if catalogParam, ok := args["catalog"].(string); ok {
		catalog = catalogParam
	}

	log.Printf("TOOL_REQUEST: list_schemas from %s - Catalog: %s", remoteAddr, catalog)

	schemas, err := h.TrinoClient.ListSchemas(catalog)
	if err != nil {
		log.Printf("Error listing schemas: %v", err)
		mcpErr := fmt.Errorf("failed to list schemas: %w", err)
		logToolRequest("list_schemas", args, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	// Convert schemas to JSON string for display
	jsonData, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		mcpErr := fmt.Errorf("failed to marshal schemas to JSON: %w", err)
		logToolRequest("list_schemas", args, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	logToolRequest("list_schemas", args, startTime, nil, remoteAddr)
	log.Printf("TOOL_RESPONSE: list_schemas to %s - Found %d schemas", remoteAddr, len(schemas))

	return mcp.NewToolResultText(string(jsonData)), nil
}

// ListTables handles table listing
func (h *TrinoHandlers) ListTables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	// Extract remote address from context if available
	remoteAddr := "unknown"
	if req, ok := ctx.Value("http_request").(*http.Request); ok {
		remoteAddr = req.RemoteAddr
	}

	// Type assert Arguments to map[string]interface{}
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		mcpErr := fmt.Errorf("invalid arguments format")
		logToolRequest("list_tables", nil, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	// Extract catalog and schema parameters (optional)
	var catalog, schema string
	if catalogParam, ok := args["catalog"].(string); ok {
		catalog = catalogParam
	}
	if schemaParam, ok := args["schema"].(string); ok {
		schema = schemaParam
	}

	log.Printf("TOOL_REQUEST: list_tables from %s - Catalog: %s, Schema: %s", remoteAddr, catalog, schema)

	tables, err := h.TrinoClient.ListTables(catalog, schema)
	if err != nil {
		log.Printf("Error listing tables: %v", err)
		mcpErr := fmt.Errorf("failed to list tables: %w", err)
		logToolRequest("list_tables", args, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	// Convert tables to JSON string for display
	jsonData, err := json.MarshalIndent(tables, "", "  ")
	if err != nil {
		mcpErr := fmt.Errorf("failed to marshal tables to JSON: %w", err)
		logToolRequest("list_tables", args, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	logToolRequest("list_tables", args, startTime, nil, remoteAddr)
	log.Printf("TOOL_RESPONSE: list_tables to %s - Found %d tables", remoteAddr, len(tables))

	return mcp.NewToolResultText(string(jsonData)), nil
}

// GetTableSchema handles table schema retrieval
func (h *TrinoHandlers) GetTableSchema(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	// Extract remote address from context if available
	remoteAddr := "unknown"
	if req, ok := ctx.Value("http_request").(*http.Request); ok {
		remoteAddr = req.RemoteAddr
	}

	// Type assert Arguments to map[string]interface{}
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		mcpErr := fmt.Errorf("invalid arguments format")
		logToolRequest("get_table_schema", nil, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	// Extract parameters
	var catalog, schema string
	var table string

	if catalogParam, ok := args["catalog"].(string); ok {
		catalog = catalogParam
	}
	if schemaParam, ok := args["schema"].(string); ok {
		schema = schemaParam
	}

	// Table parameter is required
	tableParam, ok := args["table"].(string)
	if !ok {
		mcpErr := fmt.Errorf("table parameter is required")
		logToolRequest("get_table_schema", args, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}
	table = tableParam

	log.Printf("TOOL_REQUEST: get_table_schema from %s - Catalog: %s, Schema: %s, Table: %s", remoteAddr, catalog, schema, table)

	tableSchema, err := h.TrinoClient.GetTableSchema(catalog, schema, table)
	if err != nil {
		log.Printf("Error getting table schema: %v", err)
		mcpErr := fmt.Errorf("failed to get table schema: %w", err)
		logToolRequest("get_table_schema", args, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	// Convert table schema to JSON string for display
	jsonData, err := json.MarshalIndent(tableSchema, "", "  ")
	if err != nil {
		mcpErr := fmt.Errorf("failed to marshal table schema to JSON: %w", err)
		logToolRequest("get_table_schema", args, startTime, mcpErr, remoteAddr)
		return mcp.NewToolResultErrorFromErr(mcpErr.Error(), mcpErr), nil
	}

	logToolRequest("get_table_schema", args, startTime, nil, remoteAddr)
	log.Printf("TOOL_RESPONSE: get_table_schema to %s - Schema retrieved successfully", remoteAddr)

	return mcp.NewToolResultText(string(jsonData)), nil
}

// RegisterTrinoTools registers all Trino-related tools with the MCP server
func RegisterTrinoTools(m *server.MCPServer, h *TrinoHandlers) {
	// Get OAuth middleware if available
	var middleware func(server.ToolHandlerFunc) server.ToolHandlerFunc
	if oauthMiddleware := GetOAuthMiddleware(m); oauthMiddleware != nil {
		middleware = oauthMiddleware
	}

	// Helper function to apply middleware if available
	applyMiddleware := func(handler server.ToolHandlerFunc) server.ToolHandlerFunc {
		if middleware != nil {
			return middleware(handler)
		}
		return handler
	}

	m.AddTool(mcp.NewTool("execute_query",
		mcp.WithDescription("Execute a SQL query"),
		mcp.WithString("query", mcp.Required(), mcp.Description("SQL query")),
	), applyMiddleware(h.ExecuteQuery))

	m.AddTool(mcp.NewTool("list_catalogs", mcp.WithDescription("List catalogs")),
		applyMiddleware(h.ListCatalogs))

	m.AddTool(mcp.NewTool("list_schemas",
		mcp.WithDescription("List schemas"),
		mcp.WithString("catalog", mcp.Description("Catalog"))),
		applyMiddleware(h.ListSchemas))

	m.AddTool(mcp.NewTool("list_tables",
		mcp.WithDescription("List tables"),
		mcp.WithString("catalog", mcp.Description("Catalog")),
		mcp.WithString("schema", mcp.Description("Schema"))),
		applyMiddleware(h.ListTables))

	m.AddTool(mcp.NewTool("get_table_schema",
		mcp.WithDescription("Get table schema"),
		mcp.WithString("catalog", mcp.Description("Catalog")),
		mcp.WithString("schema", mcp.Description("Schema")),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table"))),
		applyMiddleware(h.GetTableSchema))
}
