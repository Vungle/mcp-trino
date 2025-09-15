package trino

import (
	"testing"
)

func TestIsReadOnlyQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		// Basic read-only queries
		{
			name:     "Simple SELECT query",
			query:    "SELECT * FROM table",
			expected: true,
		},
		{
			name:     "SELECT query with WHERE clause",
			query:    "SELECT id, name FROM users WHERE age > 18",
			expected: true,
		},
		{
			name:     "SHOW query",
			query:    "SHOW TABLES",
			expected: true,
		},
		{
			name:     "DESCRIBE query",
			query:    "DESCRIBE users",
			expected: true,
		},
		{
			name:     "EXPLAIN query",
			query:    "EXPLAIN SELECT * FROM users",
			expected: true,
		},
		{
			name:     "WITH query (CTE)",
			query:    "WITH cte AS (SELECT * FROM users) SELECT * FROM cte",
			expected: true,
		},

		// Complex read-only queries
		{
			name:     "SELECT with GROUP BY",
			query:    "SELECT department, COUNT(*) FROM employees GROUP BY department",
			expected: true,
		},
		{
			name:     "SELECT with ORDER BY",
			query:    "SELECT * FROM products ORDER BY price DESC",
			expected: true,
		},
		{
			name:     "SELECT with JOIN",
			query:    "SELECT u.name, o.product FROM users u JOIN orders o ON u.id = o.user_id",
			expected: true,
		},
		{
			name:     "Complex SELECT with multiple clauses",
			query:    "SELECT department, COUNT(*) as count, AVG(salary) as avg_salary FROM employees WHERE hire_date > '2020-01-01' GROUP BY department HAVING count > 5 ORDER BY avg_salary DESC LIMIT 10",
			expected: true,
		},

		// Queries with different whitespace formatting
		{
			name:     "SELECT with newlines",
			query:    "SELECT\n* FROM\nusers",
			expected: true,
		},
		{
			name:     "SELECT with tabs and spaces",
			query:    "SELECT    id,\n\t\tname\nFROM users",
			expected: true,
		},
		{
			name:     "SELECT keyword without space",
			query:    "SELECT*FROM users",
			expected: true,
		},
		{
			name:     "SELECT with leading and trailing whitespace",
			query:    "  \n  SELECT * FROM users  \n  ",
			expected: true,
		},

		// Keywords without spaces
		{
			name:     "SELECT without space after keyword",
			query:    "SELECTid, name FROM users",
			expected: true,
		},
		{
			name:     "SHOW without space after keyword",
			query:    "SHOWtables",
			expected: true,
		},
		{
			name:     "DESCRIBE without space after keyword",
			query:    "DESCRIBEusers",
			expected: true,
		},

		// Case insensitivity
		{
			name:     "Lowercase SELECT",
			query:    "select * from users",
			expected: true,
		},
		{
			name:     "Mixed case SELECT",
			query:    "SeLeCt * FrOm UsErS",
			expected: true,
		},

		// Write operations (should return false)
		{
			name:     "INSERT query",
			query:    "INSERT INTO users VALUES (1, 'John')",
			expected: false,
		},
		{
			name:     "UPDATE query",
			query:    "UPDATE users SET name = 'John' WHERE id = 1",
			expected: false,
		},
		{
			name:     "DELETE query",
			query:    "DELETE FROM users WHERE id = 1",
			expected: false,
		},
		{
			name:     "DROP query",
			query:    "DROP TABLE users",
			expected: false,
		},
		{
			name:     "CREATE query",
			query:    "CREATE TABLE users (id INT, name VARCHAR)",
			expected: false,
		},
		{
			name:     "ALTER query",
			query:    "ALTER TABLE users ADD COLUMN email VARCHAR",
			expected: false,
		},
		{
			name:     "TRUNCATE query",
			query:    "TRUNCATE TABLE users",
			expected: false,
		},

		// Sneaky write operations embedded in SELECT (should return false)
		{
			name:     "SELECT with embedded INSERT",
			query:    "SELECT * FROM users; INSERT INTO logs VALUES ('accessed')",
			expected: false,
		},
		{
			name:     "SELECT with embedded UPDATE",
			query:    "SELECT * FROM (UPDATE users SET active = true RETURNING *) AS updated",
			expected: false,
		},
		{
			name:     "SELECT with embedded DELETE",
			query:    "SELECT * FROM users WHERE id IN (DELETE FROM inactive_users RETURNING user_id)",
			expected: false,
		},

		// New write operations (should return false)
		{
			name:     "MERGE query",
			query:    "MERGE INTO target USING source ON target.id = source.id",
			expected: false,
		},
		{
			name:     "COPY query",
			query:    "COPY table FROM 's3://bucket/data.csv'",
			expected: false,
		},
		{
			name:     "GRANT query",
			query:    "GRANT SELECT ON table TO user",
			expected: false,
		},
		{
			name:     "REVOKE query",
			query:    "REVOKE SELECT ON table FROM user",
			expected: false,
		},
		{
			name:     "COMMIT query",
			query:    "COMMIT",
			expected: false,
		},
		{
			name:     "ROLLBACK query",
			query:    "ROLLBACK",
			expected: false,
		},
		{
			name:     "CALL query",
			query:    "CALL some_procedure()",
			expected: false,
		},
		{
			name:     "EXECUTE query",
			query:    "EXECUTE prepared_statement",
			expected: false,
		},
		{
			name:     "REFRESH query",
			query:    "REFRESH MATERIALIZED VIEW view_name",
			expected: false,
		},
		{
			name:     "SET SESSION query",
			query:    "SET SESSION query_max_run_time = '1h'",
			expected: false,
		},
		{
			name:     "RESET SESSION query",
			query:    "RESET SESSION query_max_run_time",
			expected: false,
		},

		// Whitespace bypass attempts (should return false)
		{
			name:     "DELETE with tab",
			query:    "DELETE\tFROM users",
			expected: false,
		},
		{
			name:     "UPDATE with multiple spaces",
			query:    "UPDATE  users SET active = false",
			expected: false,
		},
		{
			name:     "INSERT with newline",
			query:    "INSERT\nINTO users VALUES (1, 'test')",
			expected: false,
		},
		{
			name:     "MERGE with tab",
			query:    "MERGE\tINTO target USING source",
			expected: false,
		},
		{
			name:     "DROP with carriage return",
			query:    "DROP\rTABLE users",
			expected: false,
		},
		{
			name:     "RESET with multiple spaces",
			query:    "RESET  SESSION query_max_run_time",
			expected: false,
		},
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
