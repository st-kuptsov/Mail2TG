package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/st-kuptsov/mail2tg/config"
	"github.com/st-kuptsov/mail2tg/internal/scheduler"
	"github.com/st-kuptsov/mail2tg/internal/telegram"
	"github.com/st-kuptsov/mail2tg/pkg/utils"
	tb "gopkg.in/telebot.v3"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// main запускает Mail2TG сервис
func main() {
	version := "dev"
	start := time.Now()

	configPath := flag.String("config", "config/config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.GetConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	logger := utils.DefaultLogger(cfg.Logging)

	logger.Infow("starting mail2tg",
		"config", *configPath,
		"logLevel", cfg.Logging.Level,
		"pid", os.Getpid(),
		"version", version,
	)

	// Инициализация метрик
	logger.Debug("initializing metrics server")
	utils.InitMetrics()

	// Запуск HTTP-сервера для Prometheus в отдельной горутине
	go func() {
		servicePort := fmt.Sprintf(":%d", cfg.ServicePort)
		logger.Infow("metrics server started", "port", servicePort)
		if err := http.ListenAndServe(servicePort, nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorw("metrics server failed", "error", err)
		}
	}()

	// Инициализация Telegram-бота
	logger.Debug("initializing telegram bot")
	pref := tb.Settings{
		Token:  cfg.Telegram.Token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	}

	telegram.Bot, err = tb.NewBot(pref)
	if err != nil {
		logger.Errorw("telegram bot initialization failed", "error", err)
		os.Exit(1)
	}
	logger.Info("telegram bot initialized")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Debug("starting scheduler")
	go scheduler.Scheduler(ctx, cfg, logger, start)

	// Ожидание сигнала остановки
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down gracefully...")
	cancel()
	time.Sleep(2 * time.Second)
}
