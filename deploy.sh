#!/bin/bash
# ============================================
# Скрипт автодеплоя PelvicTrainer
# Вызывается из GitHub Actions при push в master
# ============================================

set -e

echo "🚀 [$(date)] Начало деплоя..."

export GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=no -i ~/.ssh/deploy_key"

cd /opt/pelvictrainer

# 0. Освобождаем порт 8081 если занят локальным процессом
echo "🔍 Проверка порта 8081..."
if ss -tlnp 2>/dev/null | grep -q ":8081"; then
    echo "⚠️ Порт 8081 занят, освобождаем..."
    
    # Стратегия 1: fuser по порту
    sudo fuser -k 8081/tcp 2>/dev/null || true
    sleep 1
    
    # Если всё ещё занят — пробуем другие методы
    if ss -tlnp 2>/dev/null | grep -q ":8081"; then
        echo "⚠️ fuser не помог, пробуем pkill..."
        
        # Стратегия 2: pkill по имени процесса main (это Go бинарник)
        sudo pkill -9 -f "main$" 2>/dev/null || true
        
        # Стратегия 3: pkill по пути go-build
        sudo pkill -9 -f "/tmp/go-build" 2>/dev/null || true
        
        # Стратегия 4: pkill по имени модуля
        sudo pkill -9 -f "pelvictrainer/backend" 2>/dev/null || true
        
        # Стратегия 5: pkill по "go run"
        sudo pkill -9 -f "go run" 2>/dev/null || true
        
        sleep 2
    fi
    
    # Финальная проверка
    if ss -tlnp 2>/dev/null | grep -q ":8081"; then
        echo "❌ Не удалось освободить порт 8081"
        ss -tlnp | grep ":8081"
        
        # Последнее средство: убить PID напрямую из ss
        PID=$(ss -tlnp 2>/dev/null | grep ":8081" | grep -o 'pid=[0-9]*' | head -1 | cut -d= -f2)
        if [ -n "$PID" ]; then
            echo "⚠️ Убиваем PID $PID напрямую..."
            sudo kill -9 $PID 2>/dev/null || true
            sleep 1
        fi
        
        # Ещё одна финальная проверка
        if ss -tlnp 2>/dev/null | grep -q ":8081"; then
            echo "❌ Критическая ошибка: порт 8081 всё ещё занят"
            exit 1
        fi
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
