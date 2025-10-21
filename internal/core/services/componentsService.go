package services

import (
	"context"
	"fmt"
	"webike_bike_microservice_nikita/internal/core/domain"
	"webike_bike_microservice_nikita/internal/core/ports"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type ComponentService struct {
	componentRepo ports.ComponentRepository
	logger        ports.LoggerPort
	validate      *validator.Validate
	cache         ports.CachePort
}

func NewComponentService(
	componentRepo ports.ComponentRepository,
	logger ports.LoggerPort,
	validate *validator.Validate,
	cache ports.CachePort,
) *ComponentService {
	return &ComponentService{
		componentRepo: componentRepo,
		logger:        logger,
		validate:      validate,
		cache:         cache,
	}
}

func (s *ComponentService) CreateComponent(ctx context.Context, component *domain.Component) (*domain.Component, error) {
	if err := s.validate.Struct(component); err != nil {
		s.logger.Error("Component validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("validation error: %w", err)
	}

	if component.ID == uuid.Nil {
		component.ID = uuid.New()
	}

	createdComponent, err := s.componentRepo.CreateComponent(ctx, component)
	if err != nil {
		s.logger.Error("Failed to create component", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": component.BikeID,
		})
		return nil, err
	}

	cacheKey := fmt.Sprintf("bike:%s", component.BikeID.String())
	if err := s.cache.Delete(cacheKey); err != nil {
		s.logger.Warn("Failed to invalidate bike cache", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": component.BikeID.String(),
		})
	}

	s.logger.Info("Component created successfully", map[string]interface{}{
		"component_id": createdComponent.ID,
		"bike_id":      createdComponent.BikeID,
		"name":         createdComponent.Name,
	})

	return createdComponent, nil
}

func (s *ComponentService) GetComponentByID(ctx context.Context, componentID string) (*domain.Component, error) {
	componentUUID, err := uuid.Parse(componentID)
	if err != nil {
		s.logger.Error("Invalid UUID format", map[string]interface{}{
			"component_id": componentID,
			"error":        err.Error(),
		})
		return nil, fmt.Errorf("invalid component ID: %w", err)
	}

	component, err := s.componentRepo.GetComponentByID(ctx, componentUUID)
	if err != nil {
		s.logger.Error("Failed to get component", map[string]interface{}{
			"error":        err.Error(),
			"component_id": componentID,
		})
		return nil, err
	}

	s.logger.Info("Retrieved component", map[string]interface{}{
		"component_id": componentID,
		"bike_id":      component.BikeID,
	})

	return component, nil
}

func (s *ComponentService) GetComponentsByBikeID(ctx context.Context, bikeID string) ([]*domain.Component, error) {
	bikeUUID, err := uuid.Parse(bikeID)
	if err != nil {
		s.logger.Error("Invalid UUID format", map[string]interface{}{
			"bike_id": bikeID,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("invalid bike ID: %w", err)
	}

	components, err := s.componentRepo.GetComponentsByBikeID(ctx, bikeUUID)
	if err != nil {
		s.logger.Error("Failed to get components", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
		return nil, err
	}

	s.logger.Info("Retrieved components for bike", map[string]interface{}{
		"bike_id":          bikeID,
		"components_count": len(components),
	})

	return components, nil
}

func (s *ComponentService) UpdateComponent(ctx context.Context, component *domain.Component) (*domain.Component, error) {
	if err := s.validate.Struct(component); err != nil {
		s.logger.Error("Component validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("validation error: %w", err)
	}

	updatedComponent, err := s.componentRepo.UpdateComponent(ctx, component)
	if err != nil {
		s.logger.Error("Failed to update component", map[string]interface{}{
			"error":        err.Error(),
			"component_id": component.ID,
		})
		return nil, err
	}

	cacheKey := fmt.Sprintf("bike:%s", component.BikeID.String())
	if err := s.cache.Delete(cacheKey); err != nil {
		s.logger.Warn("Failed to invalidate bike cache", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": component.BikeID.String(),
		})
	}

	s.logger.Info("Component updated successfully", map[string]interface{}{
		"component_id": component.ID,
	})

	return updatedComponent, nil
}

func (s *ComponentService) DeleteComponent(ctx context.Context, componentID string) error {
	componentUUID, err := uuid.Parse(componentID)
	if err != nil {
		s.logger.Error("Invalid UUID format", map[string]interface{}{
			"component_id": componentID,
			"error":        err.Error(),
		})
		return fmt.Errorf("invalid component ID: %w", err)
	}

	component, err := s.componentRepo.GetComponentByID(ctx, componentUUID)
	if err != nil {
		s.logger.Error("Failed to get component", map[string]interface{}{
			"error":        err.Error(),
			"component_id": componentID,
		})
		return err
	}

	err = s.componentRepo.DeleteComponent(ctx, componentUUID)
	if err != nil {
		s.logger.Error("Failed to delete component", map[string]interface{}{
			"error":        err.Error(),
			"component_id": componentID,
		})
		return err
	}

	cacheKey := fmt.Sprintf("bike:%s", component.BikeID.String())
	if err := s.cache.Delete(cacheKey); err != nil {
		s.logger.Warn("Failed to invalidate bike cache", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": component.BikeID.String(),
		})
	}

	s.logger.Info("Component deleted successfully", map[string]interface{}{
		"component_id": componentID,
	})

	return nil
}
