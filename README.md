# Mail2TG

**Mail2TG** — сервис для маршрутизации писем из почтовых ящиков в Telegram.  
Он получает уведомления по email (например, от системы мониторинга), фильтрует по правилам и отправляет сообщения в заданные Telegram-каналы. Поддерживаются Prometheus-метрики и логирование.

---

## Функционал

- Проверка IMAP-почты на новые письма с указанным интервалом.
- Декодирование текста и HTML-сообщений.
- Маршрутизация сообщений по регулярным выражениям.
- Отправка сообщений в Telegram с retry при необходимости.
- Метрики Prometheus (`uptime`, количество отправленных сообщений, ошибки).
- Graceful shutdown и обработка паник.
- Логирование с уровнями `debug/info/warn/error`.

---

## Установка и запуск

### 1. Подготовьте конфигурацию
В каталоге `config/` создайте два файла:
- **config.yaml** — настройки IMAP, маршрутизации, логирования и порта сервиса.
- **secrets.yaml** — секреты (пароль IMAP и токен Telegram).

Примеры:
- [`config/config.example.yaml`](config/config.example.yaml)
- [`config/secrets.example.yaml`](config/secrets.example.yaml)


```yaml
imap:
  host: "imap.yandex.ru"               # Адрес IMAP-сервера
  port: 993                            # Порт подключения (обычно 993 для TLS)
  username: "user@example.com"         # Логин для входа на почту
  # password хранится в secrets.yaml и не включается сюда для безопасности

telegram:
  default_channel: "-1111111111111"    # Канал по умолчанию для писем, если ни одно правило не сработало
  errors_channel: "-2222222222222"     # Канал для ошибок работы бота (IMAP, Telegram API и т.п.)

route:
  - folders:
      - name: "INBOX"                  # Имя папки IMAP, которую проверяем
        rules:
          - pattern: "TESTING"         # Регулярное выражение для темы письма
                                       # Поддерживаются стандартные Go-regular expressions (RE2),
            channel: "-3333333333333"  # Канал, куда отправлять письма при совпадении
          - pattern: "PREPROD"
            channel: "-4444444444444"
          - pattern: "PROD"
            channel: "-5555555555555"

log_settings:
  directory: "logs"                    # Директория хранения логов
  filename: "app.log"                  # Имя лог-файла
  max_size: 100                        # Максимальный размер файла в MB
  max_backups: 10                      # Количество старых лог-файлов, которые сохраняем
  max_age: 7                           # Срок хранения файлов в днях
  compress: true                       # Сжимать старые файлы gzip
  level: "info"                        # Уровень логирования: debug, info, warn, error
  console_enabled: true                # Писать логи на консоль

check_interval: 60                     # Интервал проверки почты в секундах
secrets: config/secrets.yaml           # Путь к файлу с секретами (пароль IMAP и др.)
service_port: 9090                     # Порт HTTP-сервера для метрик Prometheus и healthcheck
```
```yaml
imap:
  password: "YOUR_IMAP_PASSWORD"
telegram:
  token: "YOUR_TELEGRAM_BOT_TOKEN"
```

### 2. Сборка и запуск через Docker

Сборка и запуск автоматизированы в скрипте `builder.sh`.

```bash
./builder.sh
```
Скрипт:
- получает текущую версию приложения (тег git или hash коммита),
- пересобирает Docker-образ,
- поднимает контейнер через docker-compose.

### 3. Docker Compose

Файл docker/docker-compose.yaml запускает сервис:
- публикует порт 9090 для Prometheus-метрик,
- монтирует директории config/ и logs/ из проекта в контейнер,
- задаёт таймзону контейнера через TZ.

### 4. Проверка работы

Приложение запускается внутри контейнера:
```bash
docker ps
```

Логи доступны по пути logs/app.log или через Docker:
```bash
docker logs -f mail2tg
```

## HEALTHCHECK
- Контейнер содержит встроенный HEALTHCHECK, который проверяет метрику mailbox_successful_checks_total.
- Если метрика не увеличивается между проверками, контейнер считается "unhealthy".
- Период проверки — каждые 30 секунд, таймаут — 10 секунд, с 3 попытками.


## Метрики Prometheus
Mail2TG экспортирует метрики Prometheus для мониторинга состояния сервиса, обработки почты и отправки сообщений в Telegram.

### Метрики состояния сервиса

| Метрика          | Тип   | Описание                                                                          |
|------------------|-------|-----------------------------------------------------------------------------------|
| `mail2tg_uptime` | Gauge | Время работы сервиса в секундах. Полезно для alerting и отслеживания доступности. |

### Метрики почты

| Метрика                                    | Тип       | Описание                                                                                    |
|--------------------------------------------|-----------|---------------------------------------------------------------------------------------------|
| `mail2tg_mailbox_successful_checks_total`  | Counter   | Количество успешных проверок почтового ящика.                                               |
| `mail2tg_mailbox_received_messages_total`  | Counter   | Количество полученных писем.                                                                |
| `mail2tg_mailbox_errors_total`             | Counter   | Количество ошибок при проверке почты.                                                       |
| `mail2tg_mail_processing_duration_seconds` | Histogram | Время обработки писем в секундах. Позволяет видеть задержки и производительность обработки. |

### Метрики Telegram

| Метрика                                  | Тип       | Лейблы       | Описание                                                                        |
|------------------------------------------|-----------|--------------|---------------------------------------------------------------------------------|
| `mail2tg_telegram_messages_sent_total`   | Counter   | `channel_id` | Количество сообщений, успешно отправленных в Telegram по каждому каналу.        |
| `mail2tg_telegram_messages_errors_total` | Counter   | `channel_id` | Количество ошибок при отправке сообщений в Telegram по каждому каналу.          |
| `mail2tg_telegram_send_duration_seconds` | Histogram | `channel_id` | Время отправки сообщений в Telegram. Позволяет отслеживать задержки по каналам. |

## Логирование
Примеры сообщений:
```text
INFO  starting Mail2TG
DEBUG initializing metrics server
INFO  metrics server started port=8080
DEBUG initializing telegram bot
INFO  telegram bot initialized
DEBUG starting scheduler
INFO  Scheduler stopped
```
- Все сообщения логируются через zap SugaredLogger.
- Стиль сообщений — строчные буквы, короткие описательные фразы.
- Используются уровни: debug, info, warn, error.

## Graceful shutdown
- Контекст context.Context используется для корректного завершения всех фоновых горутин.
- Планировщик и HTTP-сервер останавливаются при получении сигнала SIGINT или SIGTERM.
- Метрика uptime продолжает работать до остановки сервиса.

## Пример workflow
1. Новое письмо приходит на IMAP.
2. Планировщик проверяет папку.
3. Письмо декодируется.
4. Проверяются правила маршрутизации:
5. Если совпадает регулярное выражение, письмо отправляется в соответствующий Telegram-канал.
6. Если нет совпадений, используется канал по умолчанию.
7. Метрики Prometheus обновляются.
8. Логи пишутся на консоль и могут использоваться для мониторинга.
