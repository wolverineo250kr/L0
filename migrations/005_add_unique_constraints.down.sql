-- +migrate Down
ALTER TABLE deliveries DROP CONSTRAINT IF EXISTS deliveries_order_uid_key;
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_order_uid_key;
ALTER TABLE items DROP CONSTRAINT IF EXISTS items_chrt_id_key;