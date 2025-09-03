CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    order_uid TEXT NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    chrt_id BIGINT,
    track_number TEXT,
    price INTEGER,
    rid TEXT,
    name TEXT,
    sale INTEGER,
    size TEXT,
    total_price INTEGER,
    nm_id BIGINT,
    brand TEXT,
    status INTEGER
);