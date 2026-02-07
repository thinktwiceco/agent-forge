package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// executeSelect performs a SELECT query on the database
func (pg *Postgres) executeSelect(table, selectClause string, limit, offset int) (string, error) {
	// Table is already validated by the handler

	// Build parameterized query
	qb := NewQueryBuilder(table)
	query, params, err := qb.BuildSelect(selectClause, limit, offset)
	if err != nil {
		return "", err
	}

	// Execute query with connection management
	var response *PostgresResponse
	err = pg.executeWithConnection(func(db *sql.DB) error {
		rows, err := db.Query(query, params...)
		if err != nil {
			return fmt.Errorf("query execution failed: %w", err)
		}
		defer rows.Close()

		// Get column names
		columns, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("failed to get column names: %w", err)
		}

		// Fetch all rows
		var allRows [][]interface{}
		for rows.Next() {
			// Create a slice of interface{} to hold each column value
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return fmt.Errorf("failed to scan row: %w", err)
			}

			allRows = append(allRows, values)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating rows: %w", err)
		}

		// Format response
		dataPreview := formatDataPreview(columns, allRows, 10) // Show max 10 rows in preview

		response = &PostgresResponse{
			Operation:    "SELECT",
			Table:        table,
			RowsReturned: len(allRows),
			DataPreview:  dataPreview,
			ExecutedSQL:  query,
			Message:      fmt.Sprintf("Successfully retrieved %d rows", len(allRows)),
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return response.String(), nil
}

// connect establishes a connection to the PostgreSQL database
func (pg *Postgres) connect() (*sql.DB, error) {
	db, err := sql.Open("postgres", pg.postgresURL)
	if err != nil {
		return nil, fmt.Errorf(`failed to open database connection.
Error: %v

Check your postgres_url format:
Expected: postgresql://user:password@host:port/database
Example: postgresql://myuser:mypass@localhost:5432/mydb`, err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf(`failed to connect to database.
Error: %v

Possible issues:
- Database server is not running
- Incorrect host or port
- Invalid credentials
- Database does not exist
- Network connectivity issues
- Firewall blocking connection`, err)
	}

	// Set connection limits
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)

	return db, nil
}

// executeWithConnection manages the database connection lifecycle
func (pg *Postgres) executeWithConnection(queryFunc func(*sql.DB) error) error {
	db, err := pg.connect()
	if err != nil {
		return err
	}
	defer db.Close()

	return queryFunc(db)
}
