package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

// executeGetTables retrieves the list of tables in the specified schema
func (pg *Postgres) executeGetTables(schema string) (string, error) {
	// Validate schema name
	if err := validateSchemaName(schema); err != nil {
		return "", err
	}

	// Build query to get tables
	query := `
		SELECT table_name, table_type
		FROM information_schema.tables
		WHERE table_schema = $1
		ORDER BY table_name
	`

	// Execute query with connection management
	var response *PostgresResponse
	err := pg.executeWithConnection(func(db *sql.DB) error {
		rows, err := db.Query(query, schema)
		if err != nil {
			return fmt.Errorf("failed to retrieve tables: %w", err)
		}
		defer rows.Close()

		var allTables []string
		var filteredTables []string

		for rows.Next() {
			var tableName, tableType string
			if err := rows.Scan(&tableName, &tableType); err != nil {
				return fmt.Errorf("failed to scan table row: %w", err)
			}

			allTables = append(allTables, fmt.Sprintf("%s (%s)", tableName, tableType))

			// Filter based on allowed tables
			if pg.isTableAllowed(tableName) {
				filteredTables = append(filteredTables, fmt.Sprintf("%s (%s)", tableName, tableType))
			}
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating table rows: %w", err)
		}

		// Format response
		var dataPreview string
		if len(filteredTables) == 0 {
			dataPreview = fmt.Sprintf("No accessible tables found in schema '%s'.\nYou have access to tables: %v", schema, pg.allowedTables)
		} else {
			dataPreview = fmt.Sprintf("Accessible tables in schema '%s':\n%s\n\nTotal accessible: %d\nAllowed tables: %v",
				schema,
				strings.Join(filteredTables, "\n"),
				len(filteredTables),
				pg.allowedTables,
			)
		}

		response = &PostgresResponse{
			Operation:    "GET_TABLES",
			Table:        schema,
			RowsReturned: len(filteredTables),
			DataPreview:  dataPreview,
			ExecutedSQL:  query,
			Message:      fmt.Sprintf("Retrieved %d accessible tables from schema '%s'", len(filteredTables), schema),
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return response.String(), nil
}

// isTableAllowed checks if a table is in the allowed tables list
func (pg *Postgres) isTableAllowed(tableName string) bool {
	for _, allowed := range pg.allowedTables {
		if tableName == allowed {
			return true
		}
	}
	return false
}
