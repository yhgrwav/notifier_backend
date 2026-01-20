package main

import (
	"RedCollar/internal/worker"
	"context"
	"log"
	"os/signal"
	"syscall"

	"RedCollar/internal/config"
	v1 "RedCollar/internal/delivery/http/v1"
	"RedCollar/internal/repository"
	"RedCollar/internal/service"
)

func main() {
	//создаём контекст для управления программой
	//при нажатии ctrl+c отправляем сигнал и отключаемся
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	//читаем и записываем переменные из .env в переменную cfg
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("не удалось получить данные из .env: %v", err)
	}

	//устанавливаем подключение с PostgreSQL
	db, err := repository.NewPostgresConnection(ctx, cfg)
	if err != nil {
		log.Fatal("не удалось подключиться к БД(postgres):", err)
	}

	//устанавливаем подключение с Redis
	rdb, err := repository.RedisConnection(ctx, cfg.RedisAddr)
	if err != nil {
		log.Fatal("не удалось подключиться к БД(redis):", err)
	}

	//инициализируем сервис
	serv := service.NewIncidentService(db, rdb, cfg.WarningZone, cfg.CacheTTL)

	//инициализируем HTTP клиента
	client := service.NewHTTPClient(cfg.WebhookTimeout)

	//инициализируем воркера
	w := worker.NewWebhookWorker(rdb, client, cfg.WebhookUrl, cfg.WebhookRetries)

	//запускаем в воркера в горутине чтобы не блокировать основное выполнение программы
	go func() {
		w.Run(ctx)
	}()

	//инициализируем сервер
	h := v1.NewHandler(serv, cfg.StatsTime)

	log.Printf("сервер запущен на порту: %s", cfg.AppPort)

	//запускаем сервер
	if err := h.Run(cfg.AppPort, cfg.ApiKey); err != nil {
		log.Fatal("ошибка запуска сервера:", err)
	}
}
