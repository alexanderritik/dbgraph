-- Create Tables
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE, -- Added UNIQUE for safety
    email VARCHAR(100) NOT NULL,
    address VARCHAR(100) NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    category VARCHAR(50)
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'pending',
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    unit_price DECIMAL(10, 2),
    CONSTRAINT fk_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    CONSTRAINT fk_product FOREIGN KEY (product_id) REFERENCES products(id)
);

-- 1. Self-Referencing Table (Hierarchy / Recursion)
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    manager_id INT,
    department VARCHAR(50),
    CONSTRAINT fk_manager FOREIGN KEY (manager_id) REFERENCES employees(id)
);

-- 2. Deep Dependency Chain (A -> B -> C -> D -> E)
-- Testing longest path algorithm
CREATE TABLE chain_a (id SERIAL PRIMARY KEY, data TEXT);
CREATE TABLE chain_b (id SERIAL PRIMARY KEY, a_id INT REFERENCES chain_a(id), data TEXT);
CREATE TABLE chain_c (id SERIAL PRIMARY KEY, b_id INT REFERENCES chain_b(id), data TEXT);
CREATE TABLE chain_d (id SERIAL PRIMARY KEY, c_id INT REFERENCES chain_c(id), data TEXT);
CREATE TABLE chain_e (id SERIAL PRIMARY KEY, d_id INT REFERENCES chain_d(id), data TEXT);

-- 3. Isolated Island (Cluster)
-- Testing connected components algorithm
CREATE TABLE island_alpha (id SERIAL PRIMARY KEY, note TEXT);
CREATE TABLE island_beta (id SERIAL PRIMARY KEY, alpha_id INT REFERENCES island_alpha(id), note TEXT);

-- 4. Another Tiny Island (Pair)
CREATE TABLE tiny_x (id SERIAL PRIMARY KEY);
CREATE TABLE tiny_y (id SERIAL PRIMARY KEY, x_id INT REFERENCES tiny_x(id));

-- 5. Triggers & Audit Log
-- Testing Trigger detection and counting
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    action VARCHAR(50),
    table_name VARCHAR(50),
    record_id INT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION log_order_changes() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        INSERT INTO audit_logs (action, table_name, record_id) VALUES ('INSERT', 'orders', NEW.id);
        RETURN NEW;
    ELSIF (TG_OP = 'UPDATE') THEN
        INSERT INTO audit_logs (action, table_name, record_id) VALUES ('UPDATE', 'orders', NEW.id);
        RETURN NEW;
    ELSIF (TG_OP = 'DELETE') THEN
        INSERT INTO audit_logs (action, table_name, record_id) VALUES ('DELETE', 'orders', OLD.id);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER order_audit_trigger
AFTER INSERT OR UPDATE OR DELETE ON orders
FOR EACH ROW EXECUTE FUNCTION log_order_changes();

-- 6. Partitioned Table (Logs)
-- Testing Node counting (should likely count parent + partitions depending on adapter logic)
CREATE TABLE app_logs (
    id SERIAL,
    log_date DATE NOT NULL,
    message TEXT
) PARTITION BY RANGE (log_date);

CREATE TABLE app_logs_2023 PARTITION OF app_logs
    FOR VALUES FROM ('2023-01-01') TO ('2024-01-01');

CREATE TABLE app_logs_2024 PARTITION OF app_logs
    FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');

-- 7. Missing Index Scenario
-- explicit_fk_missing_index has an FK to products but no index on product_id
-- Should trigger a warning if implemented
CREATE TABLE explicit_fk_missing_index (
    id SERIAL PRIMARY KEY,
    product_id INT NOT NULL,
    note TEXT,
    CONSTRAINT fk_missing_idx_product FOREIGN KEY (product_id) REFERENCES products(id)
);

-- 8. Composite Keys & Fancy Constraints
CREATE TABLE composite_keys (
    key_part_1 INT,
    key_part_2 INT,
    data TEXT,
    PRIMARY KEY (key_part_1, key_part_2)
);

CREATE TABLE composite_child (
    id SERIAL PRIMARY KEY,
    ref_1 INT,
    ref_2 INT,
    CONSTRAINT fk_composite FOREIGN KEY (ref_1, ref_2) REFERENCES composite_keys(key_part_1, key_part_2)
);

-- Create Views

-- Standard View
-- Safer GROUP BY: u.id, u.username
CREATE VIEW user_order_summary AS
SELECT 
    u.username,
    COUNT(o.id) as total_orders
FROM users u
JOIN orders o ON u.id = o.user_id
GROUP BY u.id, u.username;

-- Complex View with Aggregations
CREATE VIEW high_value_orders AS
SELECT 
    o.id as order_id,
    SUM(p.price * oi.quantity) as total_value
FROM orders o
JOIN order_items oi ON o.id = oi.order_id
JOIN products p ON oi.product_id = p.id
GROUP BY o.id
HAVING SUM(p.price * oi.quantity) > 100;

-- Nested View (View depending on View)
-- Testing View Dependency edges
CREATE VIEW vip_customers AS
SELECT username FROM user_order_summary WHERE total_orders > 5;

-- Materialized View (if supported by adapter, behaves like Table/View hybrid)
CREATE MATERIALIZED VIEW mview_product_stats AS
SELECT 
    p.category,
    COUNT(*) as count,
    AVG(p.price) as avg_price
FROM products p
GROUP BY p.category
WITH DATA;

-- Insert Mock Data
INSERT INTO users (username, email) VALUES
('alice', 'alice@example.com'),
('bob', 'bob@example.com'),
('charlie', 'charlie@example.com');

INSERT INTO products (name, price, category) VALUES
('Laptop', 1200.00, 'Electronics'),
('Mouse', 25.00, 'Electronics'),
('Keyboard', 50.00, 'Electronics'),
('Coffee Mug', 12.50, 'Home'),
('Desk Chair', 150.00, 'Furniture');

INSERT INTO orders (user_id, status) VALUES
(1, 'completed'), (2, 'pending'), (1, 'shipped'), (3, 'completed');

INSERT INTO order_items (order_id, product_id, quantity, unit_price) VALUES
(1, 1, 1, 1200.00),
(1, 2, 1, 25.00),
(2, 2, 1, 25.00),
(3, 3, 2, 50.00),
(4, 4, 4, 12.50);

INSERT INTO employees (name, manager_id, department) VALUES
('CEO', NULL, 'Executive'),
('CTO', 1, 'Engineering'),
('CFO', 1, 'Finance'),
('Dev Manager', 2, 'Engineering'),
('Senior Dev', 4, 'Engineering'),
('Junior Dev', 5, 'Engineering'); -- Deep hierarchy here too

-- Chain Data
INSERT INTO chain_a DEFAULT VALUES;
INSERT INTO chain_b (a_id) VALUES (1);
INSERT INTO chain_c (b_id) VALUES (1);
INSERT INTO chain_d (c_id) VALUES (1);
INSERT INTO chain_e (d_id) VALUES (1);

-- Island Data
INSERT INTO island_alpha (note) VALUES ('Lonely island');
INSERT INTO island_beta (alpha_id, note) VALUES (1, 'Companion');

-- Refresh Mat View
REFRESH MATERIALIZED VIEW mview_product_stats;
