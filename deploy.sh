#!/bin/bash
# ============================================
# Скрипт автодеплоя PelvicTrainer
# Вызывается из GitHub Actions при push в master
# ============================================

set -e

echo "🚀 [$(date)] Начало деплоя..."

cd /opt/pelvictrainer

# 1. Обновляем код
echo "📥 git pull..."
git pull origin master

# 2. Пересобираем и перезапускаем backend
echo "🐳 Пересборка backend..."
docker compose build api
docker compose up -d api

# 3. Пересобираем frontend (админку)
echo "⚛️ Пересборка админки..."
cd admin-frontend
npm install --silent
npm run build

# 4. Проверка что API жив
echo " Health check..."
sleep 3
curl -sf http://127.0.0.1:8081/health > /dev/null && echo "✅ API работает" || echo "⚠️ API не ответил"

echo "🎉 [$(date)] Деплой завершён!"
