package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// executeInsert performs an INSERT query on the database
//
//nolint:unused // Reserved for future use
func (pg *Postgres) executeInsert(table, addClause string) (string, error) {
	// Table is already validated by the handler

	// Build parameterized query
	qb := NewQueryBuilder(table)
	query, params, err := qb.BuildInsert(addClause)
	if err != nil {
		return "", err
	}

	// Execute query with connection management
	var response *PostgresResponse
	err = pg.executeWithConnection(func(db *sql.DB) error {
		result, err := db.Exec(query, params...)
		if err != nil {
			return fmt.Errorf("insert execution failed: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		response = &PostgresResponse{
			Operation:    "INSERT",
			Table:        table,
			RowsAffected: int(rowsAffected),
			ExecutedSQL:  query,
			Message:      fmt.Sprintf("Successfully inserted %d row(s)", rowsAffected),
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return response.String(), nil
}
