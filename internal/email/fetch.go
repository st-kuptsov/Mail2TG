package email

import (
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/st-kuptsov/mail2tg/config"
	"github.com/st-kuptsov/mail2tg/pkg/utils"
	"go.uber.org/zap"
	"net/mail"
)

// FetchUnreadEmails получает все новые непрочитанные письма из указанной папки IMAP.
// Возвращает слайс сообщений mail.Message и ошибку при неудаче.
// Логирует все ключевые шаги и обновляет метрики.
func FetchUnreadEmails(cfg *config.Config, f config.Folder, c *client.Client, logger *zap.SugaredLogger) ([]*mail.Message, error) {
	logger.Infow("selecting IMAP folder", "folder", f.Name)

	// Выбираем папку
	mbox, err := c.Select(f.Name, false)
	if err != nil {
		logger.Errorw("failed to select folder", "folder", f.Name, "error", err)
		return nil, fmt.Errorf("failed to select folder: %w", err)
	}

	logger.Infow("folder selected", "folder", f.Name, "messages_total", mbox.Messages)
	if mbox.Messages == 0 {
		return nil, nil
	}

	// Создаем критерий поиска непрочитанных писем
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}

	ids, err := c.Search(criteria)
	if err != nil {
		logger.Errorw("failed to search for unread emails", "folder", f.Name, "error", err)
		return nil, fmt.Errorf("failed to search emails: %w", err)
	}

	logger.Infow("unread emails found", "folder", f.Name, "count", len(ids))
	utils.MailChecks.Inc()

	if len(ids) == 0 {
		return nil, nil
	}

	// Подготавливаем последовательность для выборки
	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	section := &imap.BodySectionName{}
	messagesChan := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	// Получаем письма асинхронно
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{section.FetchItem()}, messagesChan)
	}()

	var result []*mail.Message
	for msg := range messagesChan {
		if msg == nil {
			continue
		}

		r := msg.GetBody(section)
		if r == nil {
			logger.Warn("email body is empty")
			continue
		}

		m, err := mail.ReadMessage(r)
		if err != nil {
			logger.Warnw("failed to read email", "error", err)
			continue
		}

		result = append(result, m)
		utils.MailReceived.Inc()
	}

	// Проверяем ошибки после завершения Fetch
	if err := <-done; err != nil {
		logger.Errorw("failed to fetch emails", "folder", f.Name, "error", err)
		return nil, fmt.Errorf("failed to fetch emails: %w", err)
	}

	logger.Infow("emails processed", "folder", f.Name, "count", len(result))
	return result, nil
}
