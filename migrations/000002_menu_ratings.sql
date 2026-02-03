ALTER TABLE feedbacks ALTER COLUMN rating DROP NOT NULL;

CREATE TABLE menu_ratings (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	menu_item_id uuid NOT NULL REFERENCES menu_items(id) ON DELETE RESTRICT,
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
	canteen_id uuid NOT NULL REFERENCES canteens(id) ON DELETE CASCADE,
	rating integer NOT NULL CHECK (rating BETWEEN 1 AND 5),
	is_removed boolean NOT NULL DEFAULT false,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE menu_ratings ADD CONSTRAINT menu_ratings_order_menu_unique UNIQUE (order_id, menu_item_id);

CREATE INDEX idx_menu_ratings_canteen_menu ON menu_ratings(canteen_id, menu_item_id);
CREATE INDEX idx_menu_ratings_menu_item_id ON menu_ratings(menu_item_id);
CREATE INDEX idx_menu_ratings_order_id ON menu_ratings(order_id);
CREATE INDEX idx_menu_ratings_user_id ON menu_ratings(user_id);
CREATE INDEX idx_menu_ratings_canteen_removed ON menu_ratings(canteen_id, is_removed);
