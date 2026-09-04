-- Миграция: Таблица истории входов для отслеживания новых устройств
-- Задача: Отправлять email при входе с нового устройства

CREATE TABLE IF NOT EXISTS login_history (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address VARCHAR(45),
    user_agent TEXT,
    user_agent_hash VARCHAR(64) NOT NULL,
    device_info VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы для быстрых запросов
CREATE INDEX IF NOT EXISTS idx_login_history_user_id ON login_history(user_id);
CREATE INDEX IF NOT EXISTS idx_login_history_user_agent_hash ON login_history(user_agent_hash);
CREATE INDEX IF NOT EXISTS idx_login_history_created_at ON login_history(created_at DESC);

-- Уникальный индекс: один хеш на пользователя (для определения "нового устройства")
CREATE UNIQUE INDEX IF NOT EXISTS idx_login_history_unique_device 
    ON login_history(user_id, user_agent_hash);

COMMENT ON TABLE login_history IS 'История входов пользователей для отслеживания новых устройств';
COMMENT ON COLUMN login_history.user_agent_hash IS 'SHA256 хеш от User-Agent для идентификации устройства';
COMMENT ON COLUMN login_history.device_info IS 'Упрощённое описание устройства (например: "Chrome on Android")';