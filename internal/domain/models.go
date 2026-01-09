package domain

import (
	"time"

	"github.com/google/uuid"
)

type Incident struct { // Тело инцидента
	ID           uuid.UUID `json:"id"`            //UUID
	Title        string    `json:"title"`         //Заголовок устанавливаемый оператором (например "Пожар")
	Description  string    `json:"description"`   //Описание инцидента (например "Огонь разрастается в Южную сторону")
	Latitude     float64   `json:"latitude"`      //Широта
	Longitude    float64   `json:"longitude"`     //Долгота
	RadiusMeters float64   `json:"radius_meters"` //Радиус опасной зоны
	IsActive     bool      `json:"is_active"`     //Активен ли инцидент
	CreatedAt    time.Time `json:"created_at"`    //Время инициализации инцидента (по условию нужно вернуть user_count за N минут)
}

type LocationCheckRequest struct { // Структура запроса геоданных пользователя которую мы будем валидировать (т.е. то, что мы просим у пользователя)
	UserID    string  `json:"user_id"`   //UUID
	Latitude  float64 `json:"latitude"`  ////Геоданные из которых мы складываем координаты
	Longitude float64 `json:"longitude"` ////И ещё
}
type LocationCheckResponse struct { //То что пользователь получает
	IsInDanger bool        `json:"is_in_danger"` //Опасно ли (находится ли в данный момент пользователь в радиусе активного инцидента)
	Incidents  []*Incident `json:"incidents"`    //Список указателей на структуру инцидента
}

type StatisticResponse struct { //Структура ответа для эндпоинта /api/v1/incidents/stats (по условию задачи)
	IncidentID string `json:"incident_id"` //UUID
	UserCount  int    `json:"user_count"`  //Количество пользователей попавших в радиус инцидента пока инцидент был IsActive true
}
