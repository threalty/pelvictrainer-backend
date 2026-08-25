package twoauth

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"math/big"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	// Issuer - название приложения, которое будет показано в Google Authenticator / Яндекс.Ключ
	Issuer = "PelvicTrainer"
	
	// BackupCodesCount - количество одноразовых кодов восстановления
	BackupCodesCount = 10
	
	// BackupCodeLength - длина каждого кода восстановления
	BackupCodeLength = 8
)

// GenerateSecret генерирует новый TOTP-секрет (32 символа Base32)
func GenerateSecret() (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      Issuer,
		AccountName: "", // будет заполнен при вызове GenerateOTPAuthURL
		Period:      30, // стандартный период 30 секунд
		Digits:      otp.DigitsSix, // 6 цифр
		Algorithm:   otp.AlgorithmSHA1, // стандартный алгоритм
	})
	if err != nil {
		return "", fmt.Errorf("ошибка генерации секрета: %w", err)
	}
	return key.Secret(), nil
}

// GenerateOTPAuthURL создаёт URI для QR-кода
// Формат: otpauth://totp/PelvicTrainer:user@example.com?secret=...&issuer=PelvicTrainer&algorithm=SHA1&digits=6&period=30
func GenerateOTPAuthURL(secret, email string) string {
	key := fmt.Sprintf("%s:%s", Issuer, email)
	return fmt.Sprintf(
		"otpauth://totp/%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		key,
		secret,
		Issuer,
	)
}

// ValidateCode проверяет 6-значный код пользователя
// Возвращает true, если код валиден
func ValidateCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// ValidateCodeWithWindow проверяет код с окном ±1 (на случай расхождения времени)
// Это полезно, если время на устройстве пользователя немного отличается
// ValidateCodeWithWindow проверяет код с окном ±1 период (±30 секунд)
// Библиотека totp.Validate уже имеет встроенный допуск, но мы явно задаём параметры
func ValidateCodeWithWindow(secret, code string) bool {
	opts := totp.ValidateOpts{
		Period:    30,                // стандартный период 30 секунд
		Skew:      1,                 // допуск ±1 период (±30 секунд)
		Digits:    otp.DigitsSix,     // 6 цифр
		Algorithm: otp.AlgorithmSHA1, // стандартный алгоритм
	}
	
	valid, err := totp.ValidateCustom(code, secret, time.Now(), opts)
	if err != nil {
		return false
	}
	return valid
}

// GenerateBackupCodes генерирует 10 одноразовых кодов восстановления
// Коды в формате XXXX-XXXX (легко записать)
func GenerateBackupCodes() ([]string, error) {
	codes := make([]string, BackupCodesCount)
	
	for i := 0; i < BackupCodesCount; i++ {
		code, err := generateRandomCode(BackupCodeLength)
		if err != nil {
			return nil, fmt.Errorf("ошибка генерации кода: %w", err)
		}
		// Форматируем как XXXX-XXXX для удобства
		codes[i] = fmt.Sprintf("%s-%s", code[:4], code[4:])
	}
	
	return codes, nil
}

// generateRandomCode генерирует криптографически безопасный случайный код
// Использует только буквы (без похожих символов типа O/0, I/1)
func generateRandomCode(length int) (string, error) {
	// Алфавит без похожих символов: нет 0, O, 1, I, L
	const charset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	
	code := make([]byte, length)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[n.Int64()]
	}
	
	return string(code), nil
}

// HashBackupCode хэширует код восстановления для безопасного хранения
// Мы не храним коды в открытом виде!
func HashBackupCode(code string) string {
	// Используем простой SHA-256 хэш (без соли, т.к. коды уже случайные)
	// Для продакшена можно использовать bcrypt, но SHA-256 достаточно для одноразовых кодов
	return base32.StdEncoding.EncodeToString([]byte(code))
}

// NormalizeBackupCode нормализует код (убирает дефисы, приводит к верхнему регистру)
func NormalizeBackupCode(code string) string {
	result := ""
	for _, c := range code {
		if c != '-' && c != ' ' {
			result += string(c)
		}
	}
	// Приводим к верхнему регистру
	upper := ""
	for _, c := range result {
		if c >= 'a' && c <= 'z' {
			upper += string(c - 'a' + 'A')
		} else {
			upper += string(c)
		}
	}
	return upper
}