package main

import (
	"fmt"
	"log"
	"os"

	"pelvictrainer/backend/internal/email"
)

func main() {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")
	fromName := os.Getenv("SMTP_FROM_NAME")

	fmt.Println("=== ТЕКУЩИЕ SMTP НАСТРОЙКИ ===")
	fmt.Printf("  Host:     %s\n", host)
	fmt.Printf("  Port:     %s\n", port)
	fmt.Printf("  Username: %s\n", username)
	fmt.Printf("  From:     %s\n", from)
	if len(password) >= 4 {
		fmt.Printf("  Password: %s... (скрыто)\n", password[:4])
	} else {
		fmt.Println("  Password: (пусто)")
	}
	fmt.Println("===============================")

	if host == "" || username == "" || password == "" {
		log.Fatal("❌ Проверьте .env файл — отсутствуют обязательные переменные")
	}

	// Информационное сообщение о выбранном методе
	switch port {
	case "2525":
		fmt.Println("ℹ️  Порт 2525: будет пробоваться STARTTLS, затем обычное соединение")
	case "465":
		fmt.Println("ℹ️  Порт 465: используется SSL/TLS")
	case "587":
		fmt.Println("ℹ️  Порт 587: используется STARTTLS")
	default:
		fmt.Printf("ℹ️  Нестандартный порт %s: пробуем разные методы подключения", port)
	}

	sender := email.NewSender(host, port, username, password, from, fromName)

	fmt.Printf("📧 Начинаем отправку письма на %s...\n", username)

	err := sender.SendMultiple(email.EmailMessage{
		To:      []string{username},
		Subject: "✅ Тест PelvicTrainer — SMTP работает!",
		Body: `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; padding: 20px; background: #f4f4f7;">
    <div style="max-width: 600px; margin: 0 auto; background: white; padding: 30px; border-radius: 12px;">
        <h2 style="color: #8B1538;">🎉 Поздравляем!</h2>
        <p style="font-size: 16px; line-height: 1.6;">
            SMTP-настройки работают корректно через порт <strong>` + port + `</strong>.<br>
            Теперь пользователи будут получать письма от <strong>PelvicTrainer</strong>.
        </p>
        <hr style="margin: 20px 0; border: none; border-top: 1px solid #eee;">
        <p style="color: #999; font-size: 12px;">
            Это тестовое письмо. Если вы получили его, значит отправка работает.
        </p>
    </div>
</body>
</html>`,
		IsHTML: true,
	})

	if err != nil {
		log.Fatalf("❌ Ошибка отправки: %v", err)
	}

	fmt.Println("✅ Письмо успешно отправлено! Проверьте почту.")
}
