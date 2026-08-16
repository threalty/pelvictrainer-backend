#!/bin/bash
# ============================================
# Скрипт бекапов PostgreSQL + отправка в Yandex S3
# Запускается каждые 2 часа через cron
# ============================================

set -e

# Явный путь к конфигу rclone (работает при любом запуске)
export RCLONE_CONFIG="/home/deploy/.config/rclone/rclone.conf"

# Конфигурация
TIMESTAMP=$(date +%Y-%m-%d_%H-%M)
BACKUP_DIR="/tmp/pelvic-backups"
DB_CONTAINER="pelvic-postgres"
DB_USER="pelvic"
DB_NAME="pelvictrainer"
S3_BUCKET="yandex-s3:pelvetic"
RETENTION_DAYS=7
LOG_FILE="/var/log/pelvic-backup.log"

# Функция логирования
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log "🔄 Начало бекапа..."

# Создаём директорию
mkdir -p "$BACKUP_DIR"

# 1. Дамп базы данных из Docker контейнера
log "📦 Создание дампа PostgreSQL..."
docker exec "$DB_CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" -F c -f "/tmp/backup_${TIMESTAMP}.dump"
docker cp "$DB_CONTAINER:/tmp/backup_${TIMESTAMP}.dump" "$BACKUP_DIR/backup_${TIMESTAMP}.dump"
docker exec "$DB_CONTAINER" rm -f "/tmp/backup_${TIMESTAMP}.dump"

# 2. Сжимаем
log "🗜 Сжатие дампа..."
gzip -f "$BACKUP_DIR/backup_${TIMESTAMP}.dump"
BACKUP_FILE="$BACKUP_DIR/backup_${TIMESTAMP}.dump.gz"
SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
log "✅ Бекап создан: backup_${TIMESTAMP}.dump.gz ($SIZE)"

# 3. Отправляем в Yandex S3
log "📤 Отправка в Yandex Object Storage..."
rclone copy "$BACKUP_FILE" "$S3_BUCKET/db/" --progress
log "✅ Отправлено в S3"

# 4. Удаляем локальную копию
rm -f "$BACKUP_FILE"

# 5. Ротация: удаляем бекапы старше 7 дней в S3
log "🗑 Ротация бекапов (старше $RETENTION_DAYS дней)..."
rclone delete "$S3_BUCKET/db/" --min-age "${RETENTION_DAYS}d"
log "✅ Ротация завершена"

log "🎉 Бекап завершён успешно!"
echo "" >> "$LOG_FILE"
