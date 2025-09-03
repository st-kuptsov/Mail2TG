package scheduler

import (
	"context"
	"fmt"
	"github.com/st-kuptsov/mail2tg/config"
	"github.com/st-kuptsov/mail2tg/internal/email"
	"github.com/st-kuptsov/mail2tg/internal/route"
	"github.com/st-kuptsov/mail2tg/internal/telegram"
	"github.com/st-kuptsov/mail2tg/pkg/utils"
	"go.uber.org/zap"
	"time"
)

// Scheduler запускает цикл опроса почты по заданному интервалу.
// Работает до отмены контекста.
func Scheduler(ctx context.Context, cfg *config.Config, logger *zap.SugaredLogger, start time.Time) {
	ticker := time.NewTicker(time.Duration(cfg.CheckInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Infow("scheduler stopped")
			return
		case <-ticker.C:
			utils.UptimeGauge.Set(time.Since(start).Seconds())

			func() {
				// Отслеживание времени обработки всех писем
				processingStart := time.Now()
				defer func() {
					utils.MailProcessingDuration.Observe(time.Since(processingStart).Seconds())
				}()

				defer func() {
					if r := recover(); r != nil {
						logger.Errorf("panic recovered: %v", r)
						utils.MailErrors.Inc()
						telegram.SendToTelegram(
							fmt.Sprintf("Паника в обработчике почты: %v", r),
							cfg.Telegram.ErrorsChannel,
							logger,
						)
					}
				}()

				// Подключение к IMAP
				c, err := email.ConnectToIMAP(cfg, logger)
				if err != nil {
					logger.Errorf("IMAP connection error: %v", err)
					utils.MailErrors.Inc()
					telegram.SendToTelegram("Ошибка подключения к почте: "+err.Error(),
						cfg.Telegram.ErrorsChannel, logger)
					return
				}
				defer func() {
					if err := c.Logout(); err != nil {
						logger.Warnf("Ошибка выхода из IMAP: %v", err)
					}
				}()

				// Обходим все папки и правила
				for _, r := range cfg.Route {
					for _, f := range r.Folders {
						messages, err := email.FetchUnreadEmails(cfg, f, c, logger)
						if err != nil {
							logger.Errorf("fetch unread emails error: %v", err)
							utils.MailErrors.Inc()
							telegram.SendToTelegram("Ошибка получения писем: "+err.Error(),
								cfg.Telegram.ErrorsChannel, logger)
							continue
						}

						for _, m := range messages {
							subject, body := email.DecodeMessage(m, logger)
							route.RouteMessage(cfg, f, subject, body, logger)
						}
					}
				}
			}()
		}
	}
}
