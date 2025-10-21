package ports

import "webike_bike_microservice_nikita/internal/core/domain"

type TokenService interface {
	VerifyToken(token string) (*domain.TokenPayload, error)
}
