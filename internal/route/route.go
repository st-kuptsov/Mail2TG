package route

import (
	"fmt"
	"regexp"

	"github.com/st-kuptsov/mail2tg/config"
	"github.com/st-kuptsov/mail2tg/internal/telegram"
	"go.uber.org/zap"
)

// RouteMessage проверяет тему письма по правилам маршрутизации и отправляет
// его в соответствующий Telegram-канал. Если ни одно правило не совпало,
// сообщение отправляется в канал по умолчанию.
func RouteMessage(cfg *config.Config, f config.Folder, subject, body string, logger *zap.SugaredLogger) {
	for _, rule := range f.Rules {
		logger.Debugw("checking pattern for email",
			"pattern", rule.Pattern,
			"subject", subject,
		)

		matched, err := regexp.MatchString(rule.Pattern, subject)
		if err != nil {
			logger.Warnw("failed to match pattern", "pattern", rule.Pattern, "error", err)
			continue
		}

		if matched {
			logger.Debugw("message routed to channel",
				"channel", rule.Channel,
				"pattern", rule.Pattern,
			)
			telegram.SendToTelegram(fmt.Sprintf("%s\n%s", subject, body), rule.Channel, logger)
			return
		}
	}

	// Если ни одно правило не сработало, отправляем в канал по умолчанию
	logger.Infow("message routed to default channel",
		"channel", cfg.Telegram.DefaultChannel,
	)
	telegram.SendToTelegram(fmt.Sprintf("subject: %s\n%s", subject, body), cfg.Telegram.DefaultChannel, logger)
}
