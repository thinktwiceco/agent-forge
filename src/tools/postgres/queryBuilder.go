package postgres

import (
	"fmt"
	"regexp"
	"strings"
)

// QueryBuilder constructs parameterized SQL queries from fragments
type QueryBuilder struct {
	table  string
	params []interface{}
}

// NewQueryBuilder creates a new query builder for a specific table
func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{
		table:  table,
		params: make([]interface{}, 0),
	}
}

// BuildSelect constructs a SELECT query with parameterized WHERE clause
func (qb *QueryBuilder) BuildSelect(selectClause string, limit, offset int) (string, []interface{}, error) {
	// Validate the select clause
	if err := validateSQLFragment(selectClause, "select"); err != nil {
		return "", nil, err
	}

	// Validate limit and offset
	if err := validateLimit(limit); err != nil {
		return "", nil, err
	}
	if err := validateOffset(offset); err != nil {
		return "", nil, err
	}

	// Parse the select clause to separate columns and WHERE clause
	columns, whereClause, err := qb.parseSelectClause(selectClause)
	if err != nil {
		return "", nil, err
	}

	// Start building the query
	query := fmt.Sprintf("SELECT %s FROM %s", columns, qb.table)

	// Add WHERE clause if present
	if whereClause != "" {
		parameterizedWhere, err := qb.parameterizeWhereClause(whereClause)
		if err != nil {
			return "", nil, fmt.Errorf("failed to parameterize WHERE clause: %w", err)
		}
		query += " WHERE " + parameterizedWhere
	}

	// Add LIMIT and OFFSET as parameters
	qb.params = append(qb.params, limit, offset)
	limitPlaceholder := fmt.Sprintf("$%d", len(qb.params)-1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(qb.params))

	query += fmt.Sprintf(" LIMIT %s OFFSET %s", limitPlaceholder, offsetPlaceholder)

	return query, qb.params, nil
}

// BuildUpdate constructs an UPDATE query with parameterized SET and WHERE clauses
func (qb *QueryBuilder) BuildUpdate(updateClause, whereClause string) (string, []interface{}, error) {
	// Validate the update clause
	if err := validateSQLFragment(updateClause, "update"); err != nil {
		return "", nil, err
	}

	// Ensure WHERE clause is provided
	if strings.TrimSpace(whereClause) == "" {
		return "", nil, fmt.Errorf(`UPDATE operation requires a WHERE clause to prevent accidental full table updates.

✗ Bad:  update: "status = 'active'"
✓ Good: update: "status = 'active' WHERE user_id = 123"

Provide the WHERE clause in the 'select' parameter.`)
	}

	// Parse and parameterize SET clause
	setClause, err := qb.parameterizeSetClause(updateClause)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parameterize SET clause: %w", err)
	}

	// Parse and parameterize WHERE clause
	parameterizedWhere, err := qb.parameterizeWhereClause(whereClause)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parameterize WHERE clause: %w", err)
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", qb.table, setClause, parameterizedWhere)

	return query, qb.params, nil
}

// BuildInsert constructs an INSERT query with parameterized values
func (qb *QueryBuilder) BuildInsert(columnsAndValues string) (string, []interface{}, error) {
	// Validate the insert clause
	if err := validateSQLFragment(columnsAndValues, "add"); err != nil {
		return "", nil, err
	}

	// Parse columns and values
	columns, values, err := qb.parseInsertClause(columnsAndValues)
	if err != nil {
		return "", nil, err
	}

	// Parameterize values
	valuePlaceholders, err := qb.parameterizeValues(values)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parameterize values: %w", err)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", qb.table, columns, valuePlaceholders)

	return query, qb.params, nil
}

// parseSelectClause separates column list from WHERE clause
func (qb *QueryBuilder) parseSelectClause(selectClause string) (string, string, error) {
	// Look for WHERE keyword (case-insensitive)
	whereRegex := regexp.MustCompile(`(?i)\s+WHERE\s+`)
	parts := whereRegex.Split(selectClause, 2)

	if len(parts) == 1 {
		// No WHERE clause
		return strings.TrimSpace(parts[0]), "", nil
	}

	columns := strings.TrimSpace(parts[0])
	whereClause := strings.TrimSpace(parts[1])

	if columns == "" {
		return "", "", fmt.Errorf("column list cannot be empty in SELECT clause")
	}

	return columns, whereClause, nil
}

// parseInsertClause separates column list from VALUES clause
func (qb *QueryBuilder) parseInsertClause(columnsAndValues string) (string, string, error) {
	// Look for VALUES keyword (case-insensitive)
	valuesRegex := regexp.MustCompile(`(?i)\s+VALUES\s+`)
	parts := valuesRegex.Split(columnsAndValues, 2)

	if len(parts) != 2 {
		return "", "", fmt.Errorf("INSERT clause must contain 'VALUES' keyword. Format: 'col1, col2 VALUES (val1, val2)'")
	}

	columns := strings.TrimSpace(parts[0])
	values := strings.TrimSpace(parts[1])

	if columns == "" {
		return "", "", fmt.Errorf("column list cannot be empty in INSERT clause")
	}
	if values == "" {
		return "", "", fmt.Errorf("values cannot be empty in INSERT clause")
	}

	return columns, values, nil
}

// parameterizeWhereClause converts WHERE clause conditions to parameterized format
func (qb *QueryBuilder) parameterizeWhereClause(whereClause string) (string, error) {
	// Remove "WHERE" keyword if present at the start
	whereClause = regexp.MustCompile(`(?i)^WHERE\s+`).ReplaceAllString(whereClause, "")
	whereClause = strings.TrimSpace(whereClause)

	if whereClause == "" {
		return "", nil
	}

	// Handle simple parameterization: find quoted strings and numbers
	// This is a simplified approach - for production, use a proper SQL parser
	result := whereClause
	paramCount := len(qb.params)

	// Pattern to match quoted strings (single or double quotes)
	stringPattern := regexp.MustCompile(`'([^']*)'|"([^"]*)"`)
	result = stringPattern.ReplaceAllStringFunc(result, func(match string) string {
		// Extract the value without quotes
		value := match[1 : len(match)-1]
		qb.params = append(qb.params, value)
		paramCount++
		return fmt.Sprintf("$%d", paramCount)
	})

	// Pattern to match standalone numbers (not part of function calls or column names)
	// Match numbers that are preceded by operators or whitespace
	numberPattern := regexp.MustCompile(`([\s=<>!]+)(\d+\.?\d*)(\s|$|,|AND|OR|;)`)
	result = numberPattern.ReplaceAllStringFunc(result, func(match string) string {
		matches := numberPattern.FindStringSubmatch(match)
		if len(matches) >= 4 {
			prefix := matches[1]
			number := matches[2]
			suffix := matches[3]

			// Try to parse as float or int
			var value interface{}
			if strings.Contains(number, ".") {
				var f float64
				fmt.Sscanf(number, "%f", &f)
				value = f
			} else {
				var i int
				fmt.Sscanf(number, "%d", &i)
				value = i
			}

			qb.params = append(qb.params, value)
			paramCount++
			return fmt.Sprintf("%s$%d%s", prefix, paramCount, suffix)
		}
		return match
	})

	return result, nil
}

// parameterizeSetClause converts SET clause assignments to parameterized format
func (qb *QueryBuilder) parameterizeSetClause(setClause string) (string, error) {
	// Split by comma to handle multiple assignments
	assignments := strings.Split(setClause, ",")
	parameterizedAssignments := make([]string, 0, len(assignments))

	for _, assignment := range assignments {
		assignment = strings.TrimSpace(assignment)

		// Check for SQL functions (NOW(), CURRENT_TIMESTAMP, etc.) - don't parameterize these
		if qb.containsSQLFunction(assignment) {
			parameterizedAssignments = append(parameterizedAssignments, assignment)
			continue
		}

		// Split by = to get column and value
		parts := strings.SplitN(assignment, "=", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid SET assignment: '%s'. Expected format: 'column = value'", assignment)
		}

		column := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Parameterize the value
		qb.params = append(qb.params, qb.parseValue(value))
		placeholder := fmt.Sprintf("$%d", len(qb.params))

		parameterizedAssignments = append(parameterizedAssignments, fmt.Sprintf("%s = %s", column, placeholder))
	}

	return strings.Join(parameterizedAssignments, ", "), nil
}

// parameterizeValues converts VALUES list to parameterized format
func (qb *QueryBuilder) parameterizeValues(values string) (string, error) {
	// Remove surrounding parentheses if present
	values = strings.TrimSpace(values)
	if strings.HasPrefix(values, "(") && strings.HasSuffix(values, ")") {
		values = values[1 : len(values)-1]
	}

	// Split by comma to get individual values
	valueParts := qb.splitValues(values)
	placeholders := make([]string, 0, len(valueParts))

	for _, value := range valueParts {
		value = strings.TrimSpace(value)

		// Check for SQL functions - don't parameterize these
		if qb.containsSQLFunction(value) {
			placeholders = append(placeholders, value)
			continue
		}

		// Parameterize the value
		qb.params = append(qb.params, qb.parseValue(value))
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(qb.params)))
	}

	return strings.Join(placeholders, ", "), nil
}

// splitValues splits a comma-separated value list, respecting quotes
func (qb *QueryBuilder) splitValues(values string) []string {
	result := make([]string, 0)
	current := ""
	inQuote := false
	quoteChar := rune(0)

	for _, char := range values {
		if (char == '\'' || char == '"') && (quoteChar == 0 || quoteChar == char) {
			if inQuote && quoteChar == char {
				inQuote = false
				quoteChar = 0
			} else if !inQuote {
				inQuote = true
				quoteChar = char
			}
			current += string(char)
		} else if char == ',' && !inQuote {
			result = append(result, strings.TrimSpace(current))
			current = ""
		} else {
			current += string(char)
		}
	}

	if current != "" {
		result = append(result, strings.TrimSpace(current))
	}

	return result
}

// parseValue removes quotes and parses the value
func (qb *QueryBuilder) parseValue(value string) interface{} {
	value = strings.TrimSpace(value)

	// Remove quotes if present
	if (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) ||
		(strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) {
		return value[1 : len(value)-1]
	}

	// Try to parse as number
	if strings.Contains(value, ".") {
		var f float64
		if _, err := fmt.Sscanf(value, "%f", &f); err == nil {
			return f
		}
	} else {
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}

	// Return as string if can't parse as number
	return value
}

// containsSQLFunction checks if a string contains SQL function calls
func (qb *QueryBuilder) containsSQLFunction(value string) bool {
	upperValue := strings.ToUpper(strings.TrimSpace(value))

	// Common SQL functions that should not be parameterized
	sqlFunctions := []string{
		"NOW()", "CURRENT_TIMESTAMP", "CURRENT_DATE", "CURRENT_TIME",
		"NULL", "TRUE", "FALSE", "DEFAULT",
	}

	for _, fn := range sqlFunctions {
		if strings.Contains(upperValue, fn) {
			return true
		}
	}

	// Check for function call pattern: word followed by parentheses
	functionPattern := regexp.MustCompile(`[A-Z_]+\s*\(`)
	return functionPattern.MatchString(upperValue)
}
