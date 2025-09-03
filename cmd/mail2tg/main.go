package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/st-kuptsov/mail2tg/config"
	"github.com/st-kuptsov/mail2tg/internal/scheduler"
	"github.com/st-kuptsov/mail2tg/internal/telegram"
	logs "github.com/st-kuptsov/mail2tg/pkg/logs"
	"github.com/st-kuptsov/mail2tg/pkg/metrics"
	tb "gopkg.in/telebot.v3"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var Version = "dev"

// main запускает Mail2TG сервис
func main() {
	start := time.Now()

	configPath := flag.String("config", "config/config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.GetConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	logger := logs.DefaultLogger(cfg.Logging)

	logger.Infow("starting mail2tg",
		"config", *configPath,
		"logLevel", cfg.Logging.Level,
		"pid", os.Getpid(),
		"version", Version,
	)

	// Инициализация метрик
	logger.Debug("initializing metrics server")
	metrics.InitMetrics()

	// Запуск HTTP-сервера для Prometheus в отдельной горутине
	go func() {
		servicePort := fmt.Sprintf(":%d", cfg.ServicePort)

		// Регистрируем handler для /metrics
		http.Handle("/metrics", promhttp.Handler())

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
