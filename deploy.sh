#!/bin/bash
# ============================================
# Скрипт автодеплоя PelvicTrainer
# Вызывается из GitHub Actions при push в master
# ============================================

set -e

echo "🚀 [$(date)] Начало деплоя..."

export GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=no -i ~/.ssh/deploy_key"

cd /opt/pelvictrainer

# 0. Убиваем ЛОКАЛЬНЫЕ dev-процессы Go (они могут держать порт 8081)
#    ВАЖНО: не трогаем Docker контейнер (его процесс = ./main)
echo "🔍 Убиваем локальные dev-процессы (go run)..."
sudo pkill -9 -f "go run" 2>/dev/null || true
sudo pkill -9 -f "/tmp/go-build" 2>/dev/null || true
sudo pkill -9 -f "exe/main" 2>/dev/null || true
sleep 1

# 1. Проверяем порт 8081: кто держит?
HOLDER=$(sudo ss -tlnp 2>/dev/null | grep ":8081" || true)
if [ -n "$HOLDER" ]; then
    if echo "$HOLDER" | grep -q "docker-proxy"; then
        echo "✅ Порт 8081 держит Docker (docker-proxy) — это нормально"
    else
        echo "⚠️ Порт 8081 занят НЕ-Docker процессом:"
        echo "$HOLDER"
        PID=$(echo "$HOLDER" | grep -o 'pid=[0-9]*' | head -1 | cut -d= -f2)
        if [ -n "$PID" ]; then
            echo "⚠️ Убиваем PID $PID..."
            sudo kill -9 "$PID" || true
            sleep 1
        else
            echo "❌ Не удалось определить PID, прерываем"
            exit 1
        fi
    fi
else
    echo "✅ Порт 8081 свободен"
fi

# 2. Обновляем код
echo "📥 git pull..."
git pull origin master

# 3. Пересобираем и перезапускаем backend
echo "🐳 Пересборка backend..."
docker compose build api
docker compose up -d api

# 4. Пересобираем frontend
echo "⚛️ Пересборка админки..."
cd admin-frontend
npm install --silent
npm run build

# 5. Health check
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
