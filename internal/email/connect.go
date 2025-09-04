package email

import (
	"crypto/tls"
	"fmt"
	"github.com/emersion/go-imap/client"
	"github.com/st-kuptsov/mail2tg/config"
	"go.uber.org/zap"
	"net"
	"time"
)

// ConnectToIMAP подключается к IMAP-серверу по TLS с таймаутами и возвращает клиента
func ConnectToIMAP(cfg *config.Config, logger *zap.SugaredLogger) (*client.Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.IMAP.Host, cfg.IMAP.Port)
	logger.Infow("connecting to IMAP server", "address", addr)

	// создаем Dialer с таймаутом
	dialer := &net.Dialer{
		Timeout: 10 * time.Second, // время на установку соединения
	}

	// создаем TLS-коннект с таймаутом
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{})
	if err != nil {
		logger.Errorw("failed to connect to IMAP server", "error", err)
		return nil, fmt.Errorf("failed to connect to IMAP: %w", err)
	}

	// создаем IMAP-клиент поверх уже установленного соединения
	c, err := client.New(conn)
	if err != nil {
		logger.Errorw("failed to create IMAP client", "error", err)
		return nil, fmt.Errorf("failed to create IMAP client: %w", err)
	}

	logger.Info("IMAP connection established")

	if err := c.Login(cfg.IMAP.Username, cfg.IMAP.Password); err != nil {
		logger.Errorw("IMAP login failed", "error", err)
		return nil, fmt.Errorf("IMAP login failed: %w", err)
	}

	logger.Infow("IMAP login successful", "username", cfg.IMAP.Username)
	return c, nil
}
