CREATE TABLE deliveries (
    id SERIAL PRIMARY KEY,
    order_uid TEXT NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    name TEXT,
    phone TEXT,
    zip TEXT,
    city TEXT,
    address TEXT,
    region TEXT,
    email TEXT
);