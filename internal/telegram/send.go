package telegram

import (
	"github.com/st-kuptsov/mail2tg/pkg/metrics"
	"go.uber.org/zap"
	tb "gopkg.in/telebot.v3"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var Bot *tb.Bot

// SendToTelegram отправляет сообщение в указанный Telegram-канал с метриками и логированием.
func SendToTelegram(msg, channel string, logger *zap.SugaredLogger) {
	if channel == "" {
		logger.Warn("empty channel_id")
		return
	}

	chatID := parseChatID(channel)
	if chatID == 0 {
		logger.Warnf("invalid channel_id format: %s", channel)
		return
	}

	// Добавляем задержку перед отправкой (например, 1 секунда)
	time.Sleep(1 * time.Second)

	start := time.Now()
	_, err := Bot.Send(&tb.Chat{ID: chatID}, msg)
	duration := time.Since(start).Seconds()
	metrics.TgSendDuration.WithLabelValues(channel).Observe(duration)

	if err != nil {
		metrics.TgErrors.WithLabelValues(channel).Inc()

		if strings.Contains(err.Error(), "retry after") {
			retryAfter := extractRetryAfter(err)
			logger.Warnf("telegram API retry after %d sec", retryAfter)
			time.Sleep(time.Duration(retryAfter) * time.Second)

			// Повторная отправка
			startRetry := time.Now()
			_, errRetry := Bot.Send(&tb.Chat{ID: chatID}, msg)
			metrics.TgSendDuration.WithLabelValues(channel).Observe(time.Since(startRetry).Seconds())

			if errRetry != nil {
				logger.Errorf("failed to send message on retry: %v", errRetry)
				return
			}
		} else {
			logger.Errorf("telegram send error: %v", err)
			return
		}
	}

	logger.Infof("message sent to channel %s", channel)
	metrics.TgMessagesSent.WithLabelValues(channel).Inc()
}

// extractRetryAfter извлекает время ожидания из ошибки "retry after"
func extractRetryAfter(err error) int {
	re := regexp.MustCompile(`retry after (\d+)`)
	matches := re.FindStringSubmatch(err.Error())
	if len(matches) > 1 {
		retryAfter, _ := strconv.Atoi(matches[1])
		return retryAfter
	}
	return 1 // дефолтное значение
}

// parseChatID парсит строку channel_id в int64
func parseChatID(s string) int64 {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return id
}
