package http

import (
	"errors"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/domain"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/core/ports"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTTokenService struct {
	secretKey []byte
	logger    ports.LoggerPort
}

func NewJWTTokenService(secretKey string, logger ports.LoggerPort) *JWTTokenService {
	return &JWTTokenService{
		secretKey: []byte(secretKey),
		logger:    logger,
	}
}

// проверка жвт
func (j *JWTTokenService) VerifyToken(token string) (*domain.TokenPayload, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return j.secretKey, nil
	})
	if err != nil {
		j.logger.Error("Failed to parse jwt", map[string]interface{}{
			"error":  err.Error(),
			"method": "VerifyToken",
		})
		return nil, err
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		j.logger.Error("Failed claims from token", map[string]interface{}{
			"method": "VerifyToken",
		})
		return nil, errors.New("failed to verify")
	}

	idStr, ok := claims["id"].(string)
	if !ok {
		return nil, errors.New("invalid id convert")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, errors.New("invalid parse id")
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, errors.New("invalid user_id claims")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid parse user_id")
	}

	roleClaimed, ok := claims["role"].(string)
	if !ok {
		return nil, errors.New("invalid role")
	}

	role := domain.UserRole(roleClaimed)
	if role != domain.Admin && role != domain.AppUser {
		j.logger.Warn("Invalid role in token", map[string]interface{}{
			"role":   roleClaimed,
			"method": "VerifyToken",
		})
		return nil, errors.New("invalid role value")
	}

	payload := &domain.TokenPayload{
		ID:     id,
		UserID: userID,
		Role:   role,
	}

	return payload, nil
}
