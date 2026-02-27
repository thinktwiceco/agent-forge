package postgres

import (
	"fmt"

	"github.com/thinktwiceco/agent-forge/src/core"
	"github.com/thinktwiceco/agent-forge/src/llms"
)

// Postgres represents a postgres database tool with configured connection and access control.
type Postgres struct {
	postgresURL    string
	mode           string
	allowedTables  []string
	allowedSchemas []string
}

// NewPostgresTool creates a postgres database tool that provides secure database operations.
// The tool accepts complete SQL queries and validates them against mode and table restrictions.
//
// Parameters:
//   - postgresURL: PostgreSQL connection URL (format: postgresql://user:pass@host:port/db)
//   - mode: Access mode - "read" (SELECT only) or "write" (SELECT, INSERT, UPDATE, DELETE)
//   - allowedTables: Whitelist of table names the agent can access. Use "*" to grant access to all tables.
//   - allowedSchemas: Whitelist of schema names; when set, tables must be in these schemas and search_path is set for convenience
func NewPostgresTool(postgresURL, mode string, allowedTables, allowedSchemas []string) llms.Tool {
	pg := &Postgres{
		postgresURL:    postgresURL,
		mode:           mode,
		allowedTables:  allowedTables,
		allowedSchemas: allowedSchemas,
	}

	desc := fmt.Sprintf("Execute SQL queries on PostgreSQL database. Allowed tables: %v. Mode: %s", allowedTables, mode)
	if len(allowedSchemas) > 0 {
		desc += fmt.Sprintf(". Allowed schemas: %v", allowedSchemas)
	}

	schemaGuidance := ""
	if len(allowedSchemas) > 0 {
		schemaGuidance = fmt.Sprintf(`  * Allowed Schemas: %v (tables must be in these schemas; use schema."TableName" for mixed-case names)
  * IMPORTANT for mixed-case table names (e.g. Org, User): Use double-quoted identifiers: schema."Org" or "Org"
  * Unquoted identifiers are lowercased by PostgreSQL - "Org" and org are different

`, allowedSchemas)
	}

	return &core.Tool{
		Name:        "postgres",
		Description: desc,
		AdvanceDesc: fmt.Sprintf(`Advanced Details:
- Configuration:
  * Mode: %s (READ allows SELECT only; WRITE allows SELECT, INSERT, UPDATE, DELETE)
  * Allowed Tables: %v (queries can only access these tables; use "*" to allow all tables)
%s- Parameters:
  * query (string, required): Complete SQL query to execute

- Behavior:
  * Provide a complete, valid SQL query
  * The tool validates the query before execution:
    - Checks if the statement type is allowed for the current mode
    - Extracts table names and verifies they are in the allowed list
    - Blocks dangerous operations (DROP, TRUNCATE, ALTER, CREATE, etc.)
  * All queries are executed directly without modification
  * Results are returned in JSON format with row count and data

- Security:
  * Query validation prevents dangerous operations
  * Table whitelist restricts access to authorized tables only (use "*" to allow all tables)
  * Mode-based restrictions (READ = SELECT only, WRITE = all data operations)
  * Dangerous operations (DROP, TRUNCATE, ALTER, etc.) are blocked

- Usage Examples:
  Select users (available in READ and WRITE modes):
    query: "SELECT name, email, age FROM users WHERE age > 25 LIMIT 50"

  Insert user (WRITE mode only):
    query: "INSERT INTO users (name, email, age) VALUES ('John Doe', 'john@example.com', 30)"

  Update user (WRITE mode only):
    query: "UPDATE users SET last_login = NOW() WHERE user_id = 123"

  Delete order (WRITE mode only):
    query: "DELETE FROM orders WHERE order_status = 'cancelled' AND order_date < '2024-01-01'"
  
  Join query (available in READ and WRITE modes):
    query: "SELECT u.name, o.order_id, o.total_amount FROM users u JOIN orders o ON u.user_id = o.user_id WHERE u.status = 'active'"

  With schema and mixed-case tables (when allowedSchemas is set):
    query: "SELECT * FROM private.\"Org\" LIMIT 10"
    query: "SELECT * FROM private.\"User\" WHERE \"orgId\" = 'xxx'"`, mode, allowedTables, schemaGuidance),
		TroubleshootingInfo: fmt.Sprintf(`Troubleshooting:
- "access denied: table(s) not in allowed list": Your query references tables that are not in the allowed list: %v. Only use these tables.
- "operation not allowed in READ mode": You attempted a write operation (INSERT, UPDATE, DELETE) but the tool is in READ mode (%s). Only SELECT queries are allowed in READ mode.
- "dangerous operation detected": Operations like DROP, TRUNCATE, ALTER, CREATE are not allowed for security reasons.
- "unsupported SQL statement": Only SELECT, INSERT, UPDATE, and DELETE statements are supported.
- "could not extract table names": The query structure is too complex or malformed. Use explicit table names (schema.\"TableName\" for mixed-case).
- "relation X does not exist": Tables may be in a different schema. Use schema-qualified names: schema.\"TableName\". For mixed-case tables, use double quotes.
- Connection errors: Check that the database server is running, credentials are correct, and network connectivity is available.
- Query execution errors: Verify your SQL syntax is correct and column/table names exist in the database.`, allowedTables, mode),
		Parameters: []core.Parameter{
			{
				Name:        "query",
				Type:        "string",
				Description: "Complete SQL query to execute (e.g., 'SELECT * FROM users WHERE age > 25')",
				Required:    true,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			// Extract query
			queryVal, ok := args["query"]
			if !ok {
				return core.NewErrorResponse("missing required parameter: query")
			}

			query, ok := queryVal.(string)
			if !ok {
				return core.NewErrorResponse("query parameter must be a string")
			}

			// Validate the query
			_, err := ValidateQuery(query, pg.mode, pg.allowedTables, pg.allowedSchemas)
			if err != nil {
				return core.NewErrorResponse(fmt.Sprintf("query validation failed: %v", err))
			}

			// Execute the query
			result, err := pg.executeQuery(query)
			if err != nil {
				return core.NewErrorResponse(err.Error())
			}

			return core.NewSuccessResponse(result)
		},
	}
}
