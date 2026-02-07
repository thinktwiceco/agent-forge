-- Create users table
CREATE TABLE users (
    user_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    age INTEGER,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create products table
CREATE TABLE products (
    product_id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    stock_quantity INTEGER DEFAULT 0,
    category VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create orders table
CREATE TABLE orders (
    order_id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(user_id),
    product_id INTEGER REFERENCES products(product_id),
    quantity INTEGER NOT NULL,
    total_amount DECIMAL(10, 2) NOT NULL,
    order_status VARCHAR(20) DEFAULT 'pending',
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample users
INSERT INTO users (name, email, age, status) VALUES
    ('Alice Johnson', 'alice@example.com', 28, 'active'),
    ('Bob Smith', 'bob@example.com', 35, 'active'),
    ('Charlie Brown', 'charlie@example.com', 42, 'inactive'),
    ('Diana Prince', 'diana@example.com', 31, 'active'),
    ('Eve Davis', 'eve@example.com', 26, 'active'),
    ('Frank Miller', 'frank@example.com', 55, 'active'),
    ('Grace Lee', 'grace@example.com', 29, 'inactive'),
    ('Henry Wilson', 'henry@example.com', 38, 'active'),
    ('Ivy Chen', 'ivy@example.com', 33, 'active'),
    ('Jack Taylor', 'jack@example.com', 41, 'active');

-- Insert sample products
INSERT INTO products (name, description, price, stock_quantity, category) VALUES
    ('Laptop Pro 15', 'High-performance laptop with 16GB RAM', 1299.99, 25, 'electronics'),
    ('Wireless Mouse', 'Ergonomic wireless mouse with 3 buttons', 29.99, 150, 'electronics'),
    ('Office Chair', 'Comfortable ergonomic office chair', 249.99, 40, 'furniture'),
    ('Standing Desk', 'Adjustable height standing desk', 599.99, 15, 'furniture'),
    ('USB-C Hub', '7-in-1 USB-C hub with multiple ports', 49.99, 200, 'electronics'),
    ('Coffee Maker', 'Programmable 12-cup coffee maker', 89.99, 60, 'appliances'),
    ('Desk Lamp', 'LED desk lamp with adjustable brightness', 39.99, 75, 'furniture'),
    ('Keyboard Mechanical', 'RGB mechanical keyboard with blue switches', 129.99, 50, 'electronics'),
    ('Monitor 27"', '4K UHD 27-inch monitor', 449.99, 30, 'electronics'),
    ('Notebook Set', 'Pack of 5 premium notebooks', 19.99, 300, 'stationery');

-- Insert sample orders
INSERT INTO orders (user_id, product_id, quantity, total_amount, order_status) VALUES
    (1, 1, 1, 1299.99, 'completed'),
    (1, 2, 2, 59.98, 'completed'),
    (2, 3, 1, 249.99, 'shipped'),
    (3, 5, 3, 149.97, 'pending'),
    (4, 9, 1, 449.99, 'completed'),
    (5, 6, 1, 89.99, 'completed'),
    (6, 4, 1, 599.99, 'shipped'),
    (7, 10, 5, 99.95, 'cancelled'),
    (8, 8, 1, 129.99, 'pending'),
    (9, 7, 2, 79.98, 'completed'),
    (10, 1, 1, 1299.99, 'pending'),
    (1, 5, 1, 49.99, 'completed'),
    (2, 2, 1, 29.99, 'shipped'),
    (4, 6, 2, 179.98, 'completed'),
    (5, 3, 1, 249.99, 'pending');

-- Create indexes for better query performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(order_status);

-- Display summary
SELECT 'Database initialized successfully!' as message;
SELECT COUNT(*) as total_users FROM users;
SELECT COUNT(*) as total_products FROM products;
SELECT COUNT(*) as total_orders FROM orders;
