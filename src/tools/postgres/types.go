package postgres

import (
	"fmt"
	"strings"
)

// PostgresResponse represents the result of a database operation
type PostgresResponse struct {
	Operation    string
	Table        string
	RowsAffected int
	RowsReturned int
	DataPreview  string
	ExecutedSQL  string
	Message      string
}

// String formats the PostgresResponse for display to the agent
func (r *PostgresResponse) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Operation: %s\n", r.Operation)
	fmt.Fprintf(&sb, "Table: %s\n", r.Table)

	if r.RowsAffected > 0 {
		fmt.Fprintf(&sb, "Rows Affected: %d\n", r.RowsAffected)
	}

	if r.RowsReturned > 0 {
		fmt.Fprintf(&sb, "Rows Returned: %d\n", r.RowsReturned)
	}

	if r.Message != "" {
		fmt.Fprintf(&sb, "Message: %s\n", r.Message)
	}

	if r.ExecutedSQL != "" {
		fmt.Fprintf(&sb, "\nExecuted Query (parameterized):\n%s\n", r.ExecutedSQL)
	}

	if r.DataPreview != "" {
		fmt.Fprintf(&sb, "\nData Preview:\n%s\n", r.DataPreview)
	}

	return sb.String()
}

// formatDataPreview formats query results for preview
//
//nolint:unused // Used internally by executeSelect
func formatDataPreview(columns []string, rows [][]interface{}, maxRows int) string {
	if len(rows) == 0 {
		return "No data returned."
	}

	var sb strings.Builder

	// Header
	sb.WriteString(strings.Join(columns, " | "))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("-", len(sb.String())-1))
	sb.WriteString("\n")

	// Rows (limited to maxRows)
	displayRows := len(rows)
	if displayRows > maxRows {
		displayRows = maxRows
	}

	for i := 0; i < displayRows; i++ {
		rowValues := make([]string, len(rows[i]))
		for j, val := range rows[i] {
			rowValues[j] = formatValue(val)
		}
		sb.WriteString(strings.Join(rowValues, " | "))
		sb.WriteString("\n")
	}

	if len(rows) > maxRows {
		fmt.Fprintf(&sb, "\n... and %d more rows (use limit/offset to view more)\n", len(rows)-maxRows)
	}

	return sb.String()
}

// formatValue converts an interface{} value to a display string
//
//nolint:unused // Used internally by formatDataPreview
func formatValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}

	switch v := val.(type) {
	case string:
		// Truncate long strings
		if len(v) > 50 {
			return v[:50] + "..."
		}
		return v
	case []byte:
		// Convert byte arrays to string
		str := string(v)
		if len(str) > 50 {
			return str[:50] + "..."
		}
		return str
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
