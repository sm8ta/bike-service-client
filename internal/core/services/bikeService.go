package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/domain"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/ports"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type BikeService struct {
	bikeRepo      ports.BikeRepository
	componentRepo ports.ComponentRepository
	logger        ports.LoggerPort
	validate      *validator.Validate
	cache         ports.CachePort
}

func NewBikeService(
	bikeRepo ports.BikeRepository,
	componentRepo ports.ComponentRepository,
	logger ports.LoggerPort,
	validate *validator.Validate,
	cache ports.CachePort,
) *BikeService {
	return &BikeService{
		bikeRepo:      bikeRepo,
		componentRepo: componentRepo,
		logger:        logger,
		validate:      validate,
		cache:         cache,
	}
}

func (s *BikeService) CreateBike(ctx context.Context, bike *domain.Bike) (*domain.Bike, error) {
	if err := s.validate.Struct(bike); err != nil {
		s.logger.Error("Bike validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("validation error: %w", err)
	}

	if bike.BikeID == uuid.Nil {
		bike.BikeID = uuid.New()
	}

	createdBike, err := s.bikeRepo.CreateBike(ctx, bike)
	if err != nil {
		s.logger.Error("Failed to create bike", map[string]interface{}{
			"error":   err.Error(),
			"user_id": bike.UserID,
		})
		return nil, err
	}

	s.logger.Info("Bike created successfully", map[string]interface{}{
		"bike_id": createdBike.BikeID,
		"user_id": createdBike.UserID,
	})

	return createdBike, nil
}

func (s *BikeService) GetBikeByID(ctx context.Context, bikeID string) (*domain.Bike, error) {
	bikeUUID, err := uuid.Parse(bikeID)
	if err != nil {
		s.logger.Error("Invalid UUID format", map[string]interface{}{
			"bike_id": bikeID,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("invalid bike ID: %w", err)
	}

	cacheKey := fmt.Sprintf("bike:%s", bikeID)
	cachedData, err := s.cache.Get(cacheKey)
	if err == nil {
		var cachedBike domain.Bike
		if err := json.Unmarshal(cachedData, &cachedBike); err == nil {
			s.logger.Info("Bike found in cache", map[string]interface{}{
				"bike_id": bikeID,
			})
			return &cachedBike, nil
		}
	}

	bike, err := s.bikeRepo.GetBikeByID(ctx, bikeUUID)
	if err != nil {
		s.logger.Error("Failed to get bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
		return nil, err
	}

	bikeData, err := json.Marshal(bike)
	if err != nil {
		s.logger.Warn("Failed to marshal bike for cache", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
	} else {
		if err := s.cache.Set(cacheKey, bikeData, 15*time.Minute); err != nil {
			s.logger.Warn("Failed to cache bike", map[string]interface{}{
				"error":   err.Error(),
				"bike_id": bikeID,
			})
		}
	}

	return bike, nil
}

func (s *BikeService) GetBikesByUserID(ctx context.Context, userID string) ([]*domain.Bike, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		s.logger.Error("Invalid UUID format", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	bikes, err := s.bikeRepo.GetBikesByUserID(ctx, userUUID)
	if err != nil {
		s.logger.Error("Failed to get bikes", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
		return nil, err
	}

	s.logger.Info("Retrieved bikes for user", map[string]interface{}{
		"user_id":     userID,
		"bikes_count": len(bikes),
	})

	return bikes, nil
}

func (s *BikeService) UpdateBike(ctx context.Context, bike *domain.Bike) (*domain.Bike, error) {
	if err := s.validate.Struct(bike); err != nil {
		s.logger.Error("Bike validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("validation error: %w", err)
	}

	updatedBike, err := s.bikeRepo.UpdateBike(ctx, bike)
	if err != nil {
		s.logger.Error("Failed to update bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bike.BikeID,
		})
		return nil, err
	}

	cacheKey := fmt.Sprintf("bike:%s", bike.BikeID.String())
	if err := s.cache.Delete(cacheKey); err != nil {
		s.logger.Warn("Failed to invalidate bike cache", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bike.BikeID.String(),
		})
	}

	s.logger.Info("Bike updated successfully", map[string]interface{}{
		"bike_id": bike.BikeID,
	})

	return updatedBike, nil
}

func (s *BikeService) DeleteBike(ctx context.Context, bikeID string) error {
	bikeUUID, err := uuid.Parse(bikeID)
	if err != nil {
		s.logger.Error("Invalid UUID format", map[string]interface{}{
			"bike_id": bikeID,
			"error":   err.Error(),
		})
		return fmt.Errorf("invalid bike ID: %w", err)
	}

	err = s.bikeRepo.DeleteBike(ctx, bikeUUID)
	if err != nil {
		s.logger.Error("Failed to delete bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
		return err
	}

	cacheKey := fmt.Sprintf("bike:%s", bikeID)
	if err := s.cache.Delete(cacheKey); err != nil {
		s.logger.Warn("Failed to invalidate bike cache", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
	}

	s.logger.Info("Bike deleted successfully", map[string]interface{}{
		"bike_id": bikeID,
	})

	return nil
}

func (s *BikeService) GetBikeWithComponents(ctx context.Context, bikeID string) (*domain.Bike, error) {
	bikeUUID, err := uuid.Parse(bikeID)
	if err != nil {
		s.logger.Error("Invalid UUID format", map[string]interface{}{
			"bike_id": bikeID,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("invalid bike ID: %w", err)
	}

	bike, err := s.bikeRepo.GetBikeByID(ctx, bikeUUID)
	if err != nil {
		s.logger.Error("Failed to get bike", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
		return nil, err
	}

	components, err := s.componentRepo.GetComponentsByBikeID(ctx, bikeUUID)
	if err != nil {
		s.logger.Warn("Failed to get components", map[string]interface{}{
			"error":   err.Error(),
			"bike_id": bikeID,
		})
		components = []*domain.Component{}
	}

	bike.Components = components

	s.logger.Info("Retrieved bike with components", map[string]interface{}{
		"bike_id":          bikeID,
		"components_count": len(components),
	})

	return bike, nil
}
