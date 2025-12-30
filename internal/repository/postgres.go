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
		return nil, fmt.Errorf("PostgresDSN is not set")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping failed: %w", err)
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
		return "", fmt.Errorf("db create: %w", err)
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
		return nil, fmt.Errorf("db get by id: %w", err)
	}
	return &incident, nil
}

func (r *PostgresStorage) Update(ctx context.Context, incident *domain.Incident) error {
	query := `UPDATE incidents SET title=$1, description=$2, lat=$3, lon=$4, radius=$5, is_active=$6 WHERE id=$7`
	_, err := r.conn.Exec(ctx, query, incident.Title, incident.Description, incident.Latitude, incident.Longitude, incident.RadiusMeters, incident.IsActive, incident.ID)
	if err != nil {
		return fmt.Errorf("db update: %w", err)
	}
	return nil
}

func (r *PostgresStorage) Delete(ctx context.Context, id string) error {
	query := `UPDATE incidents SET is_active = false WHERE id = $1`
	_, err := r.conn.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("db delete: %w", err)
	}
	return nil
}
