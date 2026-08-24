<writeFile>
<path>migrations/010_create_password_reset_tokens.sql</path>
<content>-- Таблица токенов для восстановления пароля
CREATE TABLE IF NOT EXISTS password_reset_tokens (
id SERIAL PRIMARY KEY,
user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
token VARCHAR(64) UNIQUE NOT NULL,
expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
used_at TIMESTAMP WITH TIME ZONE,
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индекс для быстрого поиска по токену
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_token ON password_reset_tokens(token);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);
</content>
</writeFile>