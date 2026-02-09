package postgres

import (
	"fmt"
	"regexp"
	"strings"
)

// validateMode ensures the mode is either "read" or "write"
//
//nolint:unused // Reserved for future use
func validateMode(value any) error {
	mode, ok := value.(string)
	if !ok {
		return fmt.Errorf("mode must be a string")
	}

	validModes := map[string]bool{
		"read":  true,
		"write": true,
	}

	if !validModes[mode] {
		return fmt.Errorf("invalid mode: %s. Must be 'read' or 'write'", mode)
	}

	return nil
}

// validateOperation ensures the operation is valid
//
//nolint:unused // Reserved for future use
func validateOperation(value any) error {
	operation, ok := value.(string)
	if !ok {
		return fmt.Errorf("operation must be a string")
	}

	validOperations := map[string]bool{
		"select":    true,
		"update":    true,
		"insert":    true,
		"getTables": true,
		"getSchema": true,
	}

	if !validOperations[operation] {
		return fmt.Errorf("invalid operation: %s. Must be 'select', 'update', 'insert', 'getTables', or 'getSchema'", operation)
	}

	return nil
}

// validateModeOperation checks if the operation is allowed in the given mode
//
//nolint:unused // Reserved for future use
func validateModeOperation(mode, operation string) error {
	if mode == "read" {
		// READ mode allows: select, getTables, getSchema
		readOperations := map[string]bool{
			"select":    true,
			"getTables": true,
			"getSchema": true,
		}
		if !readOperations[operation] {
			return fmt.Errorf(`operation '%s' is not allowed in READ mode.
Current mode: read
Allowed operations in READ mode: select, getTables, getSchema
To perform %s operations, use mode: "write"`, operation, operation)
		}
	}

	return nil
}

// validateTable checks if the table is in the allowed tables whitelist
//
//nolint:unused // Used internally by executeGetSchema
func validateTable(table string, allowedTables []string) error {
	// Check for empty table name
	if strings.TrimSpace(table) == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	// Check for SQL injection attempts in table name
	if err := validateTableName(table); err != nil {
		return err
	}

	// Check if table is in whitelist
	found := false
	for _, allowed := range allowedTables {
		if table == allowed {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf(`table '%s' is not in the allowed tables list.
Available tables: %v
Use one of the allowed tables from the whitelist`, table, allowedTables)
	}

	return nil
}

// validateTableName checks table name for injection attempts
//
//nolint:unused // Used internally by validateTable
func validateTableName(tableName string) error {
	// Check for special characters that could indicate injection
	// Allow only alphanumeric characters and underscores
	validTableName := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !validTableName.MatchString(tableName) {
		return fmt.Errorf("invalid table name: '%s'. Table names can only contain letters, numbers, and underscores", tableName)
	}

	// Check for SQL keywords in table name
	lowerTable := strings.ToLower(tableName)
	prohibitedKeywords := []string{
		"select", "insert", "update", "delete", "drop", "create", "alter",
		"truncate", "union", "join", "where", "from", "into",
	}

	for _, keyword := range prohibitedKeywords {
		if lowerTable == keyword {
			return fmt.Errorf("table name '%s' is a reserved SQL keyword and cannot be used", tableName)
		}
	}

	return nil
}

// validateSQLFragment checks SQL fragments for injection attempts
func validateSQLFragment(fragment, fragmentType string) error {
	if strings.TrimSpace(fragment) == "" {
		return fmt.Errorf("%s fragment cannot be empty", fragmentType)
	}

	// Convert to lowercase for case-insensitive checking
	lowerFragment := strings.ToLower(fragment)

	// Check for prohibited SQL keywords that indicate complete statements
	prohibitedPatterns := []struct {
		keyword string
		message string
	}{
		{"select ", "SELECT"},
		{" select ", "SELECT"},
		{"^select", "SELECT"},
		{"insert ", "INSERT"},
		{" insert ", "INSERT"},
		{"^insert", "INSERT"},
		{"update ", "UPDATE"},
		{" update ", "UPDATE"},
		{"^update", "UPDATE"},
		{"delete ", "DELETE"},
		{" delete ", "DELETE"},
		{"^delete", "DELETE"},
		{"drop ", "DROP"},
		{" drop ", "DROP"},
		{"create ", "CREATE"},
		{" create ", "CREATE"},
		{"alter ", "ALTER"},
		{" alter ", "ALTER"},
		{"truncate ", "TRUNCATE"},
		{" truncate ", "TRUNCATE"},
	}

	for _, pattern := range prohibitedPatterns {
		if strings.Contains(lowerFragment, pattern.keyword) || strings.HasPrefix(lowerFragment, strings.TrimPrefix(pattern.keyword, "^")) {
			return fmt.Errorf(`invalid '%s' parameter: detected '%s' keyword.
Provide only the %s clause content, not a complete SQL statement.

✗ Bad example:  "%s name, email FROM users WHERE id = 5"
✓ Good example: "name, email WHERE id = 5"

Remove SQL keywords and provide only the clause fragment`, fragmentType, pattern.message, fragmentType, pattern.message)
		}
	}

	// Check for semicolons (statement terminators)
	if strings.Contains(fragment, ";") {
		return fmt.Errorf(`invalid '%s' parameter: detected semicolon (;).
Semicolons are used to terminate SQL statements and are not allowed in fragments.
Provide only a single clause fragment without semicolons`, fragmentType)
	}

	// Check for SQL comments
	if strings.Contains(fragment, "--") || strings.Contains(fragment, "/*") || strings.Contains(fragment, "*/") {
		return fmt.Errorf(`invalid '%s' parameter: detected SQL comment syntax (-- or /* */).
SQL comments are not allowed as they can be used for injection attacks.
Provide clean clause fragments without comments`, fragmentType)
	}

	// Check for UNION attacks
	if strings.Contains(lowerFragment, "union") {
		return fmt.Errorf(`invalid '%s' parameter: detected 'UNION' keyword.
UNION is not allowed as it can be used for SQL injection attacks.
Structure your query properly without UNION statements`, fragmentType)
	}

	// Fragment-specific validation
	switch fragmentType {
	case "select":
		return validateSelectFragment(fragment)
	case "update":
		return validateUpdateFragment(fragment)
	case "add":
		return validateInsertFragment(fragment)
	}

	return nil
}

// validateSelectFragment validates SELECT clause fragments
func validateSelectFragment(fragment string) error {
	lowerFragment := strings.ToLower(fragment)

	// Check for FROM keyword (indicates complete statement)
	if strings.Contains(lowerFragment, " from ") || strings.HasSuffix(lowerFragment, "from") {
		return fmt.Errorf(`invalid 'select' parameter: detected 'FROM' keyword.
Do not include the FROM clause - the table name is specified in the 'table' parameter.

✗ Bad:  "name, email FROM users WHERE age > 25"
✓ Good: "name, email WHERE age > 25"`)
	}

	// Check for subqueries
	if strings.Contains(fragment, "(") && strings.Contains(lowerFragment, "select") {
		return fmt.Errorf(`invalid 'select' parameter: subqueries are not supported.
Provide a simple column list and WHERE conditions without nested queries`)
	}

	return nil
}

// validateUpdateFragment validates UPDATE clause fragments
func validateUpdateFragment(fragment string) error {
	lowerFragment := strings.ToLower(fragment)

	// Check for SET keyword (indicates complete statement)
	if strings.HasPrefix(lowerFragment, "set ") {
		return fmt.Errorf(`invalid 'update' parameter: detected 'SET' keyword.
Provide only the column assignments without the SET keyword.

✗ Bad:  "SET status = 'active', updated_at = NOW()"
✓ Good: "status = 'active', updated_at = NOW()"`)
	}

	// Check that it contains assignment operators
	if !strings.Contains(fragment, "=") {
		return fmt.Errorf(`invalid 'update' parameter: no assignment operator found.
UPDATE requires column assignments in the format: "column = value"

Example: "status = 'active', updated_at = NOW()"`)
	}

	return nil
}

// validateInsertFragment validates INSERT clause fragments
func validateInsertFragment(fragment string) error {
	lowerFragment := strings.ToLower(fragment)

	// Check for INTO keyword (indicates complete statement)
	if strings.Contains(lowerFragment, " into ") || strings.HasPrefix(lowerFragment, "into ") {
		return fmt.Errorf(`invalid 'add' parameter: detected 'INTO' keyword.
Provide only the column names and VALUES clause without INSERT INTO.

✗ Bad:  "INTO users (name, email) VALUES ('John', 'john@example.com')"
✓ Good: "name, email VALUES ('John', 'john@example.com')"`)
	}

	// Check that it contains VALUES keyword (required for INSERT)
	if !strings.Contains(lowerFragment, "values") {
		return fmt.Errorf(`invalid 'add' parameter: missing 'VALUES' keyword.
INSERT requires the VALUES keyword to specify the data to insert.

Example: "name, email VALUES ('John Doe', 'john@example.com')"`)
	}

	return nil
}

// validateLimit checks if the limit is within acceptable bounds
func validateLimit(limit int) error {
	const maxLimit = 1000

	if limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}

	if limit > maxLimit {
		return fmt.Errorf(`limit exceeds maximum allowed value.
Requested: %d
Maximum: %d

Use a limit of %d or less, and use 'offset' for pagination if you need more results`, limit, maxLimit, maxLimit)
	}

	return nil
}

// validateOffset checks if the offset is valid
func validateOffset(offset int) error {
	if offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}

	return nil
}

// validatePostgresURL performs basic validation on the PostgreSQL connection URL
//
//nolint:unused // Reserved for future use
func validatePostgresURL(url string) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("postgres_url cannot be empty")
	}

	// Check for postgresql:// or postgres:// prefix
	if !strings.HasPrefix(url, "postgresql://") && !strings.HasPrefix(url, "postgres://") {
		return fmt.Errorf(`invalid postgres_url format.
Expected format: postgresql://user:password@host:port/database
Example: postgresql://myuser:mypass@localhost:5432/mydb`)
	}

	return nil
}

// validateSchema checks if the schema is in the allowed schemas list
//
//nolint:unused // Reserved for future use
func validateSchema(schema string, allowedSchemas []string) error {
	// Check for empty schema name
	if strings.TrimSpace(schema) == "" {
		return fmt.Errorf("schema name cannot be empty")
	}

	// Check for SQL injection attempts in schema name
	if err := validateSchemaName(schema); err != nil {
		return err
	}

	// Check if schema is in whitelist
	found := false
	for _, allowed := range allowedSchemas {
		if schema == allowed {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf(`schema '%s' is not in the allowed schemas list.
Available schemas: %v
Use one of the allowed schemas from the whitelist`, schema, allowedSchemas)
	}

	return nil
}

// validateSchemaName checks schema name for injection attempts
//
//nolint:unused // Used internally by executeGetSchema, executeGetTables, validateSchema
func validateSchemaName(schemaName string) error {
	// Check for special characters that could indicate injection
	// Allow only alphanumeric characters and underscores
	validSchemaName := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !validSchemaName.MatchString(schemaName) {
		return fmt.Errorf("invalid schema name: '%s'. Schema names can only contain letters, numbers, and underscores", schemaName)
	}

	return nil
}
