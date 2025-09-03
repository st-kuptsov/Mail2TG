package utils

import "github.com/prometheus/client_golang/prometheus"

// =====================
// Метрики состояния сервиса
// =====================

var (
	// UptimeGauge отслеживает время работы сервиса в секундах
	UptimeGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "mail2tg_uptime",
			Help: "Service uptime in seconds",
		},
	)
)

// =====================
// Метрики почты
// =====================

var (
	// MailChecks - количество успешных проверок почтового ящика
	MailChecks = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mail2tg_mailbox_successful_checks_total",
			Help: "Count of successful mailbox checks",
		},
	)

	// MailReceived - количество полученных писем
	MailReceived = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mail2tg_mailbox_received_messages_total",
			Help: "Number of received emails",
		},
	)

	// MailErrors - количество ошибок при проверке почты
	MailErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mail2tg_mailbox_errors_total",
			Help: "Number of errors while checking mailboxes",
		},
	)

	// MailProcessingDuration - время обработки почты в секундах
	MailProcessingDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "mail2tg_mail_processing_duration_seconds",
			Help:    "Time spent processing mails",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // от 0.1s до ~50s
		},
	)
)

// =====================
// Метрики Telegram
// =====================

var (
	// TgMessagesSent - количество сообщений, отправленных в Telegram, с разбиением по channel_id
	TgMessagesSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mail2tg_telegram_messages_sent_total",
			Help: "Messages sent to Telegram",
		},
		[]string{"channel_id"},
	)

	// TgErrors - количество ошибок при отправке сообщений в Telegram
	TgErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mail2tg_telegram_messages_errors_total",
			Help: "Failed messages to Telegram",
		},
		[]string{"channel_id"},
	)

	// TgSendDuration - время отправки сообщений в Telegram
	TgSendDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mail2tg_telegram_send_duration_seconds",
			Help:    "Time spent sending messages to Telegram",
			Buckets: prometheus.ExponentialBuckets(0.05, 2, 12), // от 50ms до ~1 мин
		},
		[]string{"channel_id"},
	)
)

// InitMetrics регистрирует все метрики Prometheus
func InitMetrics() {
	prometheus.MustRegister(
		UptimeGauge,
		MailChecks,
		MailReceived,
		MailErrors,
		MailProcessingDuration,
		TgMessagesSent,
		TgErrors,
		TgSendDuration,
	)
}
