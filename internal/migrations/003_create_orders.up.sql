CREATE TABLE IF NOT EXISTS orders (
	id UUID PRIMARY KEY DEFAULT uuidv7(),
    client_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    total NUMERIC(10,2) NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_orders_clients
        FOREIGN KEY(client_id)
        REFERENCES clients(id)
);