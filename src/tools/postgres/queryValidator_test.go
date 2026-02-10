package postgres

import (
	"strings"
	"testing"
)

func TestValidateQuery(t *testing.T) {
	allowedTables := []string{"users", "products", "orders"}

	tests := []struct {
		name      string
		query     string
		mode      string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid SELECT in read mode",
			query:     "SELECT * FROM users WHERE age > 25",
			mode:      "read",
			wantError: false,
		},
		{
			name:      "valid SELECT with JOIN in read mode",
			query:     "SELECT u.name, o.order_id FROM users u JOIN orders o ON u.user_id = o.user_id",
			mode:      "read",
			wantError: false,
		},
		{
			name:      "INSERT in read mode - should fail",
			query:     "INSERT INTO users (name, email) VALUES ('John', 'john@example.com')",
			mode:      "read",
			wantError: true,
			errorMsg:  "not allowed in READ mode",
		},
		{
			name:      "UPDATE in read mode - should fail",
			query:     "UPDATE users SET status = 'active' WHERE user_id = 1",
			mode:      "read",
			wantError: true,
			errorMsg:  "not allowed in READ mode",
		},
		{
			name:      "valid INSERT in write mode",
			query:     "INSERT INTO products (name, price) VALUES ('Widget', 19.99)",
			mode:      "write",
			wantError: false,
		},
		{
			name:      "valid UPDATE in write mode",
			query:     "UPDATE orders SET order_status = 'shipped' WHERE order_id = 123",
			mode:      "write",
			wantError: false,
		},
		{
			name:      "valid DELETE in write mode",
			query:     "DELETE FROM orders WHERE order_status = 'cancelled'",
			mode:      "write",
			wantError: false,
		},
		{
			name:      "DROP table - should fail",
			query:     "DROP TABLE users",
			mode:      "write",
			wantError: true,
			errorMsg:  "dangerous operation",
		},
		{
			name:      "TRUNCATE table - should fail",
			query:     "TRUNCATE TABLE orders",
			mode:      "write",
			wantError: true,
			errorMsg:  "dangerous operation",
		},
		{
			name:      "ALTER table - should fail",
			query:     "ALTER TABLE users ADD COLUMN middle_name VARCHAR(50)",
			mode:      "write",
			wantError: true,
			errorMsg:  "dangerous operation",
		},
		{
			name:      "CREATE table - should fail",
			query:     "CREATE TABLE new_table (id SERIAL PRIMARY KEY)",
			mode:      "write",
			wantError: true,
			errorMsg:  "dangerous operation",
		},
		{
			name:      "unauthorized table access",
			query:     "SELECT * FROM admin_users",
			mode:      "read",
			wantError: true,
			errorMsg:  "not in allowed list",
		},
		{
			name:      "multiple unauthorized tables in JOIN",
			query:     "SELECT * FROM users u JOIN unauthorized_table t ON u.id = t.user_id",
			mode:      "read",
			wantError: true,
			errorMsg:  "not in allowed list",
		},
		{
			name:      "empty query",
			query:     "",
			mode:      "read",
			wantError: true,
			errorMsg:  "cannot be empty",
		},
		{
			name:      "query with extra whitespace",
			query:     "  SELECT   *   FROM   users   WHERE   age   >   25  ",
			mode:      "read",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ValidateQuery(tt.query, tt.mode, allowedTables, nil)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if info == nil {
					t.Errorf("expected QueryInfo but got nil")
				}
			}
		})
	}
}

func TestExtractTables(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		stmtType  string
		wantCount int
	}{
		{
			name:      "SELECT single table",
			query:     "select * from users",
			stmtType:  "SELECT",
			wantCount: 1,
		},
		{
			name:      "SELECT with JOIN",
			query:     "select * from users join orders on users.id = orders.user_id",
			stmtType:  "SELECT",
			wantCount: 2,
		},
		{
			name:      "INSERT",
			query:     "insert into products (name) values ('test')",
			stmtType:  "INSERT",
			wantCount: 1,
		},
		{
			name:      "UPDATE",
			query:     "update orders set status = 'done'",
			stmtType:  "UPDATE",
			wantCount: 1,
		},
		{
			name:      "DELETE",
			query:     "delete from orders where id = 1",
			stmtType:  "DELETE",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, err := extractTableRefs(tt.query, tt.stmtType)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(refs) != tt.wantCount {
				t.Errorf("expected %d tables, got %d: %v", tt.wantCount, len(refs), refs)
			}
		})
	}
}

func TestValidateQuery_QuotedIdentifiersAndSchema(t *testing.T) {
	allowedTables := []string{"Org", "User"}
	allowedSchemas := []string{"private"}

	tests := []struct {
		name      string
		query     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "quoted table name",
			query:     `SELECT * FROM "Org" LIMIT 5`,
			wantError: false,
		},
		{
			name:      "schema-qualified quoted table",
			query:     `SELECT * FROM private."Org" LIMIT 5`,
			wantError: false,
		},
		{
			name:      "schema-qualified unquoted table",
			query:     `SELECT * FROM private."User" WHERE id = 1`,
			wantError: false,
		},
		{
			name:      "JOIN with quoted and schema-qualified",
			query:     `SELECT o.*, u.* FROM private."Org" o JOIN private."User" u ON o.id = u.org_id`,
			wantError: false,
		},
		{
			name:      "disallowed schema",
			query:     `SELECT * FROM public."Org"`,
			wantError: true,
			errorMsg:  "not in allowed list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateQuery(tt.query, "read", allowedTables, allowedSchemas)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
