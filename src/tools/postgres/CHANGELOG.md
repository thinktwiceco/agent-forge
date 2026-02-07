# PostgreSQL Tool - Simplified Version

## Changes Summary

The postgres tool has been simplified to accept complete SQL queries instead of SQL fragments.

### Before (Fragment-based approach)

**Parameters:**
- `operation`: select/update/insert/getTables/getSchema
- `table`: table name
- `select`: column list + WHERE clause fragment
- `update`: SET clause fragment
- `add`: INSERT fragment with VALUES
- `limit`, `offset`: pagination

**Example usage:**
```
operation: "select"
table: "users"
select: "name, email WHERE age > 25"
limit: 50
```

**Problems:**
- Complex API with many parameters
- Agent needs to understand fragment syntax
- Difficult to express complex queries (JOINs, subqueries)
- More validation rules needed

### After (Query-based approach)

**Parameters:**
- `query`: complete SQL query

**Example usage:**
```
query: "SELECT name, email FROM users WHERE age > 25 LIMIT 50"
```

**Benefits:**
- Simple API with single parameter
- Agent writes standard SQL
- Easy to express complex queries
- Validation focuses on security (table access, dangerous operations)

## Validation & Security

The tool validates queries in three steps:

1. **Statement Type Detection**
   - Extracts SQL statement type (SELECT, INSERT, UPDATE, DELETE)
   - Blocks dangerous operations (DROP, TRUNCATE, ALTER, CREATE, GRANT, etc.)

2. **Mode Validation**
   - READ mode: only SELECT allowed
   - WRITE mode: SELECT, INSERT, UPDATE, DELETE allowed

3. **Table Whitelisting**
   - Extracts table names from query using regex
   - Validates all referenced tables are in allowedTables list
   - Supports JOINs across multiple allowed tables

## Files

### New Files
- `queryValidator.go` - Query validation and table extraction
- `actExecute.go` - Simplified query execution
- `queryValidator_test.go` - Comprehensive validation tests
- `CHANGELOG.md` - This file

### Removed Files
- `validate.go` - Old fragment validation
- `queryBuilder.go` - Old parameterized query builder
- `actSelect.go`, `actUpdate.go`, `actInsert.go` - Old action handlers
- `actGetTables.go`, `actGetSchema.go` - Metadata operations

### Modified Files
- `tool.go` - Simplified tool definition
- `tool_test.go` - Updated to test new API
- `testharness/README.md` - Updated examples

## Migration Guide

If you have existing agents using the old API:

### Old Code
```yaml
tools:
  - type: postgres
    # ... config ...
```

Agent call:
```
postgres(
  operation: "select",
  table: "users",
  select: "name, email WHERE age > 25",
  limit: 50
)
```

### New Code
```yaml
tools:
  - type: postgres
    # ... same config ...
```

Agent call:
```
postgres(
  query: "SELECT name, email FROM users WHERE age > 25 LIMIT 50"
)
```

## Testing

Run tests:
```bash
go test ./src/tools/postgres -v
```

Start test database:
```bash
cd src/tools/postgres/testharness
docker-compose up -d
```
