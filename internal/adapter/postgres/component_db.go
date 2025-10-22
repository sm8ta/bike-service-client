package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/domain"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ComponentRepository struct {
	db *sql.DB
}

func NewComponentRepository(db *sql.DB) *ComponentRepository {
	return &ComponentRepository{db: db}
}

func (r *ComponentRepository) CreateComponent(ctx context.Context, component *domain.Component) (*domain.Component, error) {
	query := `INSERT INTO components (id, bike_id, name, brand, model, installed_at, installed_mileage, max_mileage)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		component.ID,
		component.BikeID,
		component.Name,
		component.Brand,
		component.Model,
		component.InstalledAt,
		component.InstalledMileage,
		component.MaxMileage,
	).Scan(
		&component.ID,
		&component.CreatedAt,
		&component.UpdatedAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23502":
				return nil, fmt.Errorf("required field is missing")
			case "23503":
				return nil, fmt.Errorf("bike does not exist")
			default:
				return nil, err
			}
		}
		return nil, err
	}

	return component, nil
}

func (r *ComponentRepository) GetComponentByID(ctx context.Context, componentID uuid.UUID) (*domain.Component, error) {
	query := `
		SELECT id, bike_id, name, brand, model, installed_at, installed_mileage, max_mileage, created_at, updated_at
		FROM components
		WHERE id = $1
	`

	var component domain.Component
	err := r.db.QueryRowContext(ctx, query, componentID).Scan(
		&component.ID,
		&component.BikeID,
		&component.Name,
		&component.Brand,
		&component.Model,
		&component.InstalledAt,
		&component.InstalledMileage,
		&component.MaxMileage,
		&component.CreatedAt,
		&component.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("component not found")
		}
		return nil, fmt.Errorf("failed to get component: %w", err)
	}

	return &component, nil
}

func (r *ComponentRepository) GetComponentsByBikeID(ctx context.Context, bike_id uuid.UUID) ([]*domain.Component, error) {
	query := `SELECT id, bike_id, name, brand, model, installed_at, installed_mileage, max_mileage, created_at, updated_at
		FROM components WHERE bike_id = $1
		ORDER BY installed_at DESC`

	rows, err := r.db.QueryContext(ctx, query, bike_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var components []*domain.Component

	for rows.Next() {
		component := &domain.Component{}
		err := rows.Scan(
			&component.ID,
			&component.BikeID,
			&component.Name,
			&component.Brand,
			&component.Model,
			&component.InstalledAt,
			&component.InstalledMileage,
			&component.MaxMileage,
			&component.CreatedAt,
			&component.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		components = append(components, component)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return components, nil
}

func (r *ComponentRepository) UpdateComponent(ctx context.Context, component *domain.Component) (*domain.Component, error) {
	query := `UPDATE components
		SET 
			name = COALESCE(NULLIF($1, ''), name),
			brand = COALESCE(NULLIF($2, ''), brand),
			model = COALESCE(NULLIF($3, ''), model),
			installed_at = COALESCE(NULLIF($4, '0001-01-01 00:00:00+00'::timestamp), installed_at),
			installed_mileage = COALESCE(NULLIF($5, 0), installed_mileage),
			max_mileage = COALESCE(NULLIF($6, 0), max_mileage),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $7
		RETURNING id, bike_id, name, brand, model, installed_at, installed_mileage, max_mileage, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		component.Name,
		component.Brand,
		component.Model,
		component.InstalledAt,
		component.InstalledMileage,
		component.MaxMileage,
		component.ID,
	).Scan(
		&component.ID,
		&component.BikeID,
		&component.Name,
		&component.Brand,
		&component.Model,
		&component.InstalledAt,
		&component.InstalledMileage,
		&component.MaxMileage,
		&component.CreatedAt,
		&component.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("component not found")
		}
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23502" {
			return nil, fmt.Errorf("required field is missing")
		}
		return nil, fmt.Errorf("error updating component: %w", err)
	}

	return component, nil
}

func (r *ComponentRepository) DeleteComponent(ctx context.Context, component_id uuid.UUID) error {
	query := `DELETE FROM components WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, component_id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("component not found")
	}

	return nil
}
