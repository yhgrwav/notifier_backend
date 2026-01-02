package repository

import (
	"RedCollar/internal/config"
	"RedCollar/internal/domain"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type IncidentRepository interface {
	Create(ctx context.Context, incident *domain.Incident) (string, error)
	GetByID(ctx context.Context, id string) (*domain.Incident, error)
	Get(ctx context.Context, lat float64, long float64, limit, offset int) ([]*domain.Incident, error)
	Update(ctx context.Context, incident *domain.Incident) error
	Delete(ctx context.Context, id string) error

	Close()
}

type PostgresStorage struct {
	conn *pgxpool.Pool
}

func NewPostgresConnection(ctx context.Context, cfg config.Config) (*PostgresStorage, error) {
	dsn := cfg.PostgresDSN
	if dsn == "" {
		return nil, fmt.Errorf("Строка подключения к БД не установлена")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("Ошибка: не удалось установить соединения с БД: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("Не удалось получить ответ от БД: %w", err)
	}

	return &PostgresStorage{conn: pool}, nil
}

func (r *PostgresStorage) Close() {
	r.conn.Close()
}

func (r *PostgresStorage) Create(ctx context.Context, incident *domain.Incident) (string, error) {
	var id string
	query := `
        INSERT INTO incidents (title, description, lat, lon, radius, is_active, created_at) 
        VALUES ($1, $2, $3, $4, $5, $6, $7) 
        RETURNING id`

	err := r.conn.QueryRow(ctx, query,
		incident.Title, incident.Description, incident.Latitude, incident.Longitude, incident.RadiusMeters, incident.IsActive, incident.CreatedAt,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("Ошибка создания записи: %w", err)
	}
	return id, nil
}

func (r *PostgresStorage) GetByID(ctx context.Context, id string) (*domain.Incident, error) {
	var incident domain.Incident
	query := `SELECT id, title, description, lat, lon, radius, is_active, created_at FROM incidents WHERE id = $1`

	err := r.conn.QueryRow(ctx, query, id).Scan(
		&incident.ID, &incident.Title, &incident.Description, &incident.Latitude, &incident.Longitude, &incident.RadiusMeters, &incident.IsActive, &incident.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("Ошибка получения по ID: %w", err)
	}
	return &incident, nil
}

func (r *PostgresStorage) Update(ctx context.Context, incident *domain.Incident) error {
	query := `UPDATE incidents SET title=$1, description=$2, lat=$3, lon=$4, radius=$5, is_active=$6 WHERE id=$7`
	_, err := r.conn.Exec(ctx, query, incident.Title, incident.Description, incident.Latitude, incident.Longitude, incident.RadiusMeters, incident.IsActive, incident.ID)
	if err != nil {
		return fmt.Errorf("Ошибка обновления записи: %w", err)
	}
	return nil
}

func (r *PostgresStorage) Delete(ctx context.Context, id string) error {
	query := `UPDATE incidents SET is_active = false WHERE id = $1`
	_, err := r.conn.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("Ошибка удаления: %w", err)
	}
	return nil
}
func (r *PostgresStorage) Get(ctx context.Context, lat float64, long float64, limit, offset int) ([]*domain.Incident, error) {
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
        ) <= radius 
        AND is_active = true
        LIMIT $3 OFFSET $4`
	rows, err := r.conn.Query(ctx, query, lat, long, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("Ошибка получения записи: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var i domain.Incident
		err = rows.Scan(&i.ID, &i.Title, &i.Description, &i.Latitude, &i.Longitude, &i.RadiusMeters, &i.IsActive, &i.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("Ошибка получения записи: %w", err)
		}
		incidents = append(incidents, &i)
	}
	return incidents, nil
}
