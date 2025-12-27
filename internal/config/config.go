package config

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort     string
	PostgresDSN string
	RedisAddr   string
}

func GetEnv() (*Config, error) {
	//1. С помощью godotenv.Load() записываем содержимое файла .env в память приложения
	//Если не удалось прочитать через .env, читаем из enviroment в docker-compose
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, производится попытка получить системные переменные...")
	}

	//2. Явно записываем необходимые данные в переменные
	App_port := os.Getenv("APP_PORT")
	postgres := os.Getenv("POSTGRES_DSN")
	RedisAddr := os.Getenv("REDIS_ADDR")

	//3. Валидируем полученные данные
	if postgres == "" {
		return nil, errors.New("Ошибка: не указан адрес подключения к базе данных")
	}
	if App_port == "" {
		App_port = "8080"
	}
	if RedisAddr == "" {
		RedisAddr = "localhost:6379"
	}
	//4. Возвращаем указатель на структуру
	return &Config{
		AppPort:     App_port,
		PostgresDSN: postgres,
		RedisAddr:   RedisAddr,
	}, nil
}
