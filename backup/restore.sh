#!/bin/bash
# ============================================
# Восстановление PostgreSQL из бекапа
# Использование: ./restore.sh backup_2026-08-16_12-00.dump.gz
# ============================================

set -e

export RCLONE_CONFIG="/home/deploy/.config/rclone/rclone.conf"


BACKUP_FILE="$1"
DB_CONTAINER="pelvic-postgres"
DB_USER="pelvic"
DB_NAME="pelvictrainer"
S3_BUCKET="yandex-s3:pelvictrainer-backups"

if [ -z "$BACKUP_FILE" ]; then
    echo "❌ Укажите имя файла бекапа:"
    echo "   ./restore.sh backup_2026-08-16_12-00.dump.gz"
    echo ""
    echo "Доступные бекапы:"
    rclone ls "$S3_BUCKET/db/"
    exit 1
fi

echo "⚠️  ВНИМАНИЕ! Будет восстановлена БД из $BACKUP_FILE"
echo "Текущие данные будут ЗАМЕНЕНЫ!"
read -p "Продолжить? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "❌ Отменено"
    exit 0
fi

echo "📥 Скачиваем бекап из S3..."
rclone copy "$S3_BUCKET/db/$BACKUP_FILE" /tmp/

echo "🔧 Распаковываем..."
gunzip -f "/tmp/$BACKUP_FILE"
DUMP_FILE="/tmp/${BACKUP_FILE%.gz}"

echo "📦 Копируем в контейнер..."
docker cp "$DUMP_FILE" "$DB_CONTAINER:/tmp/restore.dump"

echo "🔄 Восстанавливаем базу..."
docker exec "$DB_CONTAINER" pg_restore -U "$DB_USER" -d "$DB_NAME" --clean --if-exists /tmp/restore.dump

echo "🧹 Чистим..."
docker exec "$DB_CONTAINER" rm -f /tmp/restore.dump
rm -f "$DUMP_FILE"

echo "✅ Восстановление завершено!"
