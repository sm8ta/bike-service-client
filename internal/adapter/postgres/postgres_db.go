package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/domain"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type BikeRepository struct {
	db *sql.DB
}

func NewBikeRepository(db *sql.DB) *BikeRepository {
	return &BikeRepository{
		db,
	}
}

func (r *BikeRepository) CreateBike(ctx context.Context, bike *domain.Bike) (*domain.Bike, error) {
	query := `INSERT INTO bikes (user_id, bike_id, bike_name, type, model, year, mileage)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING bike_id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query, bike.UserID, bike.BikeID, bike.BikeName, bike.Type, bike.Model, bike.Year, bike.Mileage).Scan(
		&bike.BikeID,
		&bike.CreatedAt,
		&bike.UpdatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23502":
				return nil, fmt.Errorf("required field is missing")
			case "23503":
				return nil, fmt.Errorf("user does not exist")
			default:
				return nil, err
			}
		}
		return nil, err
	}
	return bike, nil
}

func (r *BikeRepository) GetBikeByID(ctx context.Context, bike_id uuid.UUID) (*domain.Bike, error) {
	query := `SELECT user_id, bike_id, bike_name, type, model, year, mileage, created_at, updated_at
              FROM bikes WHERE bike_id = $1`

	bike := &domain.Bike{}
	err := r.db.QueryRowContext(ctx, query, bike_id).Scan(
		&bike.UserID,
		&bike.BikeID,
		&bike.BikeName,
		&bike.Type,
		&bike.Model,
		&bike.Year,
		&bike.Mileage,
		&bike.CreatedAt,
		&bike.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("bike not found")
	}
	if err != nil {
		return nil, err
	}

	return bike, nil
}

func (r *BikeRepository) GetBikesByUserID(ctx context.Context, user_id uuid.UUID) ([]*domain.Bike, error) {
	query := `SELECT user_id, bike_id, bike_name, type, model, year, mileage, created_at, updated_at
              FROM bikes WHERE user_id = $1`

	rows, err := r.db.QueryContext(ctx, query, user_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bikes []*domain.Bike

	for rows.Next() {
		bike := &domain.Bike{}
		err := rows.Scan(
			&bike.UserID,
			&bike.BikeID,
			&bike.BikeName,
			&bike.Type,
			&bike.Model,
			&bike.Year,
			&bike.Mileage,
			&bike.CreatedAt,
			&bike.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		bikes = append(bikes, bike)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return bikes, nil
}
func (r *BikeRepository) DeleteBike(ctx context.Context, bike_id uuid.UUID) error {
	query := `DELETE FROM bikes WHERE bike_id = $1`

	result, err := r.db.ExecContext(ctx, query, bike_id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("Bike not found")
	}

	return nil
}

func (r *BikeRepository) UpdateBike(ctx context.Context, bike *domain.Bike) (*domain.Bike, error) {
	query := `UPDATE bikes
		SET 
			bike_name = COALESCE(NULLIF($1, ''), bike_name),
			type = COALESCE(NULLIF($2, ''), type),
			model = COALESCE(NULLIF($3, ''), model),
			year = COALESCE(NULLIF($4, 0), year),
			mileage = COALESCE(NULLIF($5, 0), mileage),
			updated_at = CURRENT_TIMESTAMP
		WHERE bike_id = $6
		RETURNING user_id, bike_id, bike_name, type, model, year, mileage, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		bike.BikeName,
		bike.Type,
		bike.Model,
		bike.Year,
		bike.Mileage,
		bike.BikeID,
	).Scan(
		&bike.UserID,
		&bike.BikeID,
		&bike.BikeName,
		&bike.Type,
		&bike.Model,
		&bike.Year,
		&bike.Mileage,
		&bike.CreatedAt,
		&bike.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("bike not found")
		}
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23502" {
			return nil, fmt.Errorf("required field is missing")
		}
		return nil, fmt.Errorf("error updating bike: %w", err)
	}

	return bike, nil
}
