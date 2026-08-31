ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';
UPDATE users SET role = 'admin' WHERE email = 'admin@pelvictrainer.ru';
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
COMMENT ON COLUMN users.role IS 'Роль пользователя: user (обычный), admin (администратор)';
