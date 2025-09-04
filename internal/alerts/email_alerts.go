package alerts

import (
	"fmt"
	"github.com/st-kuptsov/mail2tg/config"
	"github.com/st-kuptsov/mail2tg/internal/telegram"
	"github.com/st-kuptsov/mail2tg/pkg/metrics"
	"go.uber.org/zap"
	"time"
)

type Status struct {
	lastSuccess time.Time
	healthy     bool
	alertSent   bool
	initialized bool
}

func ConnectToIMAPError(err error, logger *zap.SugaredLogger, conf *config.CachedConfig, status *Status) {
	if status.lastSuccess.IsZero() {
		status.lastSuccess = time.Now()
	}
	if err != nil {
		logger.Errorf("IMAP connection error: %v", err)
		metrics.MailErrors.Inc()
		status.healthy = false
		if time.Since(status.lastSuccess) > time.Duration(conf.Config.Alerting.AlertEmailDelay)*time.Second && !status.alertSent {
			telegram.SendToTelegram(fmt.Sprintf("Ошибка подключения: %v. Последняя успешная проверка в %v", err, status.lastSuccess.Format("2006-01-02 15:04:05")),
				conf.Config.Telegram.ErrorsChannel, logger)
			status.alertSent = true
			status.initialized = true
		}
		return
	}

	status.lastSuccess = time.Now()
	if !status.healthy && status.initialized {
		telegram.SendToTelegram(fmt.Sprintf("Подключение восстановлено в %v", status.lastSuccess.Format("2006-01-02 15:04:05")),
			conf.Config.Telegram.ErrorsChannel, logger)
		status.healthy = true
		status.alertSent = false
		status.initialized = true
	}
}

func FetchUnreadEmailsError(err error, logger *zap.SugaredLogger, conf *config.CachedConfig, status *Status) {
	if status.lastSuccess.IsZero() {
		status.lastSuccess = time.Now()
	}
	if err != nil {
		logger.Errorf("fetch unread emails error: %v", err)
		metrics.MailErrors.Inc()
		status.healthy = false
		if time.Since(status.lastSuccess) > time.Duration(conf.Config.Alerting.AlertEmailDelay)*time.Second && !status.alertSent {
			telegram.SendToTelegram(fmt.Sprintf("Ошибка получения писем: %v. Последняя успешная проверка в %v", err, status.lastSuccess.Format("2006-01-02 15:04:05")),
				conf.Config.Telegram.ErrorsChannel, logger)
			status.alertSent = true
			status.initialized = true
		}
		return
	}

	status.lastSuccess = time.Now()
	if !status.healthy && status.initialized {
		telegram.SendToTelegram(fmt.Sprintf("Получение писем восстановлено в %v", status.lastSuccess.Format("2006-01-02 15:04:05")),
			conf.Config.Telegram.ErrorsChannel, logger)
		status.healthy = true
		status.alertSent = false
		status.initialized = true
	}
}
