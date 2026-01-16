package main

import (
	"context"
	"log"

	"RedCollar/internal/config"
	v1 "RedCollar/internal/delivery/http/v1"
	"RedCollar/internal/repository"
	"RedCollar/internal/service"
)

func main() {
	ctx := context.Background()
	cfg, err := config.GetEnv()
	if err != nil {
		log.Fatalf("не удалось получить данные из .env: %v", err)
	}

	db, err := repository.NewPostgresConnection(ctx, cfg)
	if err != nil {
		log.Fatal("не удалось подключиться к БД(postgres):", err)
	}

	rdb, err := repository.RedisConnection(ctx, cfg.RedisAddr)
	if err != nil {
		log.Fatal("не удалось подключиться к БД(redis):", err)
	}

	serv := service.NewIncidentService(db, rdb, cfg.WarningZone, cfg.CacheTTL)

	client := service.NewHTTPClient(cfg.WebhookTimeout)

	go func() {
		log.Printf("вебхук воркер запущен по адресу %s", cfg.WebhookUrl)
		worker.NewWebhookWorker(rdb, client, cfg.WebhookUrl, cfg.WebhookRetries)
	}()

	h := v1.NewHandler(serv, cfg.StatsTime)

	log.Printf("сервер запущен на порту: %s", cfg.AppPort)
	if err := h.Run(cfg.AppPort); err != nil {
		log.Fatal("ошибка запуска сервера:", err)
	}
}
