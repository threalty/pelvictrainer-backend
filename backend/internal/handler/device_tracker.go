package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"pelvictrainer/backend/internal/email"
)

// DeviceTracker содержит общие функции для отслеживания устройств
type DeviceTracker struct {
	db          *pgxpool.Pool
	emailSender *email.Sender
}

// NewDeviceTracker создаёт трекер устройств
func NewDeviceTracker(db *pgxpool.Pool, emailSender *email.Sender) *DeviceTracker {
	return &DeviceTracker{
		db:          db,
		emailSender: emailSender,
	}
}

// HashUserAgent создаёт SHA256 хеш от User-Agent
func (dt *DeviceTracker) HashUserAgent(userAgent string) string {
	hash := sha256.Sum256([]byte(userAgent))
	return hex.EncodeToString(hash[:])
}

// ParseDeviceInfo извлекает упрощённое описание устройства из User-Agent
func (dt *DeviceTracker) ParseDeviceInfo(userAgent string) string {
	ua := strings.ToLower(userAgent)

	// Определяем устройство
	var device string
	switch {
	case strings.Contains(ua, "android"):
		device = "Android"
	case strings.Contains(ua, "iphone"), strings.Contains(ua, "ipad"):
		device = "iOS"
	case strings.Contains(ua, "windows"):
		device = "Windows"
	case strings.Contains(ua, "macintosh"), strings.Contains(ua, "mac os"):
		device = "macOS"
	case strings.Contains(ua, "linux"):
		device = "Linux"
	default:
		device = "Unknown Device"
	}

	// Определяем браузер
	var browser string
	switch {
	case strings.Contains(ua, "pelvictrainer") || strings.Contains(ua, "okhttp"):
		browser = "PelvicTrainer App"
	case strings.Contains(ua, "chrome") && !strings.Contains(ua, "edge"):
		browser = "Chrome"
	case strings.Contains(ua, "firefox"):
		browser = "Firefox"
	case strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome"):
		browser = "Safari"
	case strings.Contains(ua, "edge"):
		browser = "Edge"
	default:
		browser = "Browser"
	}

	return fmt.Sprintf("%s on %s", browser, device)
}

// LogLogin сохраняет запись о входе в login_history (игнорируя дубликаты)
func (dt *DeviceTracker) LogLogin(ctx context.Context, userID int, ip string, userAgent string) {
	userAgentHash := dt.HashUserAgent(userAgent)
	deviceInfo := dt.ParseDeviceInfo(userAgent)

	_, err := dt.db.Exec(ctx, `
		INSERT INTO login_history (user_id, ip_address, user_agent, user_agent_hash, device_info, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id, user_agent_hash) DO UPDATE SET
			ip_address = EXCLUDED.ip_address,
			created_at = NOW()
	`, userID, ip, userAgent, userAgentHash, deviceInfo)

	if err != nil {
		log.Printf("⚠️ Не удалось сохранить запись о входе: %v", err)
	}
}

// CheckAndNotifyNewDevice проверяет новое устройство и отправляет email если нужно
func (dt *DeviceTracker) CheckAndNotifyNewDevice(ctx context.Context, userID int, emailAddr, userName, ip, userAgent string) {
	userAgentHash := dt.HashUserAgent(userAgent)
	deviceInfo := dt.ParseDeviceInfo(userAgent)

	// Проверяем был ли этот хеш уже для данного пользователя
	var exists bool
	err := dt.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM login_history 
			WHERE user_id = $1 AND user_agent_hash = $2
		)
	`, userID, userAgentHash).Scan(&exists)

	if err != nil {
		log.Printf("⚠️ Ошибка проверки login_history: %v", err)
		// Продолжаем даже если проверка не удалась
	}

	isNewDevice := !exists

	// Сохраняем запись (upsert)
	dt.LogLogin(ctx, userID, ip, userAgent)

	// Если устройство новое — отправляем email
	if isNewDevice && dt.emailSender != nil {
		loginTime := time.Now().Format("02.01.2006 в 15:04")
		go func() {
			log.Printf("📧 Отправляем уведомление о новом устройстве: user=%d, device=%s, ip=%s",
				userID, deviceInfo, ip)

			if err := dt.emailSender.SendNewDeviceLogin(emailAddr, userName, ip, deviceInfo, loginTime); err != nil {
				log.Printf("⚠️ Не удалось отправить уведомление о новом устройстве: %v", err)
			} else {
				log.Printf("✅ Уведомление о новом устройстве отправлено на %s", emailAddr)
			}
		}()
	} else if isNewDevice {
		log.Printf("⚠️ Обнаружено новое устройство для user=%d, но emailSender не настроен", userID)
	}
}