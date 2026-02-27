package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

// getTableSchema retrieves column information for a given table
func (pg *Postgres) getTableSchema(table string) (string, error) {
	// Parse table name: supports "schema.table" or just "table"
	schema, tableName := parseTableName(table, pg.allowedSchemas)

	// Open database connection
	db, err := sql.Open("postgres", pg.postgresURL)
	if err != nil {
		return "", fmt.Errorf("failed to connect to database.\nError: %v\n\nPossible issues:\n- Database server is not running\n- Incorrect host or port\n- Invalid credentials\n- Database does not exist\n- Network connectivity issues\n- Firewall blocking connection", err)
	}
	defer func() { _ = db.Close() }()

	// Test the connection
	if err := db.Ping(); err != nil {
		return "", fmt.Errorf("failed to ping database.\nError: %v\n\nThe connection parameters are correct but the database is not responding", err)
	}

	// Validate table is in allowed list
	fullTable := tableName
	if schema != "" && schema != "public" {
		fullTable = schema + "." + tableName
	}
	if !isTableAllowed(fullTable, tableName, pg.allowedTables) {
		return "", fmt.Errorf("access denied: table %q not in allowed list: %v", fullTable, pg.allowedTables)
	}

	// Query information_schema for column details
	query := `
		SELECT column_name, data_type, is_nullable, column_default, udt_name
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := db.Query(query, schema, tableName)
	if err != nil {
		return "", fmt.Errorf("query execution failed: %v", err)
	}
	defer func() { _ = rows.Close() }()

	// Fetch all column rows
	columns := make([]map[string]interface{}, 0)
	for rows.Next() {
		var columnName, dataType, isNullable string
		var columnDefault *string
		var udtName string

		if err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault, &udtName); err != nil {
			return "", fmt.Errorf("failed to scan row: %v", err)
		}

		col := map[string]interface{}{
			"name":     columnName,
			"type":     dataType,
			"nullable": isNullable,
		}

		if columnDefault != nil {
			col["default"] = *columnDefault
		} else {
			col["default"] = nil
		}

		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating rows: %v", err)
	}

	// Format response
	response := map[string]interface{}{
		"operation": "GET_TABLE_SCHEMA",
		"schema":    schema,
		"table":     tableName,
		"columns":   columns,
	}

	if len(columns) == 0 {
		response["message"] = "Table exists but has no columns or was not found"
	} else {
		response["message"] = fmt.Sprintf("Retrieved schema for table %q with %d column(s)", fullTable, len(columns))
	}

	// Convert to JSON for clean output
	jsonOutput, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format output: %v", err)
	}

	return string(jsonOutput), nil
}

// parseTableName parses a table reference in "schema.table" or "table" format
// Returns (schema, table). If no schema is provided, uses the first allowed schema or "public"
func parseTableName(table string, allowedSchemas []string) (string, string) {
	parts := strings.Split(table, ".")
	if len(parts) == 2 {
		// schema.table format
		return parts[0], parts[1]
	}

	// Just table name; determine schema
	schema := "public"
	if len(allowedSchemas) > 0 {
		schema = allowedSchemas[0]
	}
	return schema, parts[0]
}

// isTableAllowed checks if a table is in the allowed list
func isTableAllowed(fullTable, tableName string, allowedTables []string) bool {
	for _, allowed := range allowedTables {
		// Match both "table" and "schema.table" formats
		if allowed == tableName || allowed == fullTable {
			return true
		}
	}
	return false
}
