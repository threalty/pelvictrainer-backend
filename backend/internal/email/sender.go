package email

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Sender сервис отправки email через SMTP
type Sender struct {
	host     string
	port     string
	username string
	password string
	from     string
	fromName string
}

// NewSender создаёт новый email sender
func NewSender(host, port, username, password, from, fromName string) *Sender {
	return &Sender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		fromName: fromName,
	}
}

// EmailMessage структура письма
type EmailMessage struct {
	To      []string
	Subject string
	Body    string
	IsHTML  bool
}

// SendMultiple отправляет письмо с автоматическим выбором метода подключения
func (s *Sender) SendMultiple(msg EmailMessage) error {
	if len(msg.To) == 0 {
		return fmt.Errorf("не указан получатель")
	}

	// Формируем заголовки письма
	var emailBody strings.Builder
	emailBody.WriteString(fmt.Sprintf("From: %s <%s>\r\n", s.fromName, s.from))
	emailBody.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	emailBody.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	emailBody.WriteString("MIME-Version: 1.0\r\n")
	emailBody.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))

	if msg.IsHTML {
		emailBody.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	} else {
		emailBody.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	}

	emailBody.WriteString("\r\n")
	emailBody.WriteString(msg.Body)

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	// Пробуем разные методы в зависимости от порта
	switch s.port {
	case "465":
		// SSL/TLS (порт 465)
		log.Printf("📡 Подключение по SSL к %s...", addr)
		return s.sendWithSSL(dialer, addr, msg.To, emailBody.String())
	case "587":
		// STARTTLS (порт 587)
		log.Printf("📡 Подключение по STARTTLS к %s...", addr)
		return s.sendWithSTARTTLS(dialer, addr, msg.To, emailBody.String())
	case "2525":
		// Альтернативный порт — пробуем сначала STARTTLS, потом обычное соединение
		log.Printf("📡 Подключение к %s (порт 2525)...", addr)
		err := s.sendWithSTARTTLS(dialer, addr, msg.To, emailBody.String())
		if err != nil {
			log.Printf("⚠️ STARTTLS не сработал: %v. Пробую обычное соединение...", err)
			err = s.sendPlain(dialer, addr, msg.To, emailBody.String())
			if err != nil {
				return fmt.Errorf("порт 2525 не поддерживает ни STARTTLS, ни обычное соединение: %w", err)
			}
		}
		return nil
	default:
		// Для других портов пробуем сначала STARTTLS, потом SSL
		log.Printf("📡 Подключение к нестандартному порту %s...", s.port)
		err := s.sendWithSTARTTLS(dialer, addr, msg.To, emailBody.String())
		if err != nil {
			err = s.sendWithSSL(dialer, addr, msg.To, emailBody.String())
			if err != nil {
				return fmt.Errorf("не удалось подключиться к %s: %w", addr, err)
			}
		}
		return nil
	}
}

// sendWithSSL отправка через SSL/TLS (порт 465)
func (s *Sender) sendWithSSL(dialer *net.Dialer, addr string, recipients []string, body string) error {
	tlsConfig := &tls.Config{
		ServerName: s.host,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("ошибка SSL-подключения: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("ошибка создания SMTP клиента: %w", err)
	}
	defer client.Close()

	// Авторизация
	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("ошибка авторизации: %w", err)
	}

	return s.sendMailBody(client, s.from, recipients, body)
}

// sendWithSTARTTLS отправка через STARTTLS
func (s *Sender) sendWithSTARTTLS(dialer *net.Dialer, addr string, recipients []string, body string) error {
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("ошибка создания клиента: %w", err)
	}
	defer client.Close()

	// STARTTLS
	tlsConfig := &tls.Config{ServerName: s.host}
	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("ошибка STARTTLS: %w", err)
	}

	// Авторизация
	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("ошибка авторизации: %w", err)
	}

	return s.sendMailBody(client, s.from, recipients, body)
}

// sendPlain отправка через обычное соединение без шифрования (для порта 2525)
func (s *Sender) sendPlain(dialer *net.Dialer, addr string, recipients []string, body string) error {
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("ошибка создания клиента: %w", err)
	}
	defer client.Close()

	// Авторизация без STARTTLS
	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("ошибка авторизации: %w", err)
	}

	return s.sendMailBody(client, s.from, recipients, body)
}

// sendMailBody отправляет тело письма через уже авторизованный клиент
func (s *Sender) sendMailBody(client *smtp.Client, from string, recipients []string, body string) error {
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("ошибка MAIL FROM: %w", err)
	}

	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("ошибка RCPT TO (%s): %w", recipient, err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("ошибка DATA: %w", err)
	}

	_, err = writer.Write([]byte(body))
	if err != nil {
		return fmt.Errorf("ошибка записи тела письма: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("ошибка закрытия writer: %w", err)
	}

	err = client.Quit()
	if err != nil {
		return fmt.Errorf("ошибка QUIT: %w", err)
	}

	log.Printf("✅ Письмо отправлено на %v", recipients)
	return nil
}

// SendWithTemplate отправляет письмо с HTML-шаблоном
func (s *Sender) SendWithTemplate(to []string, subject string, templateHTML string, data interface{}) error {
	tmpl, err := template.New("email").Parse(templateHTML)
	if err != nil {
		return fmt.Errorf("ошибка парсинга шаблона: %w", err)
	}

	var body strings.Builder
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("ошибка выполнения шаблона: %w", err)
	}

	msg := EmailMessage{
		To:      to,
		Subject: subject,
		Body:    body.String(),
		IsHTML:  true,
	}

	return s.SendMultiple(msg)
}

// === СПЕЦИАЛИЗИРОВАННЫЕ МЕТОДЫ ===

// SendPasswordReset отправляет письмо для сброса пароля
func (s *Sender) SendPasswordReset(email string, userName string, resetLink string) error {
	subject := "Восстановление пароля в PelvicTrainer"
	return s.SendWithTemplate([]string{email}, subject, TemplatePasswordReset, map[string]string{
		"UserName":  userName,
		"ResetLink": resetLink,
	})
}

// SendWelcome отправляет приветственное письмо
func (s *Sender) SendWelcome(email string, userName string) error {
	subject := "Добро пожаловать в PelvicTrainer!"
	return s.SendWithTemplate([]string{email}, subject, TemplateWelcome, map[string]string{
		"UserName": userName,
	})
}

// SendSubscriptionActivated отправляет письмо об активации Premium
func (s *Sender) SendSubscriptionActivated(email string, userName string, plan string) error {
	subject := "Premium подписка активирована в PelvicTrainer"
	return s.SendWithTemplate([]string{email}, subject, TemplateSubscriptionActivated, map[string]string{
		"UserName": userName,
		"Plan":     plan,
	})
}

// SendBroadcast отправляет массовую рассылку
func (s *Sender) SendBroadcast(recipients []string, subject string, htmlContent string) error {
	if len(recipients) == 0 {
		return fmt.Errorf("список получателей пуст")
	}

	successCount := 0
	failCount := 0

	for _, recipient := range recipients {
		msg := EmailMessage{
			To:      []string{recipient},
			Subject: subject,
			Body:    htmlContent,
			IsHTML:  true,
		}
		if err := s.SendMultiple(msg); err != nil {
			failCount++
			log.Printf("⚠️ Не удалось отправить на %s: %v", recipient, err)
		} else {
			successCount++
		}
		time.Sleep(500 * time.Millisecond)
	}

	log.Printf("📊 Рассылка завершена: успешно=%d, ошибок=%d", successCount, failCount)
	return nil
}