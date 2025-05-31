-- DROP TABLES IF THEY EXIST
DROP TABLE IF EXISTS trades;
DROP TABLE IF EXISTS orders;

-- ==============================
-- ORDERS TABLE
-- ==============================
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    side VARCHAR(10) CHECK (side IN ('buy', 'sell')) NOT NULL,
    type VARCHAR(6) CHECK (type IN ('limit', 'market')) NOT NULL,
    price NUMERIC(12, 2),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    remaining_quantity INTEGER NOT NULL CHECK (remaining_quantity >= 0),
    status VARCHAR(10) CHECK (status IN ('open', 'partial', 'filled', 'canceled')) NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- INDEX for matching efficiency
CREATE INDEX idx_orders_symbol_side_price_time ON orders (symbol, side, price, created_at);

-- ==============================
-- TRADES TABLE
-- ==============================
CREATE TABLE trades (
    id BIGSERIAL PRIMARY KEY,
    buy_order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    sell_order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    price NUMERIC(12, 2) NOT NULL CHECK (price > 0),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- INDEX for symbol lookup via JOIN
CREATE INDEX idx_trades_symbol_lookup ON trades (created_at);
