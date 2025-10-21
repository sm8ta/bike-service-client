package ports

import (
	"context"
	"webike_bike_microservice_nikita/internal/core/domain"

	"github.com/google/uuid"
)

type BikeRepository interface {
	CreateBike(ctx context.Context, bike *domain.Bike) (*domain.Bike, error)
	GetBikeByID(ctx context.Context, bike_id uuid.UUID) (*domain.Bike, error)
	GetBikesByUserID(ctx context.Context, user_id uuid.UUID) ([]*domain.Bike, error)
	UpdateBike(ctx context.Context, bike *domain.Bike) (*domain.Bike, error)
	DeleteBike(ctx context.Context, bike_id uuid.UUID) error
}
type BikeService interface {
	CreateBike(ctx context.Context, bike *domain.Bike) (*domain.Bike, error)
	GetBikeByID(ctx context.Context, bike_id string) (*domain.Bike, error)
	GetBikesByUserID(ctx context.Context, user_id string) ([]*domain.Bike, error)
	UpdateBike(ctx context.Context, bike *domain.Bike) (*domain.Bike, error)
	DeleteBike(ctx context.Context, bike_id string) error
}
