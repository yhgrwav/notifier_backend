package worker

import (
	"RedCollar/internal/domain"
	"RedCollar/internal/repository"
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
		//пытаемся получить вебхук из очереди
		webhook, err := w.redisRepo.PopWebhook(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) { //если получили context.Canceled - выходим
				return
			}
			log.Printf("Ошибка получения данных: %v\n", err) //если ошибка не связана с контекстом - логируем и делаем ретрай
			continue
		}
		//при получении ошибки вызываем обёртку с ретраем, передаем кол-во из .env и делаем проверку на context.Canceled
		if err := w.SendWithRetry(ctx, webhook, w.retriesAmount); err != nil {
			if errors.Is(err, context.Canceled) { //получили context.Canceled - выходим
				return
			}
		}
	}
}

// SendNotification отвечает за процесс отправки вебхука
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
func (w *WebhookWorker) SendWithRetry(ctx context.Context, webhook domain.Webhook, retries int) error {
	for i := 1; i <= retries; i++ { //в цикле пытаемся отправить вебхук
		err := w.SendNotification(webhook)
		if err == nil {
			return nil //игнорируем ошибку, чтобы попасть в нижний блок с реализацией ретраев
		}

		//попали в блок реализации ретраев и делаем проверку на какой итерации мы сейчас находимся (можно ли выполняться блоку ниже или нет)
		//если нет - выполняем return, если можно - подставляем i в таймер
		if i < retries {
			timer := time.NewTimer(time.Duration(i) * time.Second)

			select {
			//здесь мы учитываем случай когда мы получили ctx.Done(), т.е. случай, когда главный контекст сказал выключаться
			//мы останавливаем таймер и отдаём ctx.Err() в качестве логов
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
				//если таймер дотикал - выходим из селекта и снова делаем проверку, если проверка успешная - повторяем
				continue
			}
		}
	}
	return fmt.Errorf("спустя %v ретраев не удалось отправить вебхук %v", retries, webhook.IncidentID)
}
