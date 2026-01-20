package config

import (
	"errors"
	"fmt"
	"log"
	"net/url"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	AppPort        string  `env:"APP_PORT" envDefault:"8080"`
	PostgresDSN    string  `env:"POSTGRES_DSN,required"`
	RedisAddr      string  `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	WarningZone    float64 `env:"WARNING_ZONE" envDefault:"500.0"`
	StatsTime      int     `env:"STATS_TIME_WINDOW_MINUTES" envDefault:"1"`
	CacheTimeout   int     `env:"CACHE_UPDATE_TIMEOUT" envDefault:"2"`
	CacheTTL       int     `env:"CACHE_TTL" envDefault:"10"`
	WebhookUrl     string  `env:"WEBHOOK_URL" envDefault:"http://localhost/"`
	WebhookRetries int     `env:"WEBHOOK_RETRIES" envDefault:"3"`
	WebhookTimeout int     `env:"WEBHOOK_TIMEOUT" envDefault:"10"`
	ApiKey         string  `env:"API_KEY,required"`
}

func Load() (*Config, error) {
	//подгружаем переменные в environment block
	//если не получилось прочитать - логируем и пробуем получить их из настроек докера
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, чтение системных переменных окружения...")
	}

	//создаём переменную в которую будем записывать тело структуры
	cfg := &Config{}

	//парсим в структуру полученный результат благодаря тегам
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("ошибка парсинга переменных окружения: %w", err)
	}

	//вызываем локальный метод validate, чтобы сразу вернуть проверенные данные
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("ошибка валидации конфига: %w", err)
	}

	return cfg, nil
}

// validate содержит всю необходимую валидацию для переменных .env
func (c *Config) validate() error {
	if len(c.ApiKey) > 20 {
		return errors.New("API_KEY слишком длинный (максимум 20 символов)")
	}

	parsedURL, err := url.Parse(c.WebhookUrl)
	if err != nil {
		return fmt.Errorf("некорректный формат WEBHOOK_URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("WEBHOOK_URL должен использовать протокол http или https")
	}

	if c.StatsTime < 1 || c.StatsTime > 10000 {
		c.StatsTime = 1
		log.Println("Предупреждение: StatsTime вне диапазона, установлено значение 1")
	}

	if c.WarningZone <= 0 {
		return errors.New("WARNING_ZONE должна быть положительным числом")
	}

	return nil
}
