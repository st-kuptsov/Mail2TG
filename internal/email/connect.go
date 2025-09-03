package email

import (
	"fmt"
	"github.com/emersion/go-imap/client"
	"github.com/st-kuptsov/mail2tg/config"
	"go.uber.org/zap"
	"time"
)

// ConnectToIMAP подключается к IMAP-серверу по TLS, выполняет авторизацию и возвращает клиент.
// Логирует ключевые шаги и обновляет метрику доступности UpGauge.
func ConnectToIMAP(cfg *config.Config, logger *zap.SugaredLogger) (*client.Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.IMAP.Host, cfg.IMAP.Port)
	logger.Infow("connecting to IMAP server", "address", addr)

	// Подключаемся к серверу
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		logger.Errorw("failed to connect to IMAP server", "error", err)
		return nil, fmt.Errorf("failed to connect to IMAP: %w", err)
	}

	// Устанавливаем таймауты
	c.Timeout = 30 * time.Second
	logger.Info("IMAP connection established")

	// Выполняем логин
	if err := c.Login(cfg.IMAP.Username, cfg.IMAP.Password); err != nil {
		logger.Errorw("IMAP login failed", "error", err)
		return nil, fmt.Errorf("IMAP login failed: %w", err)
	}

	logger.Infow("IMAP login successful", "username", cfg.IMAP.Username)
	return c, nil
}
