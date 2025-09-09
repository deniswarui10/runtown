-- Create ticket_types table
CREATE TABLE ticket_types (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price INTEGER NOT NULL, -- Price in cents
    quantity INTEGER NOT NULL,
    sold INTEGER DEFAULT 0,
    sale_start TIMESTAMP NOT NULL,
    sale_end TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_price_positive CHECK (price >= 0),
    CONSTRAINT check_quantity_positive CHECK (quantity >= 0),
    CONSTRAINT check_sold_not_negative CHECK (sold >= 0),
    CONSTRAINT check_sold_not_exceed_quantity CHECK (sold <= quantity)
);

-- Performance indexes
CREATE INDEX idx_ticket_types_event ON ticket_types(event_id);
CREATE INDEX idx_ticket_types_sale_period ON ticket_types(sale_start, sale_end);