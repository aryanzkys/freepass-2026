CREATE TYPE payment_record_status AS ENUM ('PENDING','VERIFIED','FAILED');

ALTER TABLE payments
	ALTER COLUMN status TYPE payment_record_status
	USING status::payment_record_status;

ALTER TABLE payments
	ALTER COLUMN status SET DEFAULT 'VERIFIED';

CREATE UNIQUE INDEX idx_order_items_order_menu_unique ON order_items(order_id, menu_item_id);

CREATE INDEX idx_orders_canteen_created_at ON orders(canteen_id, created_at);
CREATE INDEX idx_orders_user_created_at ON orders(user_id, created_at);
