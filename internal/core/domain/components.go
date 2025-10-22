package domain

import (
	"time"

	"github.com/google/uuid"
)

type ComponentName string

const (
	Handlebars ComponentName = "handlebars"
	Frame      ComponentName = "frame"
	Wheels     ComponentName = "wheels"
)

type Component struct {
	ID               uuid.UUID     `json:"id"`
	BikeID           uuid.UUID     `json:"bike_id" validate:"required"`
	Name             ComponentName `json:"name" validate:"required"`
	Brand            string        `json:"brand,omitempty" validate:"max=100"`
	Model            string        `json:"model,omitempty" validate:"max=100"`
	InstalledAt      time.Time     `json:"installed_at" validate:"required"`
	InstalledMileage int           `json:"installed_mileage" validate:"min=0"`
	MaxMileage       int           `json:"max_mileage" validate:"required,min=1,max=1000000"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

func (c *Component) CurrentMileage(bikeMileage int) int {
	return bikeMileage - c.InstalledMileage
}

func (c *Component) NeedsReplacement(bikeMileage int) bool {
	return c.CurrentMileage(bikeMileage) >= c.MaxMileage
}
