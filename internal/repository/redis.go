package repository

import (
	"RedCollar/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrCacheMiss будет использоваться в кейсах redis.Nil, чтобы не тянуть зависимости в другие слои программы
var ErrCacheMiss = errors.New("Кэш отстутствует")

type RedisRepository interface {
	SetCache(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	GetCache(ctx context.Context, key string) ([]byte, error)
	Close() error
	WebhookPush(ctx context.Context, webhook domain.Webhook) error
	PopWebhook(ctx context.Context) (domain.Webhook, error)
}
type redisRepository struct {
	rdb *redis.Client
}

func RedisConnection(ctx context.Context, redisAddr string) (RedisRepository, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &redisRepository{rdb: rdb}, nil
}

func (r *redisRepository) Close() error {
	err := r.rdb.Close()
	if err != nil {
		return errors.New("ошибка закрытия подключения")
	}
	return nil
}

// Метод отвечает за добавление объекта в очередь
func (r *redisRepository) WebhookPush(ctx context.Context, webhook domain.Webhook) error {
	// Сериализируем данные в json
	data, err := json.Marshal(webhook)
	if err != nil {
		return err
	}

	//вызываем метод, LPush и сохраняем данные в список webhook_q + распаковываем ошибку
	return r.rdb.LPush(ctx, "webhook_q", data).Err()
}

// Метод получения данных из очереди
func (r *redisRepository) PopWebhook(ctx context.Context) (domain.Webhook, error) {
	//BRPop позволяет в случае отсутствия данных в списке просто ожидать(0 секунд timeout = бесконечно) пока что-нибудь появится
	result, err := r.rdb.BRPop(ctx, 0, "webhook_q").Result()
	if err != nil {
		return domain.Webhook{}, err
	}
	var webhook domain.Webhook
	err = json.Unmarshal([]byte(result[1]), &webhook) //анмаршалим json данные в переменную
	return webhook, nil                               // и возвращаем результат
}

// SetCache - универсальный метод для кэширования пары ключ-значение и в нашем случае мы будем его настраивать на работу
// с необходимыми данными на уровне сервиса
func (r *redisRepository) SetCache(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	val, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, key, val, ttl).Err()
}

// GetCache отвечает за то, чтобы получить кэш по ключу, но без валидаций, анмаршалинга и тд, т.е. зона ответственности
// этого метода - вытянуть какие-то данные в []byte и отдать дальше в сервис, где уже будет производиться всё что нам нужно
func (r *redisRepository) GetCache(ctx context.Context, key string) ([]byte, error) {
	res, err := r.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrCacheMiss
	} else if err != nil {
		return nil, err
	}
	return res, nil
}
