-- Таблица рассылок
CREATE TABLE IF NOT EXISTS broadcasts (
    id SERIAL PRIMARY KEY,
    subject VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    audience VARCHAR(50) NOT NULL DEFAULT 'all', -- 'all', 'premium', 'free', 'inactive'
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'sending', 'sent', 'failed'
    sent_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    sent_at TIMESTAMP WITH TIME ZONE
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_broadcasts_status ON broadcasts(status);
CREATE INDEX IF NOT EXISTS idx_broadcasts_created_at ON broadcasts(created_at);