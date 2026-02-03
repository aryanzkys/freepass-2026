CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE user_role AS ENUM (
	'USER',
	'OWNER',
	'ADMIN'
);

CREATE TYPE payment_status AS ENUM (
	'UNPAID',
	'PAID'
);

CREATE TYPE order_status AS ENUM (
	'WAITING',
	'COOKING',
	'READY',
	'COMPLETED'
);

CREATE TABLE users (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	name text NOT NULL,
	email text NOT NULL UNIQUE,
	password_hash text NOT NULL,
	role user_role NOT NULL DEFAULT 'USER',
	phone text,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE canteens (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	name text NOT NULL,
	location text,
	owner_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE menu_items (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	canteen_id uuid NOT NULL REFERENCES canteens(id) ON DELETE CASCADE,
	name text NOT NULL,
	description text,
	price integer NOT NULL CHECK (price >= 0),
	stock integer NOT NULL CHECK (stock >= 0),
	is_available boolean NOT NULL DEFAULT true,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE orders (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
	canteen_id uuid NOT NULL REFERENCES canteens(id) ON DELETE RESTRICT,
	payment_status payment_status NOT NULL DEFAULT 'UNPAID',
	order_status order_status NOT NULL DEFAULT 'WAITING',
	total_amount integer NOT NULL CHECK (total_amount >= 0),
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	menu_item_id uuid NOT NULL REFERENCES menu_items(id) ON DELETE RESTRICT,
	qty integer NOT NULL CHECK (qty > 0),
	price_snapshot integer NOT NULL CHECK (price_snapshot >= 0),
	subtotal integer NOT NULL CHECK (subtotal >= 0),
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE payments (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	order_id uuid NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
	method text,
	amount integer NOT NULL CHECK (amount >= 0),
	status text NOT NULL DEFAULT 'VERIFIED',
	paid_at timestamptz NOT NULL DEFAULT now(),
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE feedbacks (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	order_id uuid NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
	canteen_id uuid NOT NULL REFERENCES canteens(id) ON DELETE CASCADE,
	rating integer CHECK (rating BETWEEN 1 AND 5),
	comment text,
	is_removed boolean NOT NULL DEFAULT false,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

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

CREATE INDEX idx_canteens_owner_id ON canteens(owner_id);
CREATE INDEX idx_menu_items_canteen_id ON menu_items(canteen_id);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_canteen_id ON orders(canteen_id);
CREATE INDEX idx_orders_payment_status ON orders(payment_status);
CREATE INDEX idx_orders_order_status ON orders(order_status);
CREATE INDEX idx_orders_created_at ON orders(created_at);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_menu_item_id ON order_items(menu_item_id);
CREATE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_feedbacks_canteen_id ON feedbacks(canteen_id);
CREATE INDEX idx_feedbacks_user_id ON feedbacks(user_id);
CREATE INDEX idx_menu_ratings_canteen_menu ON menu_ratings(canteen_id, menu_item_id);
CREATE INDEX idx_menu_ratings_menu_item_id ON menu_ratings(menu_item_id);
CREATE INDEX idx_menu_ratings_order_id ON menu_ratings(order_id);
CREATE INDEX idx_menu_ratings_user_id ON menu_ratings(user_id);
CREATE INDEX idx_menu_ratings_canteen_removed ON menu_ratings(canteen_id, is_removed);
