package email

import (
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
)

// DecodeMessage декодирует заголовки и тело письма.
// Возвращает subject и body письма как строки.
// Логирует все предупреждения и ошибки при декодировании.
func DecodeMessage(msg *mail.Message, logger *zap.SugaredLogger) (string, string) {
	// Декодируем тему письма
	subject, err := decodeHeader(msg.Header.Get("Subject"))
	if err != nil {
		subject = msg.Header.Get("Subject")
		logger.Warnw("failed to decode email subject", "error", err)
	}

	contentType := msg.Header.Get("Content-Type")
	mediatype, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		logger.Warnw("failed to parse content-type", "error", err)
		return subject, "error: cannot parse content-type"
	}

	// Обработка multipart сообщений
	if strings.HasPrefix(mediatype, "multipart/") {
		boundary, ok := params["boundary"]
		if !ok {
			logger.Warn("no boundary found in multipart message")
			return subject, "error: no boundary in multipart message"
		}

		mr := multipart.NewReader(msg.Body, boundary)
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				logger.Warnw("failed to read multipart part", "error", err)
				continue
			}

			partType, partParams, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
			body, err := decodePart(part, part.Header.Get("Content-Transfer-Encoding"), partParams["charset"], logger)
			if err != nil {
				logger.Warnw("failed to decode email part", "error", err)
				continue
			}

			switch partType {
			case "text/html":
				return subject, strings.TrimSpace(html.UnescapeString(htmlToText(body)))
			case "text/plain":
				return subject, strings.TrimSpace(html.UnescapeString(body))
			}
		}
		return subject, "no suitable part found"
	}

	// Одночастное сообщение
	body, err := decodePart(msg.Body, msg.Header.Get("Content-Transfer-Encoding"), params["charset"], logger)
	if err != nil {
		logger.Warnw("failed to read email body", "error", err)
		return subject, "error reading body"
	}

	if mediatype == "text/html" {
		body = htmlToText(body)
	}
	return subject, strings.TrimSpace(html.UnescapeString(body))
}

// decodePart декодирует отдельную часть письма с учётом кодировки и charset.
func decodePart(reader io.Reader, encoding string, charsetStr string, logger *zap.SugaredLogger) (string, error) {
	r := reader
	if strings.ToLower(encoding) == "quoted-printable" {
		r = quotedprintable.NewReader(r)
	}

	if charsetStr != "" && strings.ToLower(charsetStr) != "utf-8" {
		enc, _ := charset.Lookup(strings.ToLower(charsetStr))
		if enc != nil {
			r = transform.NewReader(r, enc.NewDecoder())
		} else {
			logger.Warnw("unknown charset, skipping decoding", "charset", charsetStr)
		}
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read data: %w", err)
	}
	return string(data), nil
}

// htmlToText конвертирует HTML в plain text, удаляя теги.
func htmlToText(input string) string {
	tokenizer := html.NewTokenizer(strings.NewReader(input))
	var lines []string
	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return strings.Join(lines, "\n")
		case html.TextToken:
			text := strings.TrimSpace(string(tokenizer.Text()))
			if text != "" {
				lines = append(lines, text)
			}
		}
	}
}

// decodeHeader декодирует MIME-заголовок с учётом кодировки.
func decodeHeader(hdr string) (string, error) {
	dec := new(mime.WordDecoder)
	dec.CharsetReader = func(charsetName string, input io.Reader) (io.Reader, error) {
		e, _ := charset.Lookup(strings.ToLower(charsetName))
		if e != nil {
			return transform.NewReader(input, e.NewDecoder()), nil
		}
		return input, nil
	}
	return dec.DecodeHeader(hdr)
}
