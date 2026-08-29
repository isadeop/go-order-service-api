-- product_id referencia products.id no banco do stock-service (stock_db) —
-- não existe mais FOREIGN KEY para products aqui, porque bancos diferentes
-- não podem ter FK entre si no Postgres. A existência do produto é
-- validada pela aplicação (via chamada ao stock-service), não pelo banco.
CREATE TABLE order_items (
	id UUID PRIMARY KEY DEFAULT uuidv7(),
    order_id UUID NOT NULL,
    product_id UUID NOT NULL,
    quantity INTEGER NOT NULL CHECK(quantity > 0),
    price NUMERIC(10,2) NOT NULL,

    CONSTRAINT fk_order
        FOREIGN KEY(order_id)
        REFERENCES orders(id)
);
