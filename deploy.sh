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
echo " Health check..."
sleep 3
curl -sf http://127.0.0.1:8081/health > /dev/null && echo "✅ API работает" || echo "⚠️ API не ответил"

echo "🎉 [$(date)] Деплой завершён!"
