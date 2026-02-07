package postgres

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateQuery validates a SQL query against mode and table restrictions
func ValidateQuery(query, mode string, allowedTables []string) (*QueryInfo, error) {
	query = strings.TrimSpace(query)

	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Remove extra whitespace and normalize
	query = normalizeQuery(query)

	// Detect statement type
	stmtType, err := detectStatementType(query)
	if err != nil {
		return nil, err
	}

	// Check if statement type is allowed for the mode
	if err := validateStatementForMode(stmtType, mode); err != nil {
		return nil, err
	}

	// Extract tables from the query
	tables, err := extractTables(query, stmtType)
	if err != nil {
		return nil, err
	}

	// Validate all tables are in the allowed list
	if err := validateAllTables(tables, allowedTables); err != nil {
		return nil, err
	}

	return &QueryInfo{
		StatementType: stmtType,
		Tables:        tables,
		Query:         query,
	}, nil
}

// QueryInfo contains information about a validated query
type QueryInfo struct {
	StatementType string
	Tables        []string
	Query         string
}

// normalizeQuery removes extra whitespace and normalizes the query
func normalizeQuery(query string) string {
	// Remove leading/trailing whitespace
	query = strings.TrimSpace(query)

	// Replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`\s+`)
	query = spaceRegex.ReplaceAllString(query, " ")

	return query
}

// detectStatementType determines the SQL statement type
func detectStatementType(query string) (string, error) {
	queryLower := strings.ToLower(query)

	// Check for dangerous operations first
	dangerousOps := []string{
		"drop ", "truncate ", "alter ", "create ",
		"grant ", "revoke ", "exec ", "execute ",
	}

	for _, op := range dangerousOps {
		if strings.HasPrefix(queryLower, op) {
			return "", fmt.Errorf("dangerous operation detected: %s is not allowed", strings.ToUpper(strings.TrimSpace(op)))
		}
	}

	// Detect statement type
	if strings.HasPrefix(queryLower, "select ") {
		return "SELECT", nil
	}
	if strings.HasPrefix(queryLower, "insert ") {
		return "INSERT", nil
	}
	if strings.HasPrefix(queryLower, "update ") {
		return "UPDATE", nil
	}
	if strings.HasPrefix(queryLower, "delete ") {
		return "DELETE", nil
	}

	return "", fmt.Errorf("unsupported or unrecognized SQL statement. Only SELECT, INSERT, UPDATE, and DELETE are allowed")
}

// validateStatementForMode checks if the statement type is allowed for the mode
func validateStatementForMode(stmtType, mode string) error {
	if mode == "read" {
		if stmtType != "SELECT" {
			return fmt.Errorf("operation %s is not allowed in READ mode. Only SELECT queries are permitted", stmtType)
		}
	}
	// In write mode, all statement types (SELECT, INSERT, UPDATE, DELETE) are allowed
	return nil
}

// extractTables extracts table names from a SQL query
func extractTables(query, stmtType string) ([]string, error) {
	queryLower := strings.ToLower(query)
	tables := make([]string, 0)

	switch stmtType {
	case "SELECT":
		// Extract tables from FROM and JOIN clauses
		tables = extractTablesFromSelect(queryLower)
	case "INSERT":
		// Extract table from INSERT INTO
		tables = extractTableFromInsert(queryLower)
	case "UPDATE":
		// Extract table from UPDATE
		tables = extractTableFromUpdate(queryLower)
	case "DELETE":
		// Extract table from DELETE FROM
		tables = extractTableFromDelete(queryLower)
	}

	if len(tables) == 0 {
		return nil, fmt.Errorf("could not extract table names from query")
	}

	return tables, nil
}

// extractTablesFromSelect extracts table names from SELECT statements
func extractTablesFromSelect(query string) []string {
	tables := make([]string, 0)

	// Pattern to match FROM clause
	fromRegex := regexp.MustCompile(`\bfrom\s+([a-z0-9_]+)`)
	matches := fromRegex.FindAllStringSubmatch(query, -1)
	for _, match := range matches {
		if len(match) > 1 {
			tables = append(tables, match[1])
		}
	}

	// Pattern to match JOIN clauses
	joinRegex := regexp.MustCompile(`\bjoin\s+([a-z0-9_]+)`)
	matches = joinRegex.FindAllStringSubmatch(query, -1)
	for _, match := range matches {
		if len(match) > 1 {
			tables = append(tables, match[1])
		}
	}

	return tables
}

// extractTableFromInsert extracts table name from INSERT statements
func extractTableFromInsert(query string) []string {
	// Pattern to match INSERT INTO table_name
	insertRegex := regexp.MustCompile(`\binsert\s+into\s+([a-z0-9_]+)`)
	matches := insertRegex.FindStringSubmatch(query)
	if len(matches) > 1 {
		return []string{matches[1]}
	}
	return []string{}
}

// extractTableFromUpdate extracts table name from UPDATE statements
func extractTableFromUpdate(query string) []string {
	// Pattern to match UPDATE table_name
	updateRegex := regexp.MustCompile(`\bupdate\s+([a-z0-9_]+)`)
	matches := updateRegex.FindStringSubmatch(query)
	if len(matches) > 1 {
		return []string{matches[1]}
	}
	return []string{}
}

// extractTableFromDelete extracts table name from DELETE statements
func extractTableFromDelete(query string) []string {
	// Pattern to match DELETE FROM table_name
	deleteRegex := regexp.MustCompile(`\bdelete\s+from\s+([a-z0-9_]+)`)
	matches := deleteRegex.FindStringSubmatch(query)
	if len(matches) > 1 {
		return []string{matches[1]}
	}
	return []string{}
}

// validateAllTables checks if all extracted tables are in the allowed list
func validateAllTables(tables []string, allowedTables []string) error {
	allowedMap := make(map[string]bool)
	for _, table := range allowedTables {
		allowedMap[strings.ToLower(table)] = true
	}

	unauthorizedTables := make([]string, 0)
	for _, table := range tables {
		if !allowedMap[strings.ToLower(table)] {
			unauthorizedTables = append(unauthorizedTables, table)
		}
	}

	if len(unauthorizedTables) > 0 {
		return fmt.Errorf("access denied: table(s) %v not in allowed list. Allowed tables: %v",
			unauthorizedTables, allowedTables)
	}

	return nil
}
