// Package service реализовывает валидацию, логику, автозаполнение
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"RedCollar/internal/domain"
	"RedCollar/internal/repository"

	"github.com/google/uuid"
)

// HTTPClient должен уметь делать post
type HTTPClient interface {
	Post(url, contentType string, body io.Reader) (*http.Response, error)
}

func NewHTTPClient(timeout int) *http.Client {
	return &http.Client{
		Timeout: time.Duration(timeout) * time.Second, // ожидаем таймаут в int, переводим уже как надо внутри вызова
	}
}

type IncidentService struct {
	//Мы как сервис требуем какое-то хранилище, для которого мы будем реализовывать нашу логику
	repo        repository.IncidentRepository
	rdb         repository.RedisRepository
	warningZone float64
	CacheTTL    int
}

// Принимаем объект с нужными методами(repository) и возвращаем указатель с которым будем работать
func NewIncidentService(repo repository.IncidentRepository, rdb repository.RedisRepository, warningZone float64, CacheTTL int) *IncidentService {
	return &IncidentService{repo: repo, rdb: rdb, warningZone: warningZone, CacheTTL: CacheTTL}
}

// ValidateCoordinates отвечает за валидацию координат и решает проблему дублирования кода
func ValidateCoordinates(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return errors.New("невалидная широта (должна быть в диапазоне от -90 до 90)")
	}
	if lng < -180 || lng > 180 {
		return errors.New("невалидная долгота (должна быть в диапазоне от -180 до 180")
	}
	if lng == 0.0 || lat == 0.0 {
		return errors.New("не указаны координаты")
	}
	return nil
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
	i.ID = uuid.New()
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

// Get отвечает за то, чтобы возвращать валидный список инцидентов в радиусе(warningZone из .env)
// этот метод универсален и для пользователя и для оператора, а также не требует пересборки проекта ради изменения радиуса
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

// CheckLocation Принимает структуру запроса и отдаёт структуру ответа, которые описаны в /domain/models.go
func (i *IncidentService) CheckLocation(ctx context.Context, request domain.LocationCheckRequest, limit, offset int) (domain.LocationCheckResponse, error) {
	//Проверка на валидность координат
	if request.Latitude < -90 || request.Latitude > 90 {
		return domain.LocationCheckResponse{}, errors.New("невалидная широта (должна быть в диапазоне от -90 до 90)")
	}
	if request.Longitude < -180 || request.Longitude > 180 {
		return domain.LocationCheckResponse{}, errors.New("невалидная долгота (должна быть в диапазоне от -180 до 180)")
	}

	//создаем переменную для хранения инцидентов
	var incidents []*domain.Incident
	//делаем из координат запроса ключ
	key := fmt.Sprintf("inc:%.2f:%.2f", request.Latitude, request.Longitude)

	//Проверяем есть ли по нашим координатам инцидент в кэше, чтобы не нагружать лишний раз базу
	cacheResult, err := i.GetIncidentCache(ctx, key)
	if err == nil && cacheResult != nil { // если нет ошибки и есть результат - отдаём результат и выходим
		return domain.LocationCheckResponse{
			IsInDanger: len(cacheResult) > 0,
			Incidents:  cacheResult,
		}, nil
	}

	//если не случился return на этапе проверки кэша - идём в базу с координатами пользователя и ищем инциденты там
	incidents, err = i.repo.Get(ctx, request.Latitude, request.Longitude, limit, offset, i.warningZone)
	if err != nil {
		return domain.LocationCheckResponse{}, fmt.Errorf("ошибка получения данных:%w", err)
	}

	//создаём массив с len(incidents), т.к. это более быстрое решение чем конструкция слайс+append
	incidentIDs := make([]uuid.UUID, len(incidents))

	//если инциденты не пустые - кэшируем их по TTL из конфига
	if len(incidents) > 0 {
		ttl := time.Duration(i.CacheTTL) * time.Minute //оборачиваем переменную из конфига, прошедшую валидацию в time.Minute
		_ = i.CacheIncidents(ctx, request.Latitude, request.Longitude, incidents, ttl)
	}

	//за один последовательный цикл мы и записываем в слайс айди всех инцидентов и вызываем метод WebhookPush()
	for inc := range incidents { //перебираем инциденты
		//копируем id из incidents в incidentIDs
		incidentIDs[inc] = incidents[inc].ID

		//пушим вебхук в очередь
		_ = i.rdb.WebhookPush(ctx, domain.Webhook{
			UserID:     request.UserID,
			IncidentID: incidents[inc].ID,
			DetectedAt: time.Now(),
		})
	}
	//соответственно если инциденты найдены и выполнилась главная бизнес-логика - мы вызываем SaveCheck()
	//и сохраняем факт проверки в БД
	err = i.repo.SaveCheck(ctx, request.UserID, request.Latitude, request.Longitude, incidentIDs)
	if err != nil {
		return domain.LocationCheckResponse{}, errors.New("ошибка сохранения данных")
	}
	return domain.LocationCheckResponse{
		IsInDanger: len(incidents) > 0,
		Incidents:  incidents,
	}, nil
}

// По условию задачи мы должны при запросе статистики читать переменную из .env и отдавать статистику за N минут
// В сервис слое мы читаем переменную, обрабатываем невалидные кейсы и вызываем метод репозитория
func (i *IncidentService) GetStats(ctx context.Context, STATS_TIME_WINDOW_MINUTES int) ([]domain.StatisticResponse, error) {
	//timeInt = Stats_time_window_minutes из .env по условию
	timeInt := STATS_TIME_WINDOW_MINUTES

	//Вызываем сервис
	result, err := i.repo.GetStats(ctx, timeInt)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ключ и ttl мы получаем сверху(из запроса пользователя, т.к. ключ - обрезанные координаты, а ttl мы передаем из .env)
// внутри метода мы должны установить валидацию данных, в нашем случае координаты для ключа, и валидные параметры инцидента
func (i *IncidentService) CacheIncidents(ctx context.Context, lat, lon float64, incidents []*domain.Incident, ttl time.Duration) error {
	//создаём ключ в формате "inc:12:34", где 12 - lat, а 34 - lon
	key := fmt.Sprintf("inc:%.2f:%.2f", lat, lon)

	//полученный массив инцидентов хэшируем
	data, err := json.Marshal(incidents)
	if err != nil {
		return err
	}

	//сгенерированный ключ и хэш данные используем как аргументы и возвращаем вызов метода хэширования redis
	return i.rdb.SetCache(ctx, key, data, ttl)
}

// Метод  должен принимать ключ(примерные координаты) и отдавать слайс инцидентов или фолбэк на метод базы с более долгим запросом
func (i *IncidentService) GetIncidentCache(ctx context.Context, key string) ([]*domain.Incident, error) {
	result, err := i.rdb.GetCache(ctx, key)
	if errors.Is(err, repository.ErrCacheMiss) { //Обрабатываем кейс когда в хранилище кэша пусто благодаря кастомной
		return nil, nil
	} else if err != nil { //Обрабатываем кейс когда мы действительно получили ошибку
		return nil, err
	}
	var incidents []*domain.Incident //Создаем переменную куда будем запиысвать результат
	err = json.Unmarshal(result, &incidents)
	if err != nil { //Обрабатываем ошибку анмаршалинга
		return nil, err
	}
	return incidents, nil //Возвращаем слайс с полученным результатом
}
