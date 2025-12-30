CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY,
    idn TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_customers_idn ON customers(idn);

