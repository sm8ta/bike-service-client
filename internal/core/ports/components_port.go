package ports

import (
	"context"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/domain"

	"github.com/google/uuid"
)

type ComponentRepository interface {
	CreateComponent(ctx context.Context, component *domain.Component) (*domain.Component, error)
	GetComponentByID(ctx context.Context, componentID uuid.UUID) (*domain.Component, error)
	GetComponentsByBikeID(ctx context.Context, bikeID uuid.UUID) ([]*domain.Component, error)
	UpdateComponent(ctx context.Context, component *domain.Component) (*domain.Component, error)
	DeleteComponent(ctx context.Context, componentID uuid.UUID) error
}
