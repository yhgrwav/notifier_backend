package worker

import (
	"RedCollar/internal/domain"
	"RedCollar/internal/repository"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type WebhookWorker struct {
	redisRepo     repository.RedisRepository
	client        *http.Client
	URL           string
	retriesAmount int
}

func NewWebhookWorker(redisRepo repository.RedisRepository, client *http.Client, url string, retries int) *WebhookWorker {
	return &WebhookWorker{
		redisRepo:     redisRepo,
		client:        client,
		URL:           url,
		retriesAmount: retries,
	}
}
func (w *WebhookWorker) Run(ctx context.Context) {
	log.Println("Webhook worker успешно запущен")
	for {
		select {
		case <-ctx.Done(): // Когда получаем сигнал от graceful shutdown - даём время завершиться всем запущенным воркерам
			log.Println("Вебхук воркер завершается...")
			return
		default:
			webhook, err := w.redisRepo.PopWebhook(ctx)
			if err != nil {
				log.Printf("Ошибка:%v", err)
				continue
			}
			err = w.SendWithRetry(webhook, w.retriesAmount)
			if err != nil {
				log.Printf("Ошибка:%v", err)
				continue
			}
		}

	}
}

// Я разделил логику отправки и ретраев на две функции
// SendNotification отвечает исключительно за отправку уведомления
func (w *WebhookWorker) SendNotification(webhook domain.Webhook) error {
	body, err := json.Marshal(webhook)
	if err != nil {
		return err
	}
	//Постим тело вебхука
	resp, err := w.client.Post(w.URL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	//если получаем неудовлетворительный статускод = метод sendWithRetry понимает что получил ошибку и выполняет логику ретрая
	if resp.StatusCode >= 400 {
		return fmt.Errorf("неудовлетворительный ответ: %v", resp.StatusCode)
	}
	return nil
}

// SendWithRetry отвечает за вызов SendNotification с n ретраями
func (w *WebhookWorker) SendWithRetry(webhook domain.Webhook, retries int) error {
	for i := 1; i <= retries; i++ {
		err := w.SendNotification(webhook)
		if err == nil { //если после отправки мы не получаем ошибку = ретрай был успешный и мы возвращаем nil
			return nil
		}
		//каждую итерацию ретраев мы ждём на секунду больше, чтобы увеличить шансы на успешный кейс отправки вебхука
		if i < retries {
			time.Sleep(time.Duration(i) * time.Second)
		}
	}
	return fmt.Errorf("спустя %v ретраев не удалось отправить вебхук %v", retries, webhook.IncidentID)
}
