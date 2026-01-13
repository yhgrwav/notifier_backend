package repository

import (
	"RedCollar/internal/config"
	"RedCollar/internal/domain"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IncidentRepository interface {
	Create(ctx context.Context, incident *domain.Incident) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Incident, error)
	Get(ctx context.Context, lat float64, long float64, limit, offset int, extraRadius float64) ([]*domain.Incident, error)
	Update(ctx context.Context, incident *domain.Incident) error
	Delete(ctx context.Context, id uuid.UUID) error
	SaveCheck(ctx context.Context, userID string, lat, lon float64, incidentIDs []uuid.UUID) error
	GetStats(ctx context.Context, minutes int) ([]domain.StatisticResponse, error)
	Close()
}

type PostgresStorage struct {
	conn *pgxpool.Pool
}

func NewPostgresConnection(ctx context.Context, cfg config.Config) (*PostgresStorage, error) {
	dsn := cfg.PostgresDSN
	if dsn == "" {
		return nil, fmt.Errorf("строка подключения к базе данных не установлена")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания пула соединений с базой данных: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ошибка проверки соединения с базой данных: %w", err)
	}

	return &PostgresStorage{conn: pool}, nil
}

func (r *PostgresStorage) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
}

func (r *PostgresStorage) Create(ctx context.Context, incident *domain.Incident) (uuid.UUID, error) {
	if r.conn == nil {
		return uuid.Nil, fmt.Errorf("подключение к базе данных не инициализировано")
	}

	var id uuid.UUID
	query := `
        INSERT INTO incidents (title, description, lat, lon, radius, is_active, created_at) 
        VALUES ($1, $2, $3, $4, $5, $6, $7) 
        RETURNING id`

	err := r.conn.QueryRow(ctx, query,
		incident.Title, incident.Description, incident.Latitude, incident.Longitude, incident.RadiusMeters, incident.IsActive, incident.CreatedAt,
	).Scan(&id)

	if err != nil {
		return uuid.Nil, fmt.Errorf("ошибка создания записи в базе данных: %w", err)
	}
	return id, nil
}

func (r *PostgresStorage) GetByID(ctx context.Context, id uuid.UUID) (*domain.Incident, error) {
	if r.conn == nil {
		return nil, fmt.Errorf("подключение к базе данных не инициализировано")
	}

	var incident domain.Incident
	query := `SELECT id, title, description, lat, lon, radius, is_active, created_at FROM incidents WHERE id = $1`

	err := r.conn.QueryRow(ctx, query, id).Scan(
		&incident.ID, &incident.Title, &incident.Description, &incident.Latitude, &incident.Longitude, &incident.RadiusMeters, &incident.IsActive, &incident.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("инцидент с ID %s не найден: %w", id.String(), err)
		}
		return nil, fmt.Errorf("ошибка получения записи по ID из базы данных: %w", err)
	}
	return &incident, nil
}

func (r *PostgresStorage) Update(ctx context.Context, incident *domain.Incident) error {
	if r.conn == nil {
		return fmt.Errorf("подключение к базе данных не инициализировано")
	}

	query := `UPDATE incidents SET title=$1, description=$2, lat=$3, lon=$4, radius=$5, is_active=$6 WHERE id=$7`
	_, err := r.conn.Exec(ctx, query, incident.Title, incident.Description, incident.Latitude, incident.Longitude, incident.RadiusMeters, incident.IsActive, incident.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления записи в базе данных: %w", err)
	}
	return nil
}

func (r *PostgresStorage) Delete(ctx context.Context, id uuid.UUID) error {
	if r.conn == nil {
		return fmt.Errorf("подключение к базе данных не инициализировано")
	}

	query := `UPDATE incidents SET is_active = false WHERE id = $1`
	_, err := r.conn.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления записи в базе данных: %w", err)
	}
	return nil
}

// Метод отвечает за то, чтобы относительно точки(полученной от пользователя или оператора) найти список ицидентов
// отсортированный по удалению
func (r *PostgresStorage) Get(ctx context.Context, lat float64, long float64, limit, offset int, extraRadius float64) ([]*domain.Incident, error) {
	if r.conn == nil {
		return nil, fmt.Errorf("подключение к базе данных не инициализировано")
	}

	incidents := make([]*domain.Incident, 0)
	//Запрос с формулой Гаверсинуса, которая позволяет рассчитать расстояние между двумя точками на земле
	//Логика: если точка(координаты пользователя) находятся в радиусе инцидента - инцидент попадает в слайс инцидентов
	//в которых сейчас находится пользователь и для инцидента в статистику записывается конкретный юзер (требования условия)
	query := ` 
    SELECT id, title, description, lat, lon, radius, is_active, created_at 
    FROM incidents 
    WHERE (
        6371000 * acos(
            cos(radians($1)) * cos(radians(lat)) * cos(radians(lon) - radians($2)) + 
            sin(radians($1)) * sin(radians(lat))
        )
    ) <= (radius + $5) 
    AND is_active = true
    LIMIT $3 OFFSET $4`
	rows, err := r.conn.Query(ctx, query, lat, long, limit, offset, extraRadius)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса к базе данных: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var i domain.Incident
		err = rows.Scan(&i.ID, &i.Title, &i.Description, &i.Latitude, &i.Longitude, &i.RadiusMeters, &i.IsActive, &i.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения данных из результата запроса: %w", err)
		}
		incidents = append(incidents, &i)
	}

	// Проверяем, не было ли ошибок во время итерации по строкам
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при обработке результатов запроса: %w", err)
	}

	return incidents, nil
}

// SaveCheck реализовывает условия пункта №3 ТЗ - сохранить факт проверки в БД
func (r *PostgresStorage) SaveCheck(ctx context.Context, userID string, lat, lon float64, incidentIDs []uuid.UUID) error {
	if r.conn == nil {
		return fmt.Errorf("подключение к базе данных не инициализировано")
	}
	query := ` INSERT INTO location_checks (user_id, latitude, longitude, incident_ids) 
        VALUES ($1, $2, $3, $4)`

	_, err := r.conn.Exec(ctx, query, userID, lat, lon, incidentIDs)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении лога в БД: %w", err)
	}
	return nil
}

// GetStats отвечает за то, чтобы отдавать user_count(уникальные user_id за N минут по условию) для запрашиваемого инцидента
func (r *PostgresStorage) GetStats(ctx context.Context, minutes int) ([]domain.StatisticResponse, error) {
	if r.conn == nil {
		return nil, fmt.Errorf("подключение к базе данных не инициализировано")
	}
	//запрашиваем список из двух таблиц в формате инцидент - кол-во уникальных юзеров за указанный период времени в минутах
	query := `
        SELECT incident_id, COUNT(DISTINCT user_id)
        FROM location_checks
        WHERE incident_id IS NOT NULL 
          AND checked_at >= NOW() - (interval '1 minute' * $1)
        GROUP BY incident_id`

	rows, err := r.conn.Query(ctx, query, minutes)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения статистики для инцидента: %w", err)
	}
	defer rows.Close()

	var stats []domain.StatisticResponse
	for rows.Next() {
		//В каждой итерации создаем локальную переменную в которую записываем результат поиска
		//и либо возвращаем ошибку, либо записываем полученный результат в слайс stats
		var s domain.StatisticResponse
		if err := rows.Scan(&s.IncidentID, &s.UserCount); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}
