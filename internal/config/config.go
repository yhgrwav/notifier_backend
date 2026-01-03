package config

import (
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort     string
	PostgresDSN string
	RedisAddr   string
	WarningZone float64
}

func GetEnv() (*Config, error) {
	//1. С помощью godotenv.Load() записываем содержимое файла .env в память приложения
	//Если не удалось прочитать через .env, читаем из enviroment в docker-compose
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, производится попытка получить системные переменные...")
	}

	//2. Явно записываем необходимые данные в переменные
	AppPort := os.Getenv("AppPort")
	postgres := os.Getenv("PostgresDSN")
	RedisAddr := os.Getenv("RedisAddr")
	radiusStr := os.Getenv("warningZone")
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
	//4. Возвращаем указатель на структуру
	return &Config{
		AppPort:     AppPort,
		PostgresDSN: postgres,
		RedisAddr:   RedisAddr,
		WarningZone: radius,
	}, nil
}
