-- Create tickets table
CREATE TABLE tickets (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id),
    ticket_type_id INTEGER REFERENCES ticket_types(id),
    qr_code VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'used', 'refunded')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Performance indexes
CREATE INDEX idx_tickets_order ON tickets(order_id);
CREATE INDEX idx_tickets_ticket_type ON tickets(ticket_type_id);
CREATE INDEX idx_tickets_qr_code ON tickets(qr_code);
CREATE INDEX idx_tickets_status ON tickets(status);