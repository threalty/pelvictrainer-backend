-- Таблица программ тренировок
CREATE TABLE IF NOT EXISTS presets (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    difficulty VARCHAR(20) NOT NULL DEFAULT 'beginner',
    duration_minutes INTEGER NOT NULL DEFAULT 10,
    exercises_count INTEGER NOT NULL DEFAULT 5,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Таблица устройств для push-уведомлений
CREATE TABLE IF NOT EXISTS devices (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fcm_token TEXT NOT NULL,
    platform VARCHAR(20) NOT NULL DEFAULT 'android',
    app_version VARCHAR(20),
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, fcm_token)
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id);
CREATE INDEX IF NOT EXISTS idx_presets_active ON presets(is_active);

-- Seed: стартовые программы тренировок
INSERT INTO presets (name, description, difficulty, duration_minutes, exercises_count) VALUES
('Старт', 'Базовая программа для новичков. Знакомство с основными упражнениями.', 'beginner', 10, 5),
('Базовый уровень', 'Основная программа для поддержания тонуса. 3 раза в неделю.', 'beginner', 15, 7),
('Уверенный прогресс', 'Повышенная нагрузка для заметных результатов.', 'intermediate', 20, 9),
('Максимум результата', 'Интенсивная программа для опытных пользователей.', 'advanced', 30, 12)
ON CONFLICT DO NOTHING;
