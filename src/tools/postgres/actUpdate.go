package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// executeUpdate performs an UPDATE query on the database
func (pg *Postgres) executeUpdate(table, updateClause, whereClause string) (string, error) {
	// Table is already validated by the handler

	// Build parameterized query
	qb := NewQueryBuilder(table)
	query, params, err := qb.BuildUpdate(updateClause, whereClause)
	if err != nil {
		return "", err
	}

	// Execute query with connection management
	var response *PostgresResponse
	err = pg.executeWithConnection(func(db *sql.DB) error {
		result, err := db.Exec(query, params...)
		if err != nil {
			return fmt.Errorf("update execution failed: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		response = &PostgresResponse{
			Operation:    "UPDATE",
			Table:        table,
			RowsAffected: int(rowsAffected),
			ExecutedSQL:  query,
			Message:      fmt.Sprintf("Successfully updated %d row(s)", rowsAffected),
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return response.String(), nil
}
