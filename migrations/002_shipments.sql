CREATE TABLE IF NOT EXISTS shipments (
    id UUID PRIMARY KEY,
    route TEXT NOT NULL,
    price NUMERIC NOT NULL,
    status TEXT NOT NULL DEFAULT 'CREATED',
    customer_id UUID NOT NULL REFERENCES customers(id),
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_shipments_customer_id ON shipments(customer_id);
CREATE INDEX IF NOT EXISTS idx_shipments_status ON shipments(status);

