-- +migrate Up
ALTER TABLE deliveries ADD CONSTRAINT deliveries_order_uid_key UNIQUE(order_uid);
ALTER TABLE payments ADD CONSTRAINT payments_order_uid_key UNIQUE(order_uid);
ALTER TABLE items ADD CONSTRAINT items_chrt_id_key UNIQUE(chrt_id);