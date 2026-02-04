ALTER TYPE order_status ADD VALUE IF NOT EXISTS 'PAYMENT';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'AWAITING_VERIFICATION';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'REJECTED';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'REFUNDED';

DO $$ BEGIN
	CREATE TYPE payment_method AS ENUM ('CASH','CASHLESS_QRIS');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

ALTER TABLE canteens ADD COLUMN IF NOT EXISTS qris_static_url text;
ALTER TABLE canteens ADD COLUMN IF NOT EXISTS qris_static_updated_at timestamptz;

ALTER TABLE orders ADD COLUMN IF NOT EXISTS payment_method payment_method NOT NULL DEFAULT 'CASH';

CREATE TABLE IF NOT EXISTS payment_verifications (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE UNIQUE,
	canteen_id uuid NOT NULL REFERENCES canteens(id) ON DELETE CASCADE,
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
	status text NOT NULL,
	rejection_reason text,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payment_verifications_canteen_id ON payment_verifications(canteen_id);
CREATE INDEX IF NOT EXISTS idx_payment_verifications_user_id ON payment_verifications(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_verifications_status ON payment_verifications(status);

CREATE TABLE IF NOT EXISTS refunds (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	canteen_id uuid NOT NULL REFERENCES canteens(id) ON DELETE CASCADE,
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
	amount bigint NOT NULL,
	reason text NOT NULL,
	status text NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_refunds_order_id ON refunds(order_id);
CREATE INDEX IF NOT EXISTS idx_refunds_canteen_id ON refunds(canteen_id);
CREATE INDEX IF NOT EXISTS idx_refunds_user_id ON refunds(user_id);
CREATE INDEX IF NOT EXISTS idx_refunds_status ON refunds(status);
