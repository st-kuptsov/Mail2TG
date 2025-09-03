# ===========================
# Сборка приложения
# ===========================
FROM golang:1.24-alpine AS builder

# Устанавливаем базовые инструменты
RUN apk add --no-cache git ca-certificates

WORKDIR /usr/src/mail2tg

# Копируем модули и исходники
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY config ./config
COPY internal ./internal
COPY pkg ./pkg

# Версия приложения
ARG APP_VERSION=dev

# Собираем бинарник статически для минимального образа
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w -X main.version=${APP_VERSION}" \
    -o ./bin/mail2tg ./cmd/mail2tg/main.go

# ===========================
# Минимальный runtime контейнер с поддержкой таймзоны
# ===========================
FROM alpine:latest

WORKDIR /app

# Устанавливаем curl для healthcheck и tzdata + timedatectl
RUN apk add --no-cache curl tzdata

# Копируем бинарник из builder
COPY --from=builder /usr/src/mail2tg/bin/mail2tg ./mail2tg

# Копируем конфигурацию и secrets
COPY config/config.example.yaml ./config/config.yaml
COPY config/secrets.example.yaml ./config/secrets.yaml

# Копируем healthcheck
COPY healthcheck.sh ./healthcheck.sh
RUN chmod +x ./healthcheck.sh

# Настройка таймзоны через переменную окружения TZ
ENV TZ=UTC
RUN cp /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# Прокидываем порт для метрик
EXPOSE 9090/tcp

# HEALTHCHECK
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD /bin/sh /app/healthcheck.sh

# Запуск приложения
CMD ["/app/mail2tg", "-config", "/app/config/config.yaml"]
