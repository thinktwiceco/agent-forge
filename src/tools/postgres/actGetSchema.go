package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

// executeGetSchema retrieves the schema (column information) for a specified table
func (pg *Postgres) executeGetSchema(table, schema string) (string, error) {
	// Validate schema and table names
	if err := validateSchemaName(schema); err != nil {
		return "", err
	}

	// Table is already validated by the handler, but double-check
	if err := validateTable(table, pg.allowedTables); err != nil {
		return "", err
	}

	// Build query to get column information
	query := `
		SELECT 
			column_name,
			data_type,
			is_nullable,
			column_default,
			character_maximum_length,
			numeric_precision,
			numeric_scale
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	// Execute query with connection management
	var response *PostgresResponse
	err := pg.executeWithConnection(func(db *sql.DB) error {
		rows, err := db.Query(query, schema, table)
		if err != nil {
			return fmt.Errorf("failed to retrieve schema: %w", err)
		}
		defer rows.Close()

		var columns []string
		columnCount := 0

		for rows.Next() {
			var columnName, dataType, isNullable string
			var columnDefault sql.NullString
			var charMaxLength, numericPrecision, numericScale sql.NullInt64

			if err := rows.Scan(
				&columnName,
				&dataType,
				&isNullable,
				&columnDefault,
				&charMaxLength,
				&numericPrecision,
				&numericScale,
			); err != nil {
				return fmt.Errorf("failed to scan column row: %w", err)
			}

			// Build column description
			columnInfo := fmt.Sprintf("  %s: %s", columnName, dataType)

			// Add type details if available
			if charMaxLength.Valid {
				columnInfo += fmt.Sprintf("(%d)", charMaxLength.Int64)
			} else if numericPrecision.Valid && numericScale.Valid {
				columnInfo += fmt.Sprintf("(%d,%d)", numericPrecision.Int64, numericScale.Int64)
			} else if numericPrecision.Valid {
				columnInfo += fmt.Sprintf("(%d)", numericPrecision.Int64)
			}

			// Add nullable info
			if isNullable == "NO" {
				columnInfo += " NOT NULL"
			}

			// Add default value if present
			if columnDefault.Valid && columnDefault.String != "" {
				columnInfo += fmt.Sprintf(" DEFAULT %s", columnDefault.String)
			}

			columns = append(columns, columnInfo)
			columnCount++
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating column rows: %w", err)
		}

		// Check if table exists but has no columns (shouldn't happen normally)
		if columnCount == 0 {
			return fmt.Errorf("table '%s' not found in schema '%s' or has no columns", table, schema)
		}

		// Format response
		dataPreview := fmt.Sprintf("Schema for table '%s.%s':\n\n%s\n\nTotal columns: %d",
			schema,
			table,
			strings.Join(columns, "\n"),
			columnCount,
		)

		response = &PostgresResponse{
			Operation:    "GET_SCHEMA",
			Table:        fmt.Sprintf("%s.%s", schema, table),
			RowsReturned: columnCount,
			DataPreview:  dataPreview,
			ExecutedSQL:  query,
			Message:      fmt.Sprintf("Retrieved schema for table '%s.%s' with %d columns", schema, table, columnCount),
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return response.String(), nil
}
