// Package service реализовывает валидацию, логику, автозаполнение
package service

import (
	"RedCollar/internal/domain"
	"RedCollar/internal/repository"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type IncidentService struct {
	//Мы как сервис требуем какое-то хранилище, для которого мы будем реализовывать нашу логику
	repo        repository.IncidentRepository
	warningZone float64
}

// Принимаем объект с нужными методами(repository) и возвращаем указатель с которым будем работать
func NewIncidentService(repo repository.IncidentRepository, warningZone float64) *IncidentService {
	return &IncidentService{repo: repo, warningZone: warningZone}
}

// Create отвечает за создание инцидента, валидацию полей, установку дефолтов
func (s *IncidentService) Create(ctx context.Context, i *domain.Incident) (string, error) {
	//Валидация
	if len(i.Title) < 1 {
		return "", errors.New("ошибка: пустой заголовок")
	}
	if len(i.Title) > 255 {
		return "", errors.New("заголовок слишком длинный (максимум 255 символов)")
	}
	if i.Latitude < -90 || i.Latitude > 90 {
		return "", errors.New("невалидная широта (должна быть в диапазоне от -90 до 90)")
	}
	if i.Longitude < -180 || i.Longitude > 180 {
		return "", errors.New("невалидная долгота (должна быть в диапазоне от -180 до 180)")
	}

	//Если мы не получили радиус, или получили невалидный, то ставим валидный дефолт
	if i.RadiusMeters <= 0 || i.RadiusMeters > 2000 {
		i.RadiusMeters = 200
	}
	i.IsActive = true
	i.CreatedAt = time.Now()
	//Когда у нас готово всё кроме i.ID, мы дёргаем метод репозитория и передаём туда всё необходимое, чтобы создать
	//инцидент и получить uuid который мы и будем возвращать для пользователя/фронта
	id, err := s.repo.Create(ctx, i)
	if err != nil {
		return "", fmt.Errorf("ошибка создания инцидента: %w", err)
	}
	// Конвертируем uuid.UUID в string для возврата
	return id.String(), nil
}

// Get отвечает за то, чтобы возвращать валидный список инцидентов, в радиусе которых находится пользователь
func (i *IncidentService) Get(ctx context.Context, lat float64, long float64, limit, offset int) ([]*domain.Incident, error) {
	//Проверка на валидность координат
	if lat < -90 || lat > 90 {
		return nil, errors.New("невалидная широта (должна быть в диапазоне от -90 до 90)")
	}
	if long < -180 || long > 180 {
		return nil, errors.New("невалидная долгота (должна быть в диапазоне от -180 до 180)")
	}
	//Валидация пагинации
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	//Если всё ок - вызываем репозиторий, передаем warningZone как extraRadius
	result, err := i.repo.Get(ctx, lat, long, limit, offset, i.warningZone)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка инцидентов: %w", err)
	}

	if len(result) < 1 {
		return []*domain.Incident{}, nil
	}
	return result, nil
}

// GetByID отвечает за то, чтобы вернуть какую-то конкретную запись из БД по UUID(ID)
func (i *IncidentService) GetByID(ctx context.Context, id string) (*domain.Incident, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("невалидный ID")
	}
	result, err := i.repo.GetByID(ctx, parsedID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения инцидента по ID: %w", err)
	}
	return result, nil
}

// Метод Delete по условию должен деактивировать инцидент (isActive = false)
func (i *IncidentService) Delete(ctx context.Context, id string) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("невалидный ID")
	}
	err = i.repo.Delete(ctx, parsedID)
	if err != nil {
		return fmt.Errorf("ошибка удаления инцидента: %w", err)
	}
	return nil
}

// Update Отвечает за обновление данных структуры по UUID
func (i *IncidentService) Update(ctx context.Context, id string, incident *domain.Incident) (uuid.UUID, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, errors.New("невалидный ID")
	}

	//Явно пробрасываем указанный id как id текущей сущности(структуры)
	incident.ID = parsedID

	if len(incident.Title) < 1 {
		return uuid.Nil, errors.New("заголовок не может быть пустым")
	}
	if len(incident.Title) > 255 {
		return uuid.Nil, errors.New("заголовок слишком длинный (максимум 255 символов)")
	}
	if len(incident.Description) < 1 {
		return uuid.Nil, errors.New("описание не может быть пустым")
	}
	if len(incident.Description) > 255 {
		return uuid.Nil, errors.New("описание слишком длинное (максимум 255 символов)")
	}
	if incident.Latitude < -90 || incident.Latitude > 90 {
		return uuid.Nil, errors.New("невалидная широта (должна быть в диапазоне от -90 до 90)")
	}
	if incident.Longitude < -180 || incident.Longitude > 180 {
		return uuid.Nil, errors.New("невалидная долгота (должна быть в диапазоне от -180 до 180)")
	}

	err = i.repo.Update(ctx, incident)
	if err != nil {
		return uuid.Nil, fmt.Errorf("ошибка обновления инцидента: %w", err)
	}
	return incident.ID, nil
}
