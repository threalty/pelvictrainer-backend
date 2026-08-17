#!/bin/bash
# ============================================
# Скрипт автодеплоя PelvicTrainer
# Вызывается из GitHub Actions при push в master
# ============================================

set -e

echo "🚀 [$(date)] Начало деплоя..."

# Явно указываем SSH config для git операций
export GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=no -i ~/.ssh/deploy_key"

cd /opt/pelvictrainer

# 0. Освобождаем порт 8081 если занят локальным процессом
echo "🔍 Проверка порта 8081..."
if ss -tlnp 2>/dev/null | grep -q ":8081"; then
    echo "⚠️ Порт 8081 занят, освобождаем..."
    # Способ 1: fuser
    sudo fuser -k 8081/tcp 2>/dev/null || true
    # Способ 2: pkill по имени процесса (для Go)
    sudo pkill -9 -f "/tmp/go-build.*/main" 2>/dev/null || true
    sudo pkill -9 -f "go run cmd/api" 2>/dev/null || true
    sleep 2
    
    # Проверяем что освободился
    if ss -tlnp 2>/dev/null | grep -q ":8081"; then
        echo "❌ Не удалось освободить порт 8081"
        ss -tlnp | grep ":8081"
        exit 1
    fi
    echo "✅ Порт 8081 освобождён"
else
    echo "✅ Порт 8081 свободен"
fi

# 1. Обновляем код
echo "📥 git pull..."
git pull origin master

# 2. Пересобираем и перезапускаем backend
echo "🐳 Пересборка backend..."
docker compose build api
docker compose up -d api

# 3. Пересобираем frontend
echo "⚛️ Пересборка админки..."
cd admin-frontend
npm install --silent
npm run build

# 4. Health check
echo "🔍 Health check..."
sleep 5
if curl -sf http://127.0.0.1:8081/health > /dev/null; then
    echo "✅ API работает"
else
    echo "⚠️ API не ответил, проверяем логи:"
    docker compose logs api --tail 20
    exit 1
fi

echo "🎉 [$(date)] Деплой завершён!"
