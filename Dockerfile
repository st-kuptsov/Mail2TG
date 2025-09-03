# ===========================
# Сборка приложения
# ===========================
FROM golang:1.24-alpine AS builder

WORKDIR /usr/local/src/mail2tg

COPY go.mod go.sum ./
COPY pkg ./pkg
COPY mail2tg ./mail2tg

ARG APP_VERSION=dev
RUN go build -ldflags "-X main.version=${APP_VERSION}" -o ./bin/mail2tg ./mail2tg/main.go

# ===========================
# Минимальный runtime контейнер
# ===========================
FROM alpine:latest

WORKDIR /app
RUN apk add --no-cache curl

COPY --from=builder /usr/local/src/mail2tg/bin/mail2tg ./mail2tg
COPY mail-to-telegramm/config/config.example.yml ./config/config.yaml
COPY mail-to-telegramm/config/secrets.example.yml ./config/secrets.yaml
COPY mail-to-telegramm/healthcheck.sh ./healthcheck.sh
RUN chmod +x ./healthcheck.sh

EXPOSE 9090/tcp

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD /bin/sh /app/healthcheck.sh

CMD ["/app/mail2tg", "-config", "/app/config/config.yaml"]
