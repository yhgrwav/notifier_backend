// Package service реализовывает валидацию, логику, автозаполнение
package service

import (
	"RedCollar/internal/domain"
	"RedCollar/internal/repository"
	"context"
	"errors"
	"fmt"
	"time"
)

type IncidentService struct {
	//Мы как сервис требуем какое-то хранилище, для которого мы будем реализовывать нашу логику
	repo repository.IncidentRepository
}

// Принимаем объект с нужными методами(repository) и возвращаем указатель с которым будем работать
func NewIncidentService(repo repository.IncidentRepository) *IncidentService {
	return &IncidentService{repo: repo}
}

func (s *IncidentService) Create(ctx context.Context, i *domain.Incident) (string, error) {
	//Валидация
	if len(i.Title) < 1 {
		return "", errors.New("Ошибка: пустой заголовок")
	}
	if len(i.Title) > 255 {
		return "", errors.New("Заголовок слишком длинный")
	}
	if i.Latitude < -90 || i.Latitude > 90 {
		return "", errors.New("Невалидная широта")
	}
	if i.Longitude < -180 || i.Longitude > 180 {
		return "", errors.New("Невалидная долгота")
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
		return "", err
	}
	return id, nil
}

// Get отвечает за то, чтобы возвращать список инцидентов, в радиусе которых находится пользователь
func (i *IncidentService) Get(ctx context.Context, lat float64, long float64, limit, offset int) ([]*domain.Incident, error) {
	//Проверка на валидность координат
	if (lat < -90 || lat > 90) || (long < -180 || long > 180) {
		return nil, errors.New("Ошибка: получены невалидные координаты")
	}
	//Валидация пагинации
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	//Если всё ок - вызываем репозиторий
	result, err := i.repo.Get(ctx, lat, long, limit, offset)
	if err != nil {
		return nil, err
	}

	if len(result) < 1 {
		return []*domain.Incident{}, nil
	}
	return result, nil
}

// GetByID отвечает за то, чтобы вернуть какую-то конкретную запись из БД по UUID(ID)
func (i *IncidentService) GetByID(ctx context.Context, id string) (*domain.Incident, error) {
	if len(id) != 36 {
		return nil, errors.New("Невалидная длина ID ")
	}
	result, err := i.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Метод Delete по условию должен деактивировать инцидент (isActive = false)
func (i *IncidentService) Delete(ctx context.Context, id string) error {
	if len(id) != 36 {
		return errors.New("Ошибка: невалидная длина ID")
	}
	err := i.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("Ошибка удаления инцидента: %w", err)
	}
	return nil
}

func (i *IncidentService) Update(ctx context.Context, id string, incident *domain.Incident) error {
	//Явно пробрасываем указанный id как id текущей сущности(структуры)
	incident.ID = id
	if len(id) != 36 {
		return errors.New("Ошибка: невалидный ID ")
	}
	if len(incident.Title) > 255 || len(incident.Title) < 1 {
		return errors.New("Невалидная длина заголовка ")
	}
	if len(incident.Description) > 255 || len(incident.Description) < 1 {
		return errors.New("Невалидная длина описания ")
	}
	if incident.Latitude < -90 || incident.Latitude > 90 {
		return errors.New("Невалидная широта")
	}
	if incident.Longitude < -180 || incident.Longitude > 180 {
		return errors.New("Невалидная долгота")
	}
	return i.repo.Update(ctx, incident)
}
