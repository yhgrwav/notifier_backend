package config

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort        string
	PostgresDSN    string
	RedisAddr      string
	WarningZone    float64
	CacheTimeout   int
	StatsTime      int
	CacheTTL       int
	WebhookUrl     string
	WebhookRetries int
	WebhookTimeout int
}

func GetEnv() (*Config, error) {
	//1. С помощью godotenv.Load() записываем содержимое файла .env в память приложения
	//Если не удалось прочитать через .env, читаем из environment в docker-compose
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, производится попытка получить системные переменные...")
	}

	//2. Явно записываем необходимые данные в переменные
	AppPort := os.Getenv("AppPort")

	postgres := os.Getenv("PostgresDSN")

	RedisAddr := os.Getenv("RedisAddr")

	radiusStr := os.Getenv("warningZone")

	cacheTimeout := os.Getenv("CacheTimeout")
	CacheUpdateTimeout, _ := strconv.Atoi(cacheTimeout)

	statsTimeN := os.Getenv("STATS_TIME_WINDOW_MINUTES")
	StatsTime, _ := strconv.Atoi(statsTimeN)

	cacheTimeToLive := os.Getenv("cacheTTL")
	CacheTTl, _ := strconv.Atoi(cacheTimeToLive)

	WebhookUrl := os.Getenv("WEBHOOK_URL")

	retry := os.Getenv("webhook_retries")
	retries, _ := strconv.Atoi(retry)

	webhooktimeout := os.Getenv("webhook_Timeout")
	whto, _ := strconv.Atoi(webhooktimeout)

	//3. Валидируем полученные данные
	if postgres == "" {
		return nil, errors.New("Ошибка: не указан адрес подключения к базе данных")
	}
	if AppPort == "" {
		AppPort = "8080"
	}
	if RedisAddr == "" {
		RedisAddr = "localhost:6379"
	}
	radius, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil {
		radius = 500.0 // Дефолтное значение
	}
	if radiusStr == "" {
		log.Println("Ошибка: переменная радиуса поиска инцидентов не указана, используем 500.0")
		radius = 500.0
	}
	if CacheUpdateTimeout > 10 || CacheUpdateTimeout < 1 {
		CacheUpdateTimeout = 2
	}
	if StatsTime < 1 {
		StatsTime = 1
	} else if StatsTime > 10000 {
		StatsTime = 10000
	}
	if CacheTTl > 100 || CacheTTl < 1 {
		CacheTTl = 10
	}

	parsedURL, err := url.Parse(WebhookUrl)
	if err != nil {
		return nil, fmt.Errorf("невалидный формат для webhook_url:%w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("webhook_url должен иметь http или https схему")
	}
	if len(WebhookUrl) < 1 {
		//подставляем просто локалхост, но с выводом сообщений на случай, если вдруг юзер действительно ошибся с вводом
		//и думает что использует валидный url
		WebhookUrl = "http://localhost/"
		log.Printf("получен невалидный webhook_url, используется дефолт(%s)", WebhookUrl)
	}

	if retries > 50 || retries < 0 {
		retries = 3
	}

	if whto > 100 || whto < 0 {
		whto = 10
	}

	//4. Возвращаем указатель на структуру
	return &Config{
		AppPort:        AppPort,            //порт приложения
		PostgresDSN:    postgres,           //строка подключения к postgres
		RedisAddr:      RedisAddr,          //адрес подключения к redis
		WarningZone:    radius,             //радиус, в котором мы ищем опасности относительно точки пользователя
		CacheTimeout:   CacheUpdateTimeout, //максимальное время ожидания ответа от redis (секунд)
		StatsTime:      StatsTime,          //отвечает за то, за сколько минут мы будем собирать статистику
		CacheTTL:       CacheTTl,           //время жизни кэша в минутах
		WebhookUrl:     WebhookUrl,         //ссылка на http-сервер-заглушку
		WebhookRetries: retries,            //количество попыток отправить вебхук
		WebhookTimeout: whto,
	}, nil
}
