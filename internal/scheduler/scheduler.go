package scheduler

import (
	"context"
	"fmt"
	"github.com/st-kuptsov/mail2tg/config"
	"github.com/st-kuptsov/mail2tg/internal/alerts"
	"github.com/st-kuptsov/mail2tg/internal/email"
	"github.com/st-kuptsov/mail2tg/internal/route"
	"github.com/st-kuptsov/mail2tg/internal/telegram"
	"github.com/st-kuptsov/mail2tg/pkg/metrics"
	"go.uber.org/zap"
	"sync"
	"time"
)

var mu sync.Mutex
var connectionToIMAP alerts.Status
var fetchUnreadEmails alerts.Status

// Scheduler запускает цикл опроса почты по заданному интервалу.
// Работает до отмены контекста.
func Scheduler(ctx context.Context, conf *config.CachedConfig, logger *zap.SugaredLogger, start time.Time, configPath string) {
	ticker := time.NewTicker(time.Duration(conf.Config.CheckInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Infow("scheduler stopped")
			return
		case <-ticker.C:
			metrics.UptimeGauge.Set(time.Since(start).Seconds())

			func() {
				// Отслеживание времени обработки всех писем
				processingStart := time.Now()
				defer func() {
					metrics.MailProcessingDuration.Observe(time.Since(processingStart).Seconds())
				}()

				changed, err := conf.ReloadIfChanged(configPath)
				if err != nil {
					logger.Errorw("reload config error", "error", err)
				}
				if changed {
					logger.Infow("config reloaded due to changes")
				}

				defer func() {
					if r := recover(); r != nil {
						logger.Errorf("panic recovered: %v", r)
						metrics.MailErrors.Inc()
						telegram.SendToTelegram(
							fmt.Sprintf("Паника в обработчике почты: %v", r),
							conf.Config.Telegram.ErrorsChannel,
							logger,
						)
					}
				}()

				// Подключение к IMAP
				c, err := email.ConnectToIMAP(conf.Config, logger)
				mu.Lock()
				alerts.ConnectToIMAPError(err, logger, conf, &connectionToIMAP)
				mu.Unlock()

				if err != nil {
					// не получилось подключиться — дальше смысла идти нет
					return
				}

				defer func() {
					if err := c.Logout(); err != nil {
						logger.Warnf("Ошибка выхода из IMAP: %v", err)
					}
				}()

				// Обходим все папки и правила
				for _, r := range conf.Config.Route {
					for _, f := range r.Folders {
						messages, err := email.FetchUnreadEmails(conf.Config, f, c, logger)
						mu.Lock()
						alerts.FetchUnreadEmailsError(err, logger, conf, &fetchUnreadEmails)
						mu.Unlock()

						for _, m := range messages {
							subject, body := email.DecodeMessage(m, logger)
							route.RouteMessage(conf.Config, f, subject, body, logger)
						}
					}
				}
			}()
		}
	}
}
