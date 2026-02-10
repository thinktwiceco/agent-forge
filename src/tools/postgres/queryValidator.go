package postgres

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateQuery validates a SQL query against mode, table, and schema restrictions.
// allowedSchemas can be nil/empty to skip schema validation.
func ValidateQuery(query, mode string, allowedTables []string, allowedSchemas []string) (*QueryInfo, error) {
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

	// Extract tables from the query (use original query to preserve quoted identifier case)
	tableRefs, err := extractTableRefs(query, stmtType)
	if err != nil {
		return nil, err
	}

	// Validate all tables are in the allowed list
	if err := validateTableRefs(tableRefs, allowedTables, allowedSchemas); err != nil {
		return nil, err
	}

	// Build table names list for QueryInfo (for backward compatibility)
	tableNames := make([]string, len(tableRefs))
	for i, ref := range tableRefs {
		tableNames[i] = ref.Table
	}

	return &QueryInfo{
		StatementType: stmtType,
		Tables:        tableNames,
		Query:         query,
	}, nil
}

// TableRef represents a table reference extracted from a query (may include schema)
type TableRef struct {
	Schema string // empty if unqualified
	Table  string // table name, stripped of quotes, preserved case for quoted
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

// tableRefPattern matches PostgreSQL table references:
// - table or "table" (unquoted folds to lowercase, quoted preserves case)
// - schema.table or schema."table" or "schema"."table"
var (
	// Matches table ref after FROM, JOIN, INTO, UPDATE - supports schema.table and "quoted" identifiers
	tableRefPattern = regexp.MustCompile(`(?i)(?:from|join|into|update)\s+((?:[a-zA-Z0-9_]+|"[^"]*")\.)?([a-zA-Z0-9_]+|"[^"]*")(?:\s|$|,|\))`)
	deleteFromPat   = regexp.MustCompile(`(?i)delete\s+from\s+((?:[a-zA-Z0-9_]+|"[^"]*")\.)?([a-zA-Z0-9_]+|"[^"]*")(?:\s|$|,|\))`)
)

// parseTableRef extracts schema and table from a regex match, normalizing quoted identifiers
func parseTableRef(schemaPart, tablePart string) TableRef {
	ref := TableRef{}
	if schemaPart != "" {
		ref.Schema = strings.TrimRight(schemaPart, ".")
		if len(ref.Schema) >= 2 && ref.Schema[0] == '"' {
			ref.Schema = ref.Schema[1 : len(ref.Schema)-1]
		} else {
			ref.Schema = strings.ToLower(ref.Schema)
		}
	}
	if len(tablePart) >= 2 && tablePart[0] == '"' {
		ref.Table = tablePart[1 : len(tablePart)-1] // preserve case for quoted
	} else {
		ref.Table = strings.ToLower(tablePart)
	}
	return ref
}

// extractTableRefs extracts table references from a SQL query (supports quoted identifiers and schema.table)
func extractTableRefs(query, stmtType string) ([]TableRef, error) {
	refs := make([]TableRef, 0)
	seen := make(map[string]bool)

	addRef := func(ref TableRef) {
		key := ref.Schema + "." + ref.Table
		if ref.Schema == "" {
			key = ref.Table
		}
		if !seen[key] {
			seen[key] = true
			refs = append(refs, ref)
		}
	}

	switch stmtType {
	case "SELECT":
		for _, m := range tableRefPattern.FindAllStringSubmatch(query, -1) {
			if len(m) >= 3 {
				addRef(parseTableRef(m[1], m[2]))
			}
		}
	case "INSERT":
		if m := tableRefPattern.FindStringSubmatch(query); len(m) >= 3 {
			addRef(parseTableRef(m[1], m[2]))
		}
	case "UPDATE":
		if m := tableRefPattern.FindStringSubmatch(query); len(m) >= 3 {
			addRef(parseTableRef(m[1], m[2]))
		}
	case "DELETE":
		if m := deleteFromPat.FindStringSubmatch(query); len(m) >= 3 {
			addRef(parseTableRef(m[1], m[2]))
		}
	}

	if len(refs) == 0 {
		return nil, fmt.Errorf("could not extract table names from query")
	}
	return refs, nil
}

// validateTableRefs checks that all extracted tables are allowed and schemas (if specified) are allowed
func validateTableRefs(refs []TableRef, allowedTables []string, allowedSchemas []string) error {
	allowedTableMap := make(map[string]bool)
	for _, t := range allowedTables {
		allowedTableMap[strings.ToLower(t)] = true
	}
	allowedSchemaMap := make(map[string]bool)
	for _, s := range allowedSchemas {
		allowedSchemaMap[strings.ToLower(s)] = true
	}

	unauthorized := make([]string, 0)
	for _, ref := range refs {
		tableKey := strings.ToLower(ref.Table)
		if !allowedTableMap[tableKey] {
			if ref.Schema != "" {
				unauthorized = append(unauthorized, ref.Schema+"."+ref.Table)
			} else {
				unauthorized = append(unauthorized, ref.Table)
			}
			continue
		}
		if ref.Schema != "" && len(allowedSchemas) > 0 && !allowedSchemaMap[strings.ToLower(ref.Schema)] {
			unauthorized = append(unauthorized, ref.Schema+"."+ref.Table+" (schema not in allowed list)")
		}
	}

	if len(unauthorized) > 0 {
		return fmt.Errorf("access denied: table(s) %v not in allowed list. Allowed tables: %v",
			unauthorized, allowedTables)
	}
	return nil
}
