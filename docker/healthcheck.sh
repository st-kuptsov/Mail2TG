#!/bin/bash

# URL метрик
METRICS_URL="http://localhost:9090/metrics"
# Основная метрика, по которой проверяем "живость" сервиса
METRIC_NAME="mail2tg_mailbox_successful_checks_total"
# Файл для хранения предыдущего значения метрики
STATE_FILE="/tmp/last_mail_check"

mkdir -p $(dirname "$STATE_FILE")

# Проверяем доступность HTTP
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$METRICS_URL")
if [ "$HTTP_CODE" -ne 200 ]; then
    echo "Metrics endpoint unreachable, HTTP code: $HTTP_CODE"
    exit 1
fi

# Получаем текущее значение метрики (суммируем все экземпляры, если есть лейблы)
CURRENT_VALUE=$(curl -s "$METRICS_URL" | grep "^$METRIC_NAME" | awk '{sum += $2} END {print sum}')

# Если метрика не найдена — сервис не работает
if [ -z "$CURRENT_VALUE" ]; then
    echo "Metric $METRIC_NAME not found"
    exit 1
fi

# Проверяем изменение метрики
if [ -f "$STATE_FILE" ]; then
    LAST_VALUE=$(cat "$STATE_FILE")
    if [ "$CURRENT_VALUE" -eq "$LAST_VALUE" ]; then
        echo "Metric $METRIC_NAME did not change since last check"
        exit 1
    fi
fi

# Сохраняем текущее значение для следующей проверки
echo "$CURRENT_VALUE" > "$STATE_FILE"

echo "Healthcheck OK"
exit 0
