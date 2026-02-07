# PostgreSQL Tool - Quick Start Guide

## Overview

The postgres tool allows agents to execute SQL queries on PostgreSQL databases with built-in security and access control.

## Configuration

```go
import "github.com/thinktwiceco/agent-forge/src/tools/postgres"

tool := postgres.NewPostgresTool(
    "postgresql://user:pass@host:port/db?sslmode=disable",  // Connection URL
    "write",                                                  // Mode: "read" or "write"
    []string{"users", "products", "orders"},                 // Allowed tables
    []string{"public"},                                      // Allowed schemas (optional)
)
```

### YAML Configuration

```yaml
agent:
  tools:
    - name: postgres
      postgresURL: "postgresql://testuser:testpass@localhost:5432/testdb?sslmode=disable"
      mode: "write"
      allowedTables:
        - "users"
        - "products"
        - "orders"
      allowedSchemas:
        - "public"
```

## Modes

### READ Mode
- **Allowed**: SELECT queries only
- **Use for**: Data retrieval, reporting, analytics

### WRITE Mode
- **Allowed**: SELECT, INSERT, UPDATE, DELETE
- **Use for**: Full database operations

## Usage Examples

### SELECT Queries

```sql
-- Simple select
SELECT * FROM users WHERE age > 25

-- Select specific columns
SELECT name, email, age FROM users WHERE status = 'active'

-- With LIMIT
SELECT * FROM products ORDER BY price DESC LIMIT 10

-- JOIN queries
SELECT u.name, o.order_id, o.total_amount 
FROM users u 
JOIN orders o ON u.user_id = o.user_id 
WHERE o.order_status = 'completed'

-- Aggregate functions
SELECT category, COUNT(*), AVG(price) 
FROM products 
GROUP BY category
```

### INSERT Queries (WRITE mode only)

```sql
-- Single insert
INSERT INTO users (name, email, age) 
VALUES ('John Doe', 'john@example.com', 30)

-- Multiple inserts
INSERT INTO products (name, price, category) 
VALUES 
  ('Widget A', 19.99, 'electronics'),
  ('Widget B', 29.99, 'electronics')
```

### UPDATE Queries (WRITE mode only)

```sql
-- Update with WHERE clause
UPDATE users SET status = 'inactive' WHERE user_id = 5

-- Update multiple columns
UPDATE products 
SET price = 24.99, stock_quantity = 100 
WHERE product_id = 10

-- Update with calculation
UPDATE products 
SET stock_quantity = stock_quantity - 1 
WHERE product_id = 3
```

### DELETE Queries (WRITE mode only)

```sql
-- Delete with condition
DELETE FROM orders WHERE order_status = 'cancelled'

-- Delete old records
DELETE FROM logs WHERE created_at < '2024-01-01'
```

## Security Features

### Automatic Validation

The tool automatically validates every query:

1. **Statement Type Check**
   - Allows: SELECT, INSERT, UPDATE, DELETE
   - Blocks: DROP, TRUNCATE, ALTER, CREATE, GRANT, etc.

2. **Mode Enforcement**
   - READ mode: only SELECT
   - WRITE mode: all allowed operations

3. **Table Whitelisting**
   - Extracts table names from queries
   - Verifies all tables are in allowedTables list
   - Works with JOINs and subqueries

### What's Blocked

```sql
-- ✗ Dangerous operations
DROP TABLE users
TRUNCATE TABLE orders
ALTER TABLE products ADD COLUMN x VARCHAR(50)
CREATE TABLE new_table (id INT)

-- ✗ Unauthorized tables
SELECT * FROM admin_secrets  -- if not in allowedTables

-- ✗ Write operations in READ mode
INSERT INTO users (name) VALUES ('John')
UPDATE users SET status = 'active'
DELETE FROM orders WHERE id = 1
```

## Error Messages

### "access denied: table(s) not in allowed list"
- Your query references tables not in the allowedTables whitelist
- Solution: Only use allowed tables or ask admin to add tables to whitelist

### "operation X not allowed in READ mode"
- Attempted write operation (INSERT/UPDATE/DELETE) in READ mode
- Solution: Use SELECT only or switch to WRITE mode

### "dangerous operation detected"
- Attempted dangerous operation (DROP, TRUNCATE, etc.)
- Solution: These operations are never allowed for security

### "could not extract table names from query"
- Query structure is too complex or malformed
- Solution: Simplify query or verify SQL syntax

## Test Database

A Docker-based test environment is available:

```bash
cd src/tools/postgres/testharness
docker-compose up -d
```

Connection details:
- **URL**: `postgresql://testuser:testpass@localhost:5432/testdb?sslmode=disable`
- **Tables**: users, products, orders (with sample data)

## Testing Your Queries

```bash
# Run unit tests
go test ./src/tools/postgres -v

# Connect to test database
docker exec -it postgres-testharness psql -U testuser -d testdb
```

## Best Practices

1. **Use specific columns** instead of `SELECT *`
2. **Always use LIMIT** for queries that might return many rows
3. **Use WHERE clauses** to filter data efficiently
4. **Test queries** on test database before production
5. **Keep queries simple** - complex queries are harder to validate
