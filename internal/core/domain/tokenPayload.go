package domain

import (
	"github.com/google/uuid"
)

type UserRole string

const (
	Admin   UserRole = "admin"
	AppUser UserRole = "appuser"
)

type TokenPayload struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Role   UserRole
}
