package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/lib/pq"
)

// executeQuery executes a validated SQL query
func (pg *Postgres) executeQuery(query string) (string, error) {
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

	// Detect if this is a read or write operation
	queryInfo, _ := ValidateQuery(query, pg.mode, pg.allowedTables)

	if queryInfo.StatementType == "SELECT" {
		return pg.executeSelectQuery(db, query)
	}

	return pg.executeWrite(db, query, queryInfo.StatementType)
}

// executeSelectQuery executes a SELECT query and returns formatted results
func (pg *Postgres) executeSelectQuery(db *sql.DB, query string) (string, error) {
	rows, err := db.Query(query)
	if err != nil {
		return "", fmt.Errorf("query execution failed: %v", err)
	}
	defer func() { _ = rows.Close() }()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("failed to get column names: %v", err)
	}

	// Fetch all rows
	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		// Create a slice to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		if err := rows.Scan(valuePtrs...); err != nil {
			return "", fmt.Errorf("failed to scan row: %v", err)
		}

		// Create a map for this row
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]

			// Convert byte arrays to strings
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating rows: %v", err)
	}

	// Format response
	response := map[string]interface{}{
		"operation":     "SELECT",
		"rows_returned": len(results),
		"data":          results,
		"executed_sql":  query,
	}

	if len(results) == 0 {
		response["message"] = "Query executed successfully but returned no rows"
	}

	// Convert to JSON for clean output
	jsonOutput, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format output: %v", err)
	}

	return string(jsonOutput), nil
}

// executeWrite executes INSERT, UPDATE, or DELETE queries
func (pg *Postgres) executeWrite(db *sql.DB, query, stmtType string) (string, error) {
	result, err := db.Exec(query)
	if err != nil {
		return "", fmt.Errorf("query execution failed: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("failed to get rows affected: %v", err)
	}

	// Format response
	response := map[string]interface{}{
		"operation":     stmtType,
		"rows_affected": rowsAffected,
		"executed_sql":  query,
		"status":        "success",
	}

	if rowsAffected == 0 {
		response["message"] = "Query executed successfully but no rows were affected"
	} else {
		response["message"] = fmt.Sprintf("Successfully %s %d row(s)",
			map[string]string{
				"INSERT": "inserted",
				"UPDATE": "updated",
				"DELETE": "deleted",
			}[stmtType], rowsAffected)
	}

	// Convert to JSON for clean output
	jsonOutput, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format output: %v", err)
	}

	return string(jsonOutput), nil
}
