package telegram

import (
	"github.com/st-kuptsov/mail2tg/pkg/metrics"
	"go.uber.org/zap"
	tb "gopkg.in/telebot.v3"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var Bot *tb.Bot

// структура сообщения в очереди
type tgMessage struct {
	chatID int64
	text   string
	retry  int
	logger *zap.SugaredLogger
}

// очередь сообщений
var (
	queue      = make(chan tgMessage, 100) // буфер очереди
	queueMutex sync.Mutex
)

func init() {
	go worker()
}

// SendToTelegram помещает сообщение в очередь на отправку
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

	// помещаем в очередь
	select {
	case queue <- tgMessage{chatID: chatID, text: msg, retry: 0, logger: logger}:
	default:
		logger.Warn("telegram queue full, dropping message")
	}
}

// worker обрабатывает очередь сообщений
func worker() {
	for m := range queue {
		sendWithRetry(m)
	}
}

// sendWithRetry отправляет сообщение с экспоненциальным backoff и лимитом ретраев
func sendWithRetry(m tgMessage) {
	maxRetries := 5
	backoff := time.Second * 1

	for {
		start := time.Now()
		_, err := Bot.Send(&tb.Chat{ID: m.chatID}, m.text)
		duration := time.Since(start).Seconds()
		metrics.TgSendDuration.WithLabelValues(strconv.FormatInt(m.chatID, 10)).Observe(duration)

		if err == nil {
			metrics.TgMessagesSent.WithLabelValues(strconv.FormatInt(m.chatID, 10)).Inc()
			m.logger.Infof("message sent successfully to chat %d", m.chatID)
			return
		}

		metrics.TgErrors.WithLabelValues(strconv.FormatInt(m.chatID, 10)).Inc()
		m.logger.Errorf("failed to send message to chat %d: %v", m.chatID, err)

		// проверяем retry-after
		retryAfter := parseRetryAfter(err)
		if retryAfter > 0 {
			m.logger.Warnf("telegram API retry after %d seconds for chat %d", retryAfter, m.chatID)
			time.Sleep(time.Duration(retryAfter) * time.Second)
		} else {
			m.logger.Warnf("retrying message to chat %d after %s", m.chatID, backoff)
			time.Sleep(backoff)
			backoff *= 2 // экспоненциальный рост
		}

		m.retry++
		if m.retry >= maxRetries {
			m.logger.Errorf("message to chat %d failed after %d retries", m.chatID, maxRetries)
			return
		}
	}
}

// parseRetryAfter извлекает время из ошибки "retry after"
func parseRetryAfter(err error) int {
	re := regexp.MustCompile(`retry after (\d+)`)
	matches := re.FindStringSubmatch(err.Error())
	if len(matches) > 1 {
		r, _ := strconv.Atoi(matches[1])
		return r
	}
	return 0
}

// parseChatID парсит строку channel_id в int64
func parseChatID(s string) int64 {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return id
}
