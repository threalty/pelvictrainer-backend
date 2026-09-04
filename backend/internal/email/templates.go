package email

// TemplatePasswordReset шаблон письма для сброса пароля
const TemplatePasswordReset = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Восстановление пароля</title>
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Arial, sans-serif; background-color: #f4f4f7;">
    <table width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color: #f4f4f7; padding: 40px 20px;">
        <tr>
            <td align="center">
                <table width="600" cellpadding="0" cellspacing="0" border="0" style="background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 12px rgba(0,0,0,0.1);">
                    <!-- Шапка -->
                    <tr>
                        <td style="background: linear-gradient(135deg, #8B1538 0%, #6B0F2C 100%); padding: 40px 30px; text-align: center;">
                            <h1 style="color: #ffffff; margin: 0; font-size: 28px; font-weight: 700;">💪 PelvicTrainer</h1>
                        </td>
                    </tr>
                    
                    <!-- Содержимое -->
                    <tr>
                        <td style="padding: 40px 30px;">
                            <h2 style="color: #1a1a1a; margin: 0 0 20px 0; font-size: 24px; font-weight: 600;">
                                Здравствуйте, {{.UserName}}!
                            </h2>
                            
                            <p style="color: #4a4a4a; font-size: 16px; line-height: 1.6; margin: 0 0 20px 0;">
                                Мы получили запрос на восстановление пароля для вашего аккаунта в PelvicTrainer.
                            </p>
                            
                            <p style="color: #4a4a4a; font-size: 16px; line-height: 1.6; margin: 0 0 30px 0;">
                                Чтобы установить новый пароль, нажмите на кнопку ниже:
                            </p>
                            
                            <!-- Кнопка -->
                            <table width="100%" cellpadding="0" cellspacing="0" border="0">
                                <tr>
                                    <td align="center" style="padding: 10px 0;">
                                        <a href="{{.ResetLink}}" 
                                           style="display: inline-block; background: linear-gradient(135deg, #8B1538 0%, #6B0F2C 100%); color: #ffffff; text-decoration: none; padding: 16px 40px; border-radius: 8px; font-size: 16px; font-weight: 600; box-shadow: 0 4px 12px rgba(139, 21, 56, 0.3);">
                                            Сбросить пароль
                                        </a>
                                    </td>
                                </tr>
                            </table>
                            
                            <p style="color: #666666; font-size: 14px; line-height: 1.6; margin: 30px 0 10px 0;">
                                Ссылка действительна в течение <strong>1 часа</strong>.
                            </p>
                            
                            <p style="color: #999999; font-size: 13px; line-height: 1.5; margin: 20px 0 0 0;">
                                Если вы не запрашивали сброс пароля, просто проигнорируйте это письмо. Ваш пароль останется прежним.
                            </p>
                        </td>
                    </tr>
                    
                    <!-- Подвал -->
                    <tr>
                        <td style="background-color: #f8f8f8; padding: 25px 30px; text-align: center; border-top: 1px solid #eeeeee;">
                            <p style="color: #999999; font-size: 13px; margin: 0; line-height: 1.5;">
                                © 2026 PelvicTrainer. Все права защищены.<br>
                                Это письмо отправлено автоматически, пожалуйста, не отвечайте на него.
                            </p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>`

// TemplateWelcome шаблон приветственного письма
const TemplateWelcome = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <title>Добро пожаловать!</title>
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Arial, sans-serif; background-color: #f4f4f7;">
    <table width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color: #f4f4f7; padding: 40px 20px;">
        <tr>
            <td align="center">
                <table width="600" cellpadding="0" cellspacing="0" border="0" style="background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 12px rgba(0,0,0,0.1);">
                    <tr>
                        <td style="background: linear-gradient(135deg, #8B1538 0%, #6B0F2C 100%); padding: 40px 30px; text-align: center;">
                            <h1 style="color: #ffffff; margin: 0; font-size: 28px; font-weight: 700;">💪 Добро пожаловать!</h1>
                        </td>
                    </tr>
                    <tr>
                        <td style="padding: 40px 30px;">
                            <h2 style="color: #1a1a1a; margin: 0 0 20px 0; font-size: 24px; font-weight: 600;">
                                Здравствуйте, {{.UserName}}! 🎉
                            </h2>
                            <p style="color: #4a4a4a; font-size: 16px; line-height: 1.6; margin: 0 0 15px 0;">
                                Рады приветствовать вас в <strong>PelvicTrainer</strong> — вашем персональном тренажёре для укрепления мышц тазового дна.
                            </p>
                            <p style="color: #4a4a4a; font-size: 16px; line-height: 1.6; margin: 0 0 20px 0;">
                                Вот что вас ждёт:
                            </p>
                            <ul style="color: #4a4a4a; font-size: 15px; line-height: 1.8; margin: 0 0 20px 20px; padding-left: 10px;">
                                <li>✅ Персональные программы тренировок</li>
                                <li>📊 Подробная статистика и календарь</li>
                                <li>🏆 Система достижений и мотивации</li>
                                <li>🔔 Умные напоминания о тренировках</li>
                                <li>☁️ Синхронизация между устройствами</li>
                            </ul>
                            <p style="color: #4a4a4a; font-size: 16px; line-height: 1.6; margin: 20px 0 0 0;">
                                Начните свой путь к здоровью уже сегодня!
                            </p>
                        </td>
                    </tr>
                    <tr>
                        <td style="background-color: #f8f8f8; padding: 25px 30px; text-align: center; border-top: 1px solid #eeeeee;">
                            <p style="color: #999999; font-size: 13px; margin: 0;">
                                © 2026 PelvicTrainer. Все права защищены.
                            </p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>`

// TemplateSubscriptionActivated шаблон письма об активации Premium
const TemplateSubscriptionActivated = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <title>Premium активирован</title>
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Arial, sans-serif; background-color: #f4f4f7;">
    <table width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color: #f4f4f7; padding: 40px 20px;">
        <tr>
            <td align="center">
                <table width="600" cellpadding="0" cellspacing="0" border="0" style="background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 12px rgba(0,0,0,0.1);">
                    <tr>
                        <td style="background: linear-gradient(135deg, #8B1538 0%, #6B0F2C 100%); padding: 40px 30px; text-align: center;">
                            <h1 style="color: #ffffff; margin: 0; font-size: 28px; font-weight: 700;">💎 Premium активирован!</h1>
                        </td>
                    </tr>
                    <tr>
                        <td style="padding: 40px 30px;">
                            <h2 style="color: #1a1a1a; margin: 0 0 20px 0; font-size: 24px; font-weight: 600;">
                                Здравствуйте, {{.UserName}}!
                            </h2>
                            <p style="color: #4a4a4a; font-size: 16px; line-height: 1.6; margin: 0 0 20px 0;">
                                Ваша Premium подписка <strong>{{.Plan}}</strong> успешно активирована.
                            </p>
                            <p style="color: #4a4a4a; font-size: 16px; line-height: 1.6; margin: 0 0 20px 0;">
                                Теперь вам доступны все функции PelvicTrainer без ограничений:
                            </p>
                            <ul style="color: #4a4a4a; font-size: 15px; line-height: 1.8; margin: 0 0 20px 20px;">
                                <li>♾️ Неограниченное количество тренировок</li>
                                <li>🎯 Все программы тренировок</li>
                                <li>📈 Расширенная аналитика</li>
                                <li>🔇 Без рекламы</li>
                            </ul>
                            <p style="color: #4a4a4a; font-size: 16px; line-height: 1.6; margin: 20px 0 0 0;">
                                Желаем вам отличных тренировок и крепкого здоровья!
                            </p>
                        </td>
                    </tr>
                    <tr>
                        <td style="background-color: #f8f8f8; padding: 25px 30px; text-align: center; border-top: 1px solid #eeeeee;">
                            <p style="color: #999999; font-size: 13px; margin: 0;">
                                © 2026 PelvicTrainer. Все права защищены.
                            </p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>`

// TemplateNewDeviceLogin шаблон уведомления о входе с нового устройства
const TemplateNewDeviceLogin = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <title>Новый вход в аккаунт</title>
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Arial, sans-serif; background-color: #f4f4f7;">
    <table width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color: #f4f4f7; padding: 40px 20px;">
        <tr>
            <td align="center">
                <table width="600" cellpadding="0" cellspacing="0" border="0" style="background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 12px rgba(0,0,0,0.1);">
                    <!-- Шапка -->
                    <tr>
                        <td style="background: linear-gradient(135deg, #2E7D32 0%, #1B5E20 100%); padding: 40px 30px; text-align: center;">
                            <h1 style="color: #ffffff; margin: 0; font-size: 28px; font-weight: 700;">🔐 Новый вход</h1>
                        </td>
                    </tr>
                    
                    <!-- Содержимое -->
                    <tr>
                        <td style="padding: 40px 30px;">
                            <h2 style="color: #1a1a1a; margin: 0 0 20px 0; font-size: 24px; font-weight: 600;">
                                Здравствуйте, {{.UserName}}!
                            </h2>
                            
                            <p style="color: #4a4a4a; font-size: 16px; line-height: 1.6; margin: 0 0 20px 0;">
                                Мы зафиксировали вход в ваш аккаунт PelvicTrainer с нового устройства.
                            </p>
                            
                            <!-- Детали входа -->
                            <table width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color: #f8f8f8; border-radius: 8px; margin: 25px 0; padding: 20px;">
                                <tr>
                                    <td>
                                        <p style="color: #666666; font-size: 13px; margin: 0 0 5px 0; text-transform: uppercase; letter-spacing: 0.5px;">
                                            📍 IP-адрес
                                        </p>
                                        <p style="color: #1a1a1a; font-size: 15px; font-family: monospace; margin: 0 0 15px 0; font-weight: 600;">
                                            {{.IPAddress}}
                                        </p>
                                        
                                        <p style="color: #666666; font-size: 13px; margin: 0 0 5px 0; text-transform: uppercase; letter-spacing: 0.5px;">
                                            🌐 Устройство
                                        </p>
                                        <p style="color: #1a1a1a; font-size: 15px; margin: 0 0 15px 0; font-weight: 600;">
                                            {{.DeviceInfo}}
                                        </p>
                                        
                                        <p style="color: #666666; font-size: 13px; margin: 0 0 5px 0; text-transform: uppercase; letter-spacing: 0.5px;">
                                            🕐 Время
                                        </p>
                                        <p style="color: #1a1a1a; font-size: 15px; margin: 0; font-weight: 600;">
                                            {{.LoginTime}}
                                        </p>
                                    </td>
                                </tr>
                            </table>
                            
                            <p style="color: #4a4a4a; font-size: 16px; line-height: 1.6; margin: 0 0 20px 0;">
                                Если это были вы — можете игнорировать это письмо. Ваш аккаунт в безопасности.
                            </p>
                            
                            <!-- Предупреждение -->
                            <div style="background-color: #FFF3E0; border-left: 4px solid #FF9800; padding: 15px 20px; border-radius: 4px; margin: 25px 0;">
                                <p style="color: #E65100; font-size: 14px; line-height: 1.6; margin: 0; font-weight: 500;">
                                    ⚠️ <strong>Не узнаёте это устройство?</strong><br>
                                    Немедленно смените пароль и включите двухфакторную аутентификацию в настройках аккаунта.
                                </p>
                            </div>
                        </td>
                    </tr>
                    
                    <!-- Подвал -->
                    <tr>
                        <td style="background-color: #f8f8f8; padding: 25px 30px; text-align: center; border-top: 1px solid #eeeeee;">
                            <p style="color: #999999; font-size: 13px; margin: 0; line-height: 1.5;">
                                © 2026 PelvicTrainer. Все права защищены.<br>
                                Это автоматическое уведомление о безопасности.
                            </p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>`