package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

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
	queryInfo, _ := ValidateQuery(query, pg.mode, pg.allowedTables, pg.allowedSchemas)

	if queryInfo.StatementType == "SELECT" {
		return pg.executeSelectQuery(db, query)
	}

	return pg.executeWrite(db, query, queryInfo.StatementType)
}

// runQuery runs the query, setting search_path first when allowedSchemas is configured.
// Returns (rows, commitOrNil, error). Call commitOrNil() before returning on success.
func (pg *Postgres) runSelect(db *sql.DB, query string) (*sql.Rows, func(), error) {
	if len(pg.allowedSchemas) == 0 {
		rows, err := db.Query(query)
		return rows, func() {}, err
	}
	parts := make([]string, len(pg.allowedSchemas))
	for i, s := range pg.allowedSchemas {
		parts[i] = `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, nil, err
	}
	if _, err := tx.Exec("SET search_path TO " + strings.Join(parts, ", ")); err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}
	rows, err := tx.Query(query)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}
	return rows, func() { _ = tx.Commit() }, nil
}

// runExec runs an exec query, setting search_path first when allowedSchemas is configured.
func (pg *Postgres) runExec(db *sql.DB, query string) (sql.Result, error) {
	if len(pg.allowedSchemas) == 0 {
		return db.Exec(query)
	}
	parts := make([]string, len(pg.allowedSchemas))
	for i, s := range pg.allowedSchemas {
		parts[i] = `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec("SET search_path TO " + strings.Join(parts, ", ")); err != nil {
		return nil, err
	}
	result, err := tx.Exec(query)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

// executeSelectQuery executes a SELECT query and returns formatted results
func (pg *Postgres) executeSelectQuery(db *sql.DB, query string) (string, error) {
	rows, commit, err := pg.runSelect(db, query)
	if err != nil {
		return "", fmt.Errorf("query execution failed: %v", err)
	}
	defer func() { _ = rows.Close(); commit() }()

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
	result, err := pg.runExec(db, query)
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
