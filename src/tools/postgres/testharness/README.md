# PostgreSQL Test Harness

Test environment for the postgres tool with a containerized PostgreSQL database.

## Database Configuration

- **Host**: localhost
- **Port**: 5432
- **Database**: testdb
- **User**: testuser
- **Password**: testpass
- **Connection URL**: `postgresql://testuser:testpass@localhost:5432/testdb?sslmode=disable`

## Database Schema

### Tables

1. **users** - User accounts
   - user_id (SERIAL PRIMARY KEY)
   - name (VARCHAR)
   - email (VARCHAR UNIQUE)
   - age (INTEGER)
   - status (VARCHAR)
   - created_at (TIMESTAMP)
   - updated_at (TIMESTAMP)

2. **products** - Product catalog
   - product_id (SERIAL PRIMARY KEY)
   - name (VARCHAR)
   - description (TEXT)
   - price (DECIMAL)
   - stock_quantity (INTEGER)
   - category (VARCHAR)
   - created_at (TIMESTAMP)

3. **orders** - Order records
   - order_id (SERIAL PRIMARY KEY)
   - user_id (INTEGER FK)
   - product_id (INTEGER FK)
   - quantity (INTEGER)
   - total_amount (DECIMAL)
   - order_status (VARCHAR)
   - order_date (TIMESTAMP)

### Sample Data

- 10 users
- 10 products (electronics, furniture, appliances, stationery)
- 15 orders (various statuses: completed, shipped, pending, cancelled)

## Usage

### Start the Database

```bash
cd src/tools/postgres/testharness
docker-compose up -d
```

### Stop the Database

```bash
docker-compose down
```

### Stop and Remove All Data

```bash
docker-compose down -v
```

### View Logs

```bash
docker-compose logs -f
```

### Connect with psql

```bash
docker exec -it postgres-testharness psql -U testuser -d testdb
```

## Testing Examples

### Read Mode Testing

Use with agent config:
```yaml
tools:
  - type: postgres
    postgres_url: "postgresql://testuser:testpass@localhost:5432/testdb?sslmode=disable"
    mode: "read"
    allowed_tables: ["users", "products", "orders"]
    allowed_schemas: ["public"]
```

Example queries in READ mode:
```sql
-- Select all active users
SELECT * FROM users WHERE status = 'active'

-- Get products by category
SELECT name, price, stock_quantity FROM products WHERE category = 'electronics'

-- Join users and orders
SELECT u.name, u.email, o.order_id, o.total_amount 
FROM users u 
JOIN orders o ON u.user_id = o.user_id 
WHERE o.order_status = 'completed'
```

### Write Mode Testing

Use with agent config:
```yaml
tools:
  - type: postgres
    postgres_url: "postgresql://testuser:testpass@localhost:5432/testdb?sslmode=disable"
    mode: "write"
    allowed_tables: ["users", "products", "orders"]
    allowed_schemas: ["public"]
```

Example queries in WRITE mode:
```sql
-- Insert a new user
INSERT INTO users (name, email, age, status) 
VALUES ('New User', 'newuser@example.com', 25, 'active')

-- Update user status
UPDATE users SET status = 'inactive' WHERE user_id = 5

-- Delete cancelled orders
DELETE FROM orders WHERE order_status = 'cancelled'

-- Update product stock
UPDATE products SET stock_quantity = stock_quantity - 1 WHERE product_id = 3
```

## Tool Usage

The simplified postgres tool accepts complete SQL queries:

```go
// Agent provides a full SQL query
query := "SELECT * FROM users WHERE age > 25 LIMIT 10"

// Tool validates:
// 1. Statement type (SELECT allowed in read mode)
// 2. Tables are in allowed list (users must be in allowedTables)
// 3. No dangerous operations (DROP, TRUNCATE, etc. blocked)
```

### Security Features

- **Mode-based restrictions**: READ mode only allows SELECT, WRITE mode allows SELECT/INSERT/UPDATE/DELETE
- **Table whitelisting**: Queries can only access tables in the allowed list
- **Dangerous operation blocking**: DROP, TRUNCATE, ALTER, CREATE are blocked
- **Table extraction**: Tool automatically extracts tables from queries to validate access
