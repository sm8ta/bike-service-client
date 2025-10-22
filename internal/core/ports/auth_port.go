package ports

import "github.com/sm8ta/webike_bike_microservice_nikita/internal/core/domain"

type TokenService interface {
	VerifyToken(token string) (*domain.TokenPayload, error)
}
