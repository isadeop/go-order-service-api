CREATE TABLE IF NOT EXISTS stock_reservations (
	saga_id UUID NOT NULL,
	product_id UUID NOT NULL,
	quantity INTEGER NOT NULL,
	status VARCHAR(20) NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (saga_id, product_id),
	CONSTRAINT fk_stock_reservations_products
		FOREIGN KEY(product_id)
		REFERENCES products(id)
);
