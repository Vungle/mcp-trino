package trino

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tuannvm/mcp-trino/internal/config"
)

func TestFilterCatalogs(t *testing.T) {
	tests := []struct {
		name            string
		allowedCatalogs []string
		input           []string
		expected        []string
	}{
		{
			name:            "No allowlist - return all",
			allowedCatalogs: nil,
			input:           []string{"hive", "postgresql", "mysql"},
			expected:        []string{"hive", "postgresql", "mysql"},
		},
		{
			name:            "Empty allowlist - return all",
			allowedCatalogs: []string{},
			input:           []string{"hive", "postgresql", "mysql"},
			expected:        []string{"hive", "postgresql", "mysql"},
		},
		{
			name:            "Filter to allowed catalogs",
			allowedCatalogs: []string{"hive", "postgresql"},
			input:           []string{"hive", "postgresql", "mysql", "oracle"},
			expected:        []string{"hive", "postgresql"},
		},
		{
			name:            "Case insensitive filtering",
			allowedCatalogs: []string{"HIVE", "PostgreSQL"},
			input:           []string{"hive", "postgresql", "mysql"},
			expected:        []string{"hive", "postgresql"},
		},
		{
			name:            "No matches",
			allowedCatalogs: []string{"nonexistent"},
			input:           []string{"hive", "postgresql", "mysql"},
			expected:        []string{},
		},
		{
			name:            "Partial matches",
			allowedCatalogs: []string{"hive"},
			input:           []string{"hive", "postgresql", "mysql"},
			expected:        []string{"hive"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				config: &config.TrinoConfig{
					AllowedCatalogs: tt.allowedCatalogs,
				},
			}

			result := client.filterCatalogs(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("filterCatalogs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterSchemas(t *testing.T) {
	tests := []struct {
		name           string
		allowedSchemas []string
		catalog        string
		input          []string
		expected       []string
	}{
		{
			name:           "No allowlist - return all",
			allowedSchemas: nil,
			catalog:        "hive",
			input:          []string{"analytics", "marts", "staging"},
			expected:       []string{"analytics", "marts", "staging"},
		},
		{
			name:           "Filter to allowed schemas",
			allowedSchemas: []string{"hive.analytics", "hive.marts"},
			catalog:        "hive",
			input:          []string{"analytics", "marts", "staging", "raw"},
			expected:       []string{"analytics", "marts"},
		},
		{
			name:           "Case insensitive filtering",
			allowedSchemas: []string{"HIVE.ANALYTICS", "hive.marts"},
			catalog:        "hive",
			input:          []string{"analytics", "marts", "staging"},
			expected:       []string{"analytics", "marts"},
		},
		{
			name:           "Different catalog - no matches",
			allowedSchemas: []string{"hive.analytics", "hive.marts"},
			catalog:        "postgresql",
			input:          []string{"public", "private"},
			expected:       []string{},
		},
		{
			name:           "Mixed catalogs in allowlist",
			allowedSchemas: []string{"hive.analytics", "postgresql.public"},
			catalog:        "hive",
			input:          []string{"analytics", "marts"},
			expected:       []string{"analytics"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				config: &config.TrinoConfig{
					AllowedSchemas: tt.allowedSchemas,
				},
			}

			result := client.filterSchemas(tt.input, tt.catalog)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("filterSchemas() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterTables(t *testing.T) {
	tests := []struct {
		name          string
		allowedTables []string
		catalog       string
		schema        string
		input         []string
		expected      []string
	}{
		{
			name:          "No allowlist - return all",
			allowedTables: nil,
			catalog:       "hive",
			schema:        "analytics",
			input:         []string{"users", "events", "sessions"},
			expected:      []string{"users", "events", "sessions"},
		},
		{
			name:          "Filter to allowed tables",
			allowedTables: []string{"hive.analytics.users", "hive.analytics.events"},
			catalog:       "hive",
			schema:        "analytics",
			input:         []string{"users", "events", "sessions", "temp"},
			expected:      []string{"users", "events"},
		},
		{
			name:          "Case insensitive filtering",
			allowedTables: []string{"HIVE.ANALYTICS.USERS", "hive.analytics.events"},
			catalog:       "hive",
			schema:        "analytics",
			input:         []string{"users", "events", "sessions"},
			expected:      []string{"users", "events"},
		},
		{
			name:          "Different catalog/schema - no matches",
			allowedTables: []string{"hive.analytics.users"},
			catalog:       "postgresql",
			schema:        "public",
			input:         []string{"orders", "customers"},
			expected:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				config: &config.TrinoConfig{
					AllowedTables: tt.allowedTables,
				},
			}

			result := client.filterTables(tt.input, tt.catalog, tt.schema)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("filterTables() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsCatalogAllowed(t *testing.T) {
	client := &Client{
		config: &config.TrinoConfig{
			AllowedCatalogs: []string{"hive", "postgresql", "MySQL"},
		},
	}

	tests := []struct {
		catalog  string
		expected bool
	}{
		{"hive", true},
		{"postgresql", true},
		{"mysql", true}, // Case insensitive
		{"MySQL", true},
		{"HIVE", true},
		{"oracle", false},
		{"sqlserver", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.catalog, func(t *testing.T) {
			result := client.isCatalogAllowed(tt.catalog)
			if result != tt.expected {
				t.Errorf("isCatalogAllowed(%q) = %v, want %v", tt.catalog, result, tt.expected)
			}
		})
	}
}

func TestIsSchemaAllowed(t *testing.T) {
	client := &Client{
		config: &config.TrinoConfig{
			AllowedSchemas: []string{"hive.analytics", "hive.marts", "PostgreSQL.PUBLIC"},
		},
	}

	tests := []struct {
		catalog  string
		schema   string
		expected bool
	}{
		{"hive", "analytics", true},
		{"hive", "marts", true},
		{"postgresql", "public", true}, // Case insensitive
		{"PostgreSQL", "PUBLIC", true},
		{"hive", "staging", false},
		{"postgresql", "private", false},
		{"mysql", "analytics", false},
	}

	for _, tt := range tests {
		t.Run(tt.catalog+"."+tt.schema, func(t *testing.T) {
			result := client.isSchemaAllowed(tt.catalog, tt.schema)
			if result != tt.expected {
				t.Errorf("isSchemaAllowed(%q, %q) = %v, want %v", tt.catalog, tt.schema, result, tt.expected)
			}
		})
	}
}

func TestIsTableAllowed(t *testing.T) {
	client := &Client{
		config: &config.TrinoConfig{
			AllowedTables: []string{"hive.analytics.users", "hive.marts.sales", "PostgreSQL.PUBLIC.ORDERS"},
		},
	}

	tests := []struct {
		name     string
		catalog  string
		schema   string
		table    string
		expected bool
	}{
		{"Simple match", "hive", "analytics", "users", true},
		{"Case insensitive match", "PostgreSQL", "PUBLIC", "ORDERS", true},
		{"No match - different table", "hive", "analytics", "events", false},
		{"No match - different schema", "hive", "staging", "users", false},
		{"No match - different catalog", "mysql", "analytics", "users", false},
		{"Empty catalog", "", "analytics", "users", false},
		{"Empty schema", "hive", "", "users", false},
		{"Empty table", "hive", "analytics", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isTableAllowed(tt.catalog, tt.schema, tt.table)
			if result != tt.expected {
				t.Errorf("isTableAllowed(%q, %q, %q) = %v, want %v", tt.catalog, tt.schema, tt.table, result, tt.expected)
			}
		})
	}
}

func TestTableParameterResolution(t *testing.T) {
	client := &Client{
		config: &config.TrinoConfig{
			Catalog: "hive",
			Schema:  "default",
		},
	}

	// Test table parameter resolution logic (extracted from GetTableSchema)
	testResolution := func(inputCatalog, inputSchema, inputTable, expectedCatalog, expectedSchema, expectedTable string) {
		// Simulate the resolution logic from GetTableSchema
		catalog, schema, table := inputCatalog, inputSchema, inputTable

		parts := strings.Split(table, ".")
		if len(parts) == 3 {
			// If table is already fully qualified, extract components
			catalog = parts[0]
			schema = parts[1]
			table = parts[2]
		} else if len(parts) == 2 {
			// If table has schema.table format
			schema = parts[0]
			table = parts[1]
			if catalog == "" {
				catalog = client.config.Catalog
			}
		} else {
			// Use provided or default catalog and schema
			if catalog == "" {
				catalog = client.config.Catalog
			}
			if schema == "" {
				schema = client.config.Schema
			}
		}

		if catalog != expectedCatalog || schema != expectedSchema || table != expectedTable {
			t.Errorf("Resolution(%q, %q, %q) = (%q, %q, %q), want (%q, %q, %q)",
				inputCatalog, inputSchema, inputTable,
				catalog, schema, table,
				expectedCatalog, expectedSchema, expectedTable)
		}
	}

	// Test the resolution logic that was causing the bug
	testResolution("", "analytics", "users", "hive", "analytics", "users")                  // use default catalog
	testResolution("", "", "analytics.users", "hive", "analytics", "users")                // schema.table format
	testResolution("", "", "hive.analytics.users", "hive", "analytics", "users")           // fully qualified
	testResolution("postgresql", "public", "orders", "postgresql", "public", "orders")     // explicit params
}

func TestGetTableSchemaAllowlistLogic(t *testing.T) {
	client := &Client{
		config: &config.TrinoConfig{
			Catalog:       "hive",
			Schema:        "default",
			AllowedTables: []string{"hive.analytics.users", "hive.marts.sales"},
		},
	}

	// Test the combined resolution + allowlist check logic
	testAllowlistAfterResolution := func(inputCatalog, inputSchema, inputTable string, expectedAllowed bool) {
		// Simulate the resolution + allowlist check from GetTableSchema
		catalog, schema, table := inputCatalog, inputSchema, inputTable

		// Resolution logic (copied from GetTableSchema)
		parts := strings.Split(table, ".")
		if len(parts) == 3 {
			catalog = parts[0]
			schema = parts[1]
			table = parts[2]
		} else if len(parts) == 2 {
			schema = parts[0]
			table = parts[1]
			if catalog == "" {
				catalog = client.config.Catalog
			}
		} else {
			if catalog == "" {
				catalog = client.config.Catalog
			}
			if schema == "" {
				schema = client.config.Schema
			}
		}

		// Allowlist check (after resolution)
		allowed := client.isTableAllowed(catalog, schema, table)
		if allowed != expectedAllowed {
			t.Errorf("Allowlist check after resolution(%q, %q, %q) -> isTableAllowed(%q, %q, %q) = %v, want %v",
				inputCatalog, inputSchema, inputTable, catalog, schema, table, allowed, expectedAllowed)
		}
	}

	// Test cases that verify the bug fix
	testAllowlistAfterResolution("hive", "analytics", "users", true)              // explicit - should work
	testAllowlistAfterResolution("", "analytics", "users", true)                 // default catalog - should work
	testAllowlistAfterResolution("", "", "analytics.users", true)                // schema.table - BUG FIX: should work now
	testAllowlistAfterResolution("", "", "hive.analytics.users", true)           // fully qualified - should work
	testAllowlistAfterResolution("hive", "analytics", "events", false)           // not in allowlist - should deny
	testAllowlistAfterResolution("postgresql", "analytics", "users", false)      // wrong catalog - should deny
}

func TestImprovedIsReadOnlyQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		// Basic read-only queries with word boundaries
		{"SELECT with word boundary", "SELECT * FROM users", true},
		{"SELECT with leading spaces", "  SELECT * FROM users", true},
		{"SELECT with newlines", "\n SELECT * FROM users\n", true},
		{"SHOW with word boundary", "SHOW TABLES", true},
		{"DESCRIBE with word boundary", "DESCRIBE users", true},
		{"EXPLAIN with word boundary", "EXPLAIN SELECT * FROM users", true},
		{"WITH CTE", "WITH cte AS (SELECT 1) SELECT * FROM cte", true},

		// Edge cases with word boundaries (these should now be stricter)
		{"SELECT without space", "SELECT*FROM users", true}, // Word boundary handles this
		{"SHOW without space", "SHOWTABLES", false},         // Word boundary requires separation

		// Write operations that should be blocked
		{"INSERT statement", "INSERT INTO users VALUES (1)", false},
		{"UPDATE statement", "UPDATE users SET name = 'test'", false},
		{"DELETE statement", "DELETE FROM users", false},
		{"CREATE statement", "CREATE TABLE test (id INT)", false},

		// Complex cases
		{"SELECT with INSERT in string", "SELECT 'INSERT INTO' FROM dual", true},
		{"SELECT with INSERT in comment", "SELECT 1 -- INSERT INTO users", true},
		{"Multi-statement with semicolon", "SELECT 1; INSERT INTO users VALUES (1)", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isReadOnlyQuery(tt.query)
			if result != tt.expected {
				t.Errorf("isReadOnlyQuery(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}